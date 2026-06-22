package server

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/craftbyte/learning_languages/services/docs"
	"github.com/craftbyte/learning_languages/services/internal/handler"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/repository"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)

	userRepo := repository.NewUserRepository(s.db)
	blogRepo := repository.NewBlogRepository(s.db)
	giftRepo := repository.NewGiftRepository(s.db)
	notificationRepo := repository.NewNotificationRepository(s.db)
	reportRepo := repository.NewReportRepository(s.db)
	settingRepo := repository.NewSettingRepository(s.db)
	examRepo := repository.NewExamRepository(s.db)
	questionRepo := repository.NewQuestionRepository(s.db)
	commentRepo := repository.NewCommentRepository(s.db)
	overviewRepo := repository.NewOverviewRepository(s.db)
	walletRepo := repository.NewWalletRepository(s.db)
	badgeRepo := repository.NewBadgeRepository(s.db)

	authService := service.NewAuthService(userRepo, s.cfg.JWTSecret, s.cfg.GoogleClientID)
	profileService := service.NewProfileService(userRepo)
	blogService := service.NewBlogService(blogRepo)
	leaderboardService := service.NewLeaderboardService(userRepo)
	friendService := service.NewFriendService(userRepo)
	giftService := service.NewGiftService(giftRepo)
	notificationService := service.NewNotificationService(notificationRepo)
	adminUserService := service.NewAdminUserService(userRepo)
	requireAuth := middleware.Auth(s.cfg.JWTSecret, adminUserService.IsActive)
	reportService := service.NewReportService(reportRepo)
	settingService := service.NewSettingService(settingRepo)
	aiClient := service.NewGeminiClient(s.cfg.GeminiAPIKey)
	examService := service.NewExamService(examRepo, questionRepo, aiClient)
	commentService := service.NewCommentService(commentRepo)
	overviewService := service.NewOverviewService(overviewRepo)
	walletService := service.NewWalletService(walletRepo, giftRepo)
	badgeService := service.NewBadgeService(badgeRepo)

	authHandler := handler.NewAuthHandler(authService, settingService)
	profileHandler := handler.NewProfileHandler(profileService)
	blogHandler := handler.NewBlogHandler(blogService, profileService, commentService)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardService)
	friendHandler := handler.NewFriendHandler(friendService)
	giftHandler := handler.NewGiftHandler(giftService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	adminUserHandler := handler.NewAdminUserHandler(adminUserService)
	adminReportHandler := handler.NewAdminReportHandler(reportService)
	adminSettingHandler := handler.NewAdminSettingHandler(settingService)
	statusHandler := handler.NewStatusHandler(settingService)
	adminExamHandler := handler.NewAdminExamHandler(examService, profileService)
	adminOverviewHandler := handler.NewAdminOverviewHandler(overviewService)
	walletHandler := handler.NewWalletHandler(walletService)
	badgeHandler := handler.NewBadgeHandler(badgeService)
	adminRevenueHandler := handler.NewAdminRevenueHandler(walletService)

	requireAdmin := middleware.AdminOnly(adminUserService.IsAdmin)

	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/google", authHandler.GoogleLogin)

	mux.Handle("GET /api/v1/me", requireAuth(http.HandlerFunc(profileHandler.Me)))
	mux.Handle("GET /api/v1/me/badges", requireAuth(http.HandlerFunc(badgeHandler.Me)))
	mux.Handle("GET /api/v1/me/prefs", requireAuth(http.HandlerFunc(profileHandler.GetPrefs)))
	mux.Handle("PUT /api/v1/me/prefs", requireAuth(http.HandlerFunc(profileHandler.SetPrefs)))
	mux.Handle("GET /api/v1/leaderboard", requireAuth(http.HandlerFunc(leaderboardHandler.List)))
	mux.Handle("GET /api/v1/friends", requireAuth(http.HandlerFunc(friendHandler.List)))
	mux.Handle("POST /api/v1/friends", requireAuth(http.HandlerFunc(friendHandler.Add)))
	mux.Handle("DELETE /api/v1/friends/{id}", requireAuth(http.HandlerFunc(friendHandler.Remove)))
	mux.Handle("GET /api/v1/users/search", requireAuth(http.HandlerFunc(friendHandler.Search)))

	mux.HandleFunc("GET /api/v1/status", statusHandler.Status)
	mux.HandleFunc("GET /api/v1/gifts", giftHandler.List)
	mux.HandleFunc("GET /api/v1/coin-packs", walletHandler.CoinPacks)

	mux.Handle("GET /api/v1/wallet/transactions", requireAuth(http.HandlerFunc(walletHandler.Transactions)))
	mux.Handle("POST /api/v1/wallet/topup", requireAuth(http.HandlerFunc(walletHandler.Topup)))
	mux.Handle("POST /api/v1/wallet/gift", requireAuth(http.HandlerFunc(walletHandler.Gift)))

	mux.Handle("GET /api/v1/notifications", requireAuth(http.HandlerFunc(notificationHandler.List)))
	mux.Handle("POST /api/v1/notifications/read", requireAuth(http.HandlerFunc(notificationHandler.MarkAllRead)))

	mux.Handle("GET /api/v1/admin/overview", requireAuth(requireAdmin(http.HandlerFunc(adminOverviewHandler.Get))))
	mux.Handle("GET /api/v1/admin/transactions", requireAuth(requireAdmin(http.HandlerFunc(adminRevenueHandler.Transactions))))
	mux.Handle("GET /api/v1/admin/revenue", requireAuth(requireAdmin(http.HandlerFunc(adminRevenueHandler.Revenue))))
	mux.Handle("POST /api/v1/admin/coin-packs", requireAuth(requireAdmin(http.HandlerFunc(adminRevenueHandler.CreatePack))))
	mux.Handle("PUT /api/v1/admin/coin-packs/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminRevenueHandler.UpdatePack))))
	mux.Handle("DELETE /api/v1/admin/coin-packs/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminRevenueHandler.DeletePack))))
	mux.Handle("GET /api/v1/admin/users", requireAuth(requireAdmin(http.HandlerFunc(adminUserHandler.List))))
	mux.Handle("POST /api/v1/admin/users", requireAuth(requireAdmin(http.HandlerFunc(adminUserHandler.Create))))
	mux.Handle("PUT /api/v1/admin/users/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminUserHandler.Update))))

	mux.Handle("GET /api/v1/admin/reports", requireAuth(requireAdmin(http.HandlerFunc(adminReportHandler.List))))
	mux.Handle("POST /api/v1/admin/reports/{id}/resolve", requireAuth(requireAdmin(http.HandlerFunc(adminReportHandler.Resolve))))

	mux.Handle("GET /api/v1/admin/settings", requireAuth(requireAdmin(http.HandlerFunc(adminSettingHandler.List))))
	mux.Handle("PUT /api/v1/admin/settings/{key}", requireAuth(requireAdmin(http.HandlerFunc(adminSettingHandler.Update))))

	mux.Handle("GET /api/v1/admin/exams", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.List))))
	mux.Handle("GET /api/v1/admin/exams/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Get))))
	mux.Handle("POST /api/v1/admin/exams", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Create))))
	mux.Handle("PUT /api/v1/admin/exams/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Update))))
	mux.Handle("DELETE /api/v1/admin/exams/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Delete))))
	mux.Handle("POST /api/v1/admin/exams/{id}/import", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Import))))
	mux.Handle("GET /api/v1/admin/exams/{id}/questions", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Questions))))

	mux.HandleFunc("GET /api/v1/blog", blogHandler.List)
	mux.HandleFunc("GET /api/v1/blog/{id}", blogHandler.Get)
	mux.HandleFunc("GET /api/v1/blog/{id}/comments", blogHandler.ListComments)
	mux.Handle("POST /api/v1/blog/{id}/comments", requireAuth(http.HandlerFunc(blogHandler.AddComment)))
	mux.Handle("POST /api/v1/blog", requireAuth(http.HandlerFunc(blogHandler.Create)))
	mux.Handle("PUT /api/v1/blog/{id}", requireAuth(http.HandlerFunc(blogHandler.Update)))
	mux.Handle("DELETE /api/v1/blog/{id}", requireAuth(http.HandlerFunc(blogHandler.Delete)))

	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	return middleware.Chain(mux,
		middleware.CORS,
		middleware.RequestID,
		middleware.Logger,
		middleware.Maintenance(s.cfg.JWTSecret, settingService.IsMaintenance, adminUserService.IsAdmin),
	)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	httputil.OK(w, map[string]string{"status": "ok"})
}
