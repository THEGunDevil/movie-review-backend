package handlers

import (
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/internal/db"
	gen "github.com/internal/db/gen"
	"github.com/internal/models"
	"github.com/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// GetUserHandler fetches user by email
func GetUsersHandler(c *gin.Context) {
	page := 1
	limit := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	params := gen.ListUsersPaginatedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	// 1️⃣ Fetch paginated users
	users, err := db.Q.ListUsersPaginated(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2️⃣ Count total users
	totalCount, err := db.Q.CountUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	// 3️⃣ Build response
	var response []models.UserResponse
	for _, user := range users {
		response = append(response, models.UserResponse{
			ID:             user.ID.Bytes,
			UserName:       user.UserName,
			Role:           user.Role,
			Email:          user.Email,
			IsBanned:       user.IsBanned,
			BanUntil:       &user.BanUntil.Time,
			BanReason:      user.BanReason.String,
			IsPermanentBan: user.IsPermanentBan,
			CreatedAt:      user.CreatedAt.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"count":       len(response),
		"total_count": totalCount,
		"total_pages": totalPages,
		"users":       response,
	})
}

func GetUserByIDHandler(c *gin.Context) {
	targetID, err := service.ParseUUIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// 1️⃣ Fetch user from DB
	user, err := db.Q.GetUserByID(c.Request.Context(), pgtype.UUID{Bytes: targetID, Valid: true})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	userRes := models.UserResponse{
		ID:             user.ID.Bytes,
		UserName:       user.UserName,
		Email:          user.Email,
		Role:           user.Role,
		IsBanned:       user.IsBanned,
		BanUntil:       &user.BanUntil.Time,
		BanReason:      user.BanReason.String,
		IsPermanentBan: user.IsPermanentBan,
		CreatedAt:      user.CreatedAt.Time,
	}
	log.Printf("👤 Returning user data for user %v (banned: %v)", user.ID, user.IsBanned)
	c.JSON(http.StatusOK, userRes)
}
func GetUserProfileByID(c *gin.Context) {
    targetID, err := service.ParseUUIDParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
        return
    }

    // Current user ID from JWT (or nil)
    currentUserID, exists := c.Get("user_id")
    var currentPgUUID pgtype.UUID
    if exists {
        uid, ok := currentUserID.(uuid.UUID)
        if ok {
            currentPgUUID = service.UUIDToPGType(uid)
        }
    } else {
        currentPgUUID = pgtype.UUID{} // zero value → NULL
    }

    user, err := db.Q.GetUserProfile(c.Request.Context(), gen.GetUserProfileParams{
        FollowingID:  service.UUIDToPGType(targetID),
        FollowerID: service.UUIDToPGType(currentPgUUID.Bytes),
    })
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    // Convert to response struct – include stats
    userRes := models.UserProfileResponse{
        ID:              user.ID.Bytes,
        UserName:        user.UserName,
        ProfilePicture:  &user.ProfilePicture.String,
        Bio:             user.Bio,
        JoinDate:        user.JoinDate.Time,
        ReviewCount:     user.ReviewCount,
        LikeCount:       user.LikeCount,
        CommentCount:    user.CommentCount,
        FollowerCount:   user.FollowerCount,
        FollowingCount:  user.FollowingCount,
        IsFollowing:     user.IsFollowing,
        IsOwnProfile:    exists && currentUserID.(uuid.UUID) == targetID, // or compare in SQL?
    }

    c.JSON(http.StatusOK, userRes)
}

// UpdateUserByIDHandler updates user by ID
func UpdateUserByIDHandler(c *gin.Context) {
	// Parse UUID
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	// Parse the incoming request
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params := gen.UpdateUserProfileParams{
		ID:       pgtype.UUID{Bytes: parsedID, Valid: true},
		UserName: *req.UserName,
	}
	if req.UserName != nil {
		params.UserName = *req.UserName
	}
	// Save changes
	updatedUser, err := db.Q.UpdateUserProfile(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	// Response
	resp := models.UserResponse{
		ID:             updatedUser.ID.Bytes,
		UserName:       updatedUser.UserName,
		Email:          updatedUser.Email,
		Role:           updatedUser.Role,
		IsBanned:       updatedUser.IsBanned,
		BanUntil:       &updatedUser.BanUntil.Time,
		BanReason:      updatedUser.BanReason.String,
		IsPermanentBan: updatedUser.IsPermanentBan,
		CreatedAt:      updatedUser.CreatedAt.Time,
	}

	c.JSON(http.StatusOK, resp)
}
func UpdateUserPassByIDHandler(c *gin.Context) {
	// Parse UUID
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	// Parse the incoming request
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hashed, err := service.HashPassword(*req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}
	params := gen.UpdatePasswordParams{
		ID:           pgtype.UUID{Bytes: parsedID, Valid: true},
		PasswordHash: hashed,
	}
	// Save changes
	err = db.Q.UpdatePassword(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "there was error changing the password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "password changed successfully!",
	})
}
