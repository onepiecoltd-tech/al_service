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

	requireAuth := middleware.Auth(s.cfg.JWTSecret)

	userRepo := repository.NewUserRepository(s.db)
	blogRepo := repository.NewBlogRepository(s.db)
	giftRepo := repository.NewGiftRepository(s.db)
	notificationRepo := repository.NewNotificationRepository(s.db)
	reportRepo := repository.NewReportRepository(s.db)
	settingRepo := repository.NewSettingRepository(s.db)
	examRepo := repository.NewExamRepository(s.db)

	authService := service.NewAuthService(userRepo, s.cfg.JWTSecret)
	profileService := service.NewProfileService(userRepo)
	blogService := service.NewBlogService(blogRepo)
	leaderboardService := service.NewLeaderboardService(userRepo)
	friendService := service.NewFriendService(userRepo)
	giftService := service.NewGiftService(giftRepo)
	notificationService := service.NewNotificationService(notificationRepo)
	adminUserService := service.NewAdminUserService(userRepo)
	reportService := service.NewReportService(reportRepo)
	settingService := service.NewSettingService(settingRepo)
	examService := service.NewExamService(examRepo)

	authHandler := handler.NewAuthHandler(authService)
	profileHandler := handler.NewProfileHandler(profileService)
	blogHandler := handler.NewBlogHandler(blogService, profileService)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardService)
	friendHandler := handler.NewFriendHandler(friendService)
	giftHandler := handler.NewGiftHandler(giftService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	adminUserHandler := handler.NewAdminUserHandler(adminUserService)
	adminReportHandler := handler.NewAdminReportHandler(reportService)
	adminSettingHandler := handler.NewAdminSettingHandler(settingService)
	adminExamHandler := handler.NewAdminExamHandler(examService, profileService)

	requireAdmin := middleware.AdminOnly(adminUserService.IsAdmin)

	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	mux.Handle("GET /api/v1/me", requireAuth(http.HandlerFunc(profileHandler.Me)))
	mux.Handle("GET /api/v1/leaderboard", requireAuth(http.HandlerFunc(leaderboardHandler.List)))
	mux.Handle("GET /api/v1/friends", requireAuth(http.HandlerFunc(friendHandler.List)))

	mux.HandleFunc("GET /api/v1/gifts", giftHandler.List)

	mux.Handle("GET /api/v1/notifications", requireAuth(http.HandlerFunc(notificationHandler.List)))
	mux.Handle("POST /api/v1/notifications/read", requireAuth(http.HandlerFunc(notificationHandler.MarkAllRead)))

	mux.Handle("GET /api/v1/admin/users", requireAuth(requireAdmin(http.HandlerFunc(adminUserHandler.List))))
	mux.Handle("POST /api/v1/admin/users", requireAuth(requireAdmin(http.HandlerFunc(adminUserHandler.Create))))
	mux.Handle("PUT /api/v1/admin/users/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminUserHandler.Update))))

	mux.Handle("GET /api/v1/admin/reports", requireAuth(requireAdmin(http.HandlerFunc(adminReportHandler.List))))
	mux.Handle("POST /api/v1/admin/reports/{id}/resolve", requireAuth(requireAdmin(http.HandlerFunc(adminReportHandler.Resolve))))

	mux.Handle("GET /api/v1/admin/settings", requireAuth(requireAdmin(http.HandlerFunc(adminSettingHandler.List))))
	mux.Handle("PUT /api/v1/admin/settings/{key}", requireAuth(requireAdmin(http.HandlerFunc(adminSettingHandler.Update))))

	mux.Handle("GET /api/v1/admin/exams", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.List))))
	mux.Handle("POST /api/v1/admin/exams", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Create))))
	mux.Handle("PUT /api/v1/admin/exams/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Update))))
	mux.Handle("DELETE /api/v1/admin/exams/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminExamHandler.Delete))))

	mux.HandleFunc("GET /api/v1/blog", blogHandler.List)
	mux.HandleFunc("GET /api/v1/blog/{id}", blogHandler.Get)
	mux.Handle("POST /api/v1/blog", requireAuth(http.HandlerFunc(blogHandler.Create)))
	mux.Handle("PUT /api/v1/blog/{id}", requireAuth(http.HandlerFunc(blogHandler.Update)))
	mux.Handle("DELETE /api/v1/blog/{id}", requireAuth(http.HandlerFunc(blogHandler.Delete)))

	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	return middleware.Chain(mux,
		middleware.CORS,
		middleware.RequestID,
		middleware.Logger,
	)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	httputil.OK(w, map[string]string{"status": "ok"})
}
