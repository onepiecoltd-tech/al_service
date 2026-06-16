package model

import "time"

type RecentUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
}

type Overview struct {
	UsersTotal  int          `json:"users_total"`
	ProTotal    int          `json:"pro_total"`
	ReportsOpen int          `json:"reports_open"`
	ExamsReview int          `json:"exams_review"`
	PostsTotal  int          `json:"posts_total"`
	RecentUsers []RecentUser `json:"recent_users"`
}
