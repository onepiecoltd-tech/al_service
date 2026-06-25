package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type ExamService interface {
	List(ctx context.Context, limit, offset int) ([]model.Exam, int, error)
	Get(ctx context.Context, id uuid.UUID) (*model.Exam, error)
	Create(ctx context.Context, e *model.Exam) error
	Update(ctx context.Context, e *model.Exam) (*model.Exam, error)
	Delete(ctx context.Context, id uuid.UUID) error
	// MarkImporting flips an existing exam to the 'processing' state ahead of a
	// background (re-)extraction, returning its previous state so ExtractImport
	// can restore it once extraction finishes.
	MarkImporting(ctx context.Context, examID uuid.UUID) (string, error)
	// ExtractImport (re-)extracts questions for an existing exam in the
	// background, replacing its question bank and restoring it to restoreState on
	// success, or marking it 'failed' on error. Like ExtractUpload, the AI call
	// is slow, so this is meant to run off the request goroutine.
	ExtractImport(ctx context.Context, examID uuid.UUID, filename string, data []byte, restoreState string)
	Questions(ctx context.Context, examID uuid.UUID) ([]model.Question, error)
	// ListMine returns only the exams owned by the given user. lang filters by
	// exam language code; "" means all languages.
	ListMine(ctx context.Context, ownerID uuid.UUID, lang string, limit, offset int) ([]model.Exam, int, error)
	// ListBank returns the published admin question bank, for any user to
	// practice. lang filters by exam language code; "" means all languages.
	ListBank(ctx context.Context, lang string, limit, offset int) ([]model.Exam, int, error)
	// GetBank returns a bank exam only if it's published and ownerless, else NotFound.
	GetBank(ctx context.Context, examID uuid.UUID) (*model.Exam, error)
	// RandomBankQuestion returns one random question from the published bank,
	// for use as a random default speaking prompt.
	RandomBankQuestion(ctx context.Context) (*model.Question, error)
	// CreateUpload records a user-owned exam in the 'processing' state, before
	// its questions are extracted. language is the exam's target language code
	// (defaults to 'en'). Question extraction is slow (the AI call takes
	// minutes), so it runs separately via ExtractUpload rather than blocking the
	// upload request.
	CreateUpload(ctx context.Context, ownerID uuid.UUID, author, name, language, filename string) (*model.Exam, error)
	// ExtractUpload extracts questions from the uploaded file and attaches them
	// to an already-created exam, flipping it to 'published' on success or
	// 'failed' on error. Intended to run in the background, so it takes its own
	// context rather than the upload request's.
	ExtractUpload(ctx context.Context, examID uuid.UUID, filename string, data []byte)
	// GetOwned returns the exam only if it belongs to ownerID, else NotFound
	// (never leaks existence of another user's exam).
	GetOwned(ctx context.Context, examID, ownerID uuid.UUID) (*model.Exam, error)
	// AskStream answers a free-text question about the owner's exam, using its
	// extracted questions plus the persisted conversation as context, streaming
	// each text fragment to onChunk as it arrives, then appends both the
	// question and the full answer to that history.
	AskStream(ctx context.Context, examID, ownerID uuid.UUID, question string, onChunk func(chunk string)) error
	// ChatHistory returns the persisted Giải đề AI conversation for the exam.
	ChatHistory(ctx context.Context, examID, ownerID uuid.UUID) ([]model.ChatMessage, error)
}

type examService struct {
	repo      repository.ExamRepository
	questions repository.QuestionRepository
	chat      repository.ChatMessageRepository
	ai        *GeminiClient
}

func NewExamService(repo repository.ExamRepository, questions repository.QuestionRepository, chat repository.ChatMessageRepository, ai *GeminiClient) ExamService {
	return &examService{repo: repo, questions: questions, chat: chat, ai: ai}
}

func (s *examService) MarkImporting(ctx context.Context, examID uuid.UUID) (string, error) {
	exam, err := s.repo.Get(ctx, examID)
	if err != nil {
		return "", err
	}
	prior := exam.State
	if prior == "processing" {
		prior = "draft" // never restore back into the transient state
	}
	exam.State = "processing"
	if _, err := s.repo.Update(ctx, exam); err != nil {
		return "", err
	}
	return prior, nil
}

func (s *examService) ExtractImport(ctx context.Context, examID uuid.UUID, filename string, data []byte, restoreState string) {
	s.extractInto(ctx, examID, filename, data, restoreState)
}

func (s *examService) Questions(ctx context.Context, examID uuid.UUID) ([]model.Question, error) {
	return s.questions.ListByExam(ctx, examID)
}

func (s *examService) ListMine(ctx context.Context, ownerID uuid.UUID, lang string, limit, offset int) ([]model.Exam, int, error) {
	return s.repo.ListByOwner(ctx, ownerID, normalizeLanguage(lang), limit, offset)
}

func (s *examService) ListBank(ctx context.Context, lang string, limit, offset int) ([]model.Exam, int, error) {
	return s.repo.ListPublished(ctx, normalizeLanguage(lang), limit, offset)
}

func (s *examService) RandomBankQuestion(ctx context.Context) (*model.Question, error) {
	return s.questions.RandomFromBank(ctx)
}

func (s *examService) GetBank(ctx context.Context, examID uuid.UUID) (*model.Exam, error) {
	exam, err := s.repo.Get(ctx, examID)
	if err != nil {
		return nil, err
	}
	if exam.OwnerID != nil || exam.State != "published" {
		return nil, apperror.NotFound("exam not found")
	}
	return exam, nil
}

func (s *examService) CreateUpload(ctx context.Context, ownerID uuid.UUID, author, name, language, filename string) (*model.Exam, error) {
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(filename, fileExt(filename))
	}
	if strings.TrimSpace(name) == "" {
		return nil, apperror.BadRequest("thiếu tên đề")
	}
	lang := normalizeLanguage(language)
	if lang == "" {
		lang = "en"
	}
	exam := &model.Exam{
		Name:      name,
		Type:      "Tự tải lên",
		Language:  lang,
		Questions: 0,
		Author:    author,
		State:     "processing",
		OwnerID:   &ownerID,
	}
	if err := s.repo.Create(ctx, exam); err != nil {
		return nil, err
	}
	return exam, nil
}

func (s *examService) ExtractUpload(ctx context.Context, examID uuid.UUID, filename string, data []byte) {
	s.extractInto(ctx, examID, filename, data, "published")
}

// extractInto runs the AI extraction for an existing exam and replaces its
// question bank, flipping the exam to successState on success or 'failed' on
// error. Shared by the user upload (successState "published") and admin import
// (successState = the exam's prior, restored state).
func (s *examService) extractInto(ctx context.Context, examID uuid.UUID, filename string, data []byte, successState string) {
	exam, err := s.repo.Get(ctx, examID)
	if err != nil {
		return
	}
	markFailed := func() {
		exam.State = "failed"
		_, _ = s.repo.Update(ctx, exam)
	}

	qs, err := s.ai.ExtractQuestions(ctx, filename, data)
	if err != nil {
		markFailed()
		return
	}
	if err := s.questions.ReplaceForExam(ctx, examID, qs); err != nil {
		markFailed()
		return
	}
	exam.Questions = len(qs)
	exam.State = successState
	_, _ = s.repo.Update(ctx, exam)
}

func (s *examService) GetOwned(ctx context.Context, examID, ownerID uuid.UUID) (*model.Exam, error) {
	exam, err := s.repo.Get(ctx, examID)
	if err != nil {
		return nil, err
	}
	if exam.OwnerID == nil || *exam.OwnerID != ownerID {
		return nil, apperror.NotFound("exam not found")
	}
	return exam, nil
}

const maxAskContextQuestions = 60

func (s *examService) ChatHistory(ctx context.Context, examID, ownerID uuid.UUID) ([]model.ChatMessage, error) {
	if _, err := s.GetOwned(ctx, examID, ownerID); err != nil {
		return nil, err
	}
	return s.chat.ListByExam(ctx, examID)
}

func (s *examService) AskStream(ctx context.Context, examID, ownerID uuid.UUID, question string, onChunk func(chunk string)) error {
	question = strings.TrimSpace(question)
	if question == "" {
		return apperror.BadRequest("thiếu câu hỏi")
	}
	if _, err := s.GetOwned(ctx, examID, ownerID); err != nil {
		return err
	}
	qs, err := s.questions.ListByExam(ctx, examID)
	if err != nil {
		return err
	}
	prior, err := s.chat.ListByExam(ctx, examID)
	if err != nil {
		return err
	}

	var b strings.Builder
	for i, q := range qs {
		if i >= maxAskContextQuestions {
			break
		}
		fmt.Fprintf(&b, "%d. %s\n", q.Position, q.Prompt)
		if q.SampleAnswer != "" {
			fmt.Fprintf(&b, "   Đáp án mẫu: %s\n", q.SampleAnswer)
		}
	}

	history := make([]ChatTurn, len(prior))
	for i, m := range prior {
		history[i] = ChatTurn{Role: m.Role, Text: m.Text}
	}

	answer, err := s.ai.AskStream(ctx, b.String(), history, question, onChunk)
	if err != nil {
		return err
	}
	if err := s.chat.Insert(ctx, examID, "user", question); err != nil {
		return err
	}
	if err := s.chat.Insert(ctx, examID, "model", answer); err != nil {
		return err
	}
	return nil
}

func (s *examService) List(ctx context.Context, limit, offset int) ([]model.Exam, int, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *examService) Get(ctx context.Context, id uuid.UUID) (*model.Exam, error) {
	return s.repo.Get(ctx, id)
}

func (s *examService) Create(ctx context.Context, e *model.Exam) error {
	if err := validateExam(e); err != nil {
		return err
	}
	if e.State == "" {
		e.State = "draft"
	}
	return s.repo.Create(ctx, e)
}

func (s *examService) Update(ctx context.Context, e *model.Exam) (*model.Exam, error) {
	if err := validateExam(e); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, e)
}

func (s *examService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func validateExam(e *model.Exam) error {
	if strings.TrimSpace(e.Name) == "" {
		return apperror.BadRequest("name is required")
	}
	if strings.TrimSpace(e.Type) == "" {
		return apperror.BadRequest("type is required")
	}
	if lang := normalizeLanguage(e.Language); lang == "" {
		e.Language = "en"
	} else {
		e.Language = lang
	}
	return nil
}

// normalizeLanguage lower-cases and trims a language code, returning "" if it
// isn't a plausible code (2–8 ASCII letters/hyphen, e.g. "en", "zh", "pt-br").
// The set is intentionally open — callers default empty to "en" where needed.
func normalizeLanguage(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) < 2 || len(s) > 8 {
		return ""
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && r != '-' {
			return ""
		}
	}
	return s
}
