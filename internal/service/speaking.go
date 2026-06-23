package service

import (
	"context"
	"strings"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

// SpeakingService grades a recorded spoken answer via Gemini and serves
// random pronunciation-drill words. Grading is stateless — add persistence
// if speaking-attempt history is ever needed.
type SpeakingService interface {
	Grade(ctx context.Context, promptText, mimeType string, audio []byte) (*SpeakingResult, error)
	RandomWord(ctx context.Context) (*model.PronunciationWord, error)
	// PracticeWord returns the word for drilling, inserting it if it's new.
	PracticeWord(ctx context.Context, word string) (*model.PronunciationWord, error)
}

type speakingService struct {
	ai    *GeminiClient
	words repository.PronunciationWordRepository
}

func NewSpeakingService(ai *GeminiClient, words repository.PronunciationWordRepository) SpeakingService {
	return &speakingService{ai: ai, words: words}
}

func (s *speakingService) Grade(ctx context.Context, promptText, mimeType string, audio []byte) (*SpeakingResult, error) {
	return s.ai.GradeSpeaking(ctx, promptText, mimeType, audio)
}

func (s *speakingService) RandomWord(ctx context.Context) (*model.PronunciationWord, error) {
	return s.words.Random(ctx)
}

func (s *speakingService) PracticeWord(ctx context.Context, word string) (*model.PronunciationWord, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, apperror.BadRequest("thiếu từ cần luyện")
	}
	w, err := s.words.FindOrCreate(ctx, word)
	if err != nil {
		return nil, err
	}
	// New word with no seeded transcription yet — generate one via Gemini.
	// Best-effort: a failed/slow phonetic lookup shouldn't block practicing
	// the word itself, so swallow the error and leave it blank.
	if w.Phonetic == "" {
		if phonetic, perr := s.ai.TranscribePhonetic(ctx, w.Word); perr == nil && phonetic != "" {
			if serr := s.words.SetPhonetic(ctx, w.ID, phonetic); serr == nil {
				w.Phonetic = phonetic
			}
		}
	}
	return w, nil
}
