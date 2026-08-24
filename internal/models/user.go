package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID  `json:"id"`
	UserName       string     `json:"user_name"`
	Email          string     `json:"email"`
	Password       string     `json:"password"`
	Role           string     `json:"role"`
	TokenVersion   int        `json:"token_version"` // added
	IsBanned       bool       `json:"is_banned"`
	BanReason      string     `json:"ban_reason"`
	BanUntil       *time.Time `json:"ban_until"`        // optional, RFC3339 format
	IsPermanentBan bool       `json:"is_permanent_ban"` // true = permanent ban
}

type UpdateUserRequest struct {
	UserName *string `json:"user_name"`
	Password *string `json:"password"`
}
type UserProfileResponse struct {
    ID             uuid.UUID      `json:"id"`
    UserName       string         `json:"user_name"`
    ProfilePicture *string        `json:"profile_picture"` // nullable
    Bio            string         `json:"bio"`
    JoinDate       time.Time      `json:"join_date"`
    ReviewCount    int64          `json:"review_count"`
    LikeCount      int64          `json:"like_count"`
    CommentCount   int64          `json:"comment_count"`
    FollowerCount  int64          `json:"follower_count"`
    FollowingCount int64          `json:"following_count"`
    IsFollowing    bool           `json:"is_following"`
    IsOwnProfile   bool           `json:"is_own_profile"`
}
type UserResponse struct {
	ID       uuid.UUID `json:"id"`
	UserName string    `json:"user_name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	CreatedAt      time.Time  `json:"created_at"`
	TokenVersion   int        `json:"token_version"` // added
	IsBanned       bool       `json:"is_banned"`
	BanReason      string     `json:"ban_reason"`
	BanUntil       *time.Time `json:"ban_until"` // nil = no ban date
	IsPermanentBan bool       `json:"is_permanent_ban"`
	LastActivity   time.Time  `json:"last_activity"`
}
type BanRequest struct {
	IsBanned       bool       `json:"is_banned"`
	BanReason      string     `json:"ban_reason"`
	BanUntil       *time.Time `json:"ban_until"`        // nullable
	IsPermanentBan bool       `json:"is_permanent_ban"` // true = permanent ban
}
