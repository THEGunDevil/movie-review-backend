package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	gen "github.com/internal/db/gen"
	"github.com/internal/service"
)

type NotificationsHandler struct {
	Queries *gen.Queries
	Hub     *NotificationHub
}

type notificationPayload struct {
	Type   string `json:"type"`
	Avatar string `json:"avatar,omitempty"`
	Link   string `json:"link,omitempty"`
}

func uuidToString(id uuid.UUID) string {
	return id.String()
}

// GET /notifications
func (n *NotificationsHandler) NotificationsHandler(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}

	notifs, err := n.Queries.GetNotificationsByUser(
		c.Request.Context(),
		gen.GetNotificationsByUserParams{
			RecipientID: service.UUIDToPGType(userID),
			Limit:       50,
		},
	)

	if err != nil {
		log.Printf("GetNotificationsByUser error: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch notifications",
		})
		return
	}

	result := make([]NotificationEvent, 0, len(notifs))

	for _, notif := range notifs {
		event := NotificationEvent{
			ID:        uuidToString(uuid.UUID(notif.ID.Bytes)),
			Title:     notif.Title,
			Message:   notif.Message,
			CreatedAt: notif.CreatedAt.Time.Format(time.RFC3339),
			Read:      notif.Read,
		}

		if notif.EventType.Valid {
			event.Type = notif.EventType.String
		}
		var payload notificationPayload

		if len(notif.Payload) > 0 {
			if err := json.Unmarshal(notif.Payload, &payload); err == nil {
				event.Avatar = payload.Avatar
				event.Link = payload.Link

				if payload.Type != "" {
					event.Type = payload.Type
				}
			}
		}

		result = append(result, event)
	}

	c.JSON(http.StatusOK, result)
}

// GET /notifications/stream
func (n *NotificationsHandler) NotificationsStreamHandler(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}

	client := make(chan []byte, 20)

	n.Hub.AddClient(userID, client)

	defer n.Hub.RemoveClient(userID, client)

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Reconnect suggestion
	_, _ = c.Writer.WriteString("retry: 5000\n\n")

	// Initial connected event
	_, _ = c.Writer.WriteString("event: connected\n")
	_, _ = c.Writer.WriteString(`data: {"status":"connected"}` + "\n\n")

	c.Writer.Flush()

	// Heartbeat
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	clientGone := c.Request.Context().Done()

	for {
		select {

		case <-clientGone:
			return

		case <-ticker.C:
			// SSE heartbeat
			_, _ = c.Writer.WriteString(": ping\n\n")
			c.Writer.Flush()

		case data, ok := <-client:
			if !ok {
				return
			}

			_, _ = c.Writer.WriteString("event: notification\n")
			_, _ = c.Writer.WriteString("data: ")
			_, _ = c.Writer.Write(data)
			_, _ = c.Writer.WriteString("\n\n")

			c.Writer.Flush()
		}
	}
}

// PATCH /notifications/:id/read
func (n *NotificationsHandler) MarkReadHandler(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}

	id, err := uuid.Parse(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid notification id",
		})
		return
	}

	err = n.Queries.MarkNotificationRead(
		c.Request.Context(),
		gen.MarkNotificationReadParams{
			ID:          service.UUIDToPGType(id),
			RecipientID: service.UUIDToPGType(userID),
		},
	)

	if err != nil {
		log.Printf("MarkNotificationRead error: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to mark notification as read",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "marked read",
	})
}

// POST /notifications/read-all
func (n *NotificationsHandler) MarkAllReadHandler(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}

	err := n.Queries.MarkAllNotificationsRead(
		c.Request.Context(),
		service.UUIDToPGType(userID),
	)

	if err != nil {
		log.Printf("MarkAllNotificationsRead error: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to mark all notifications as read",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "all notifications marked read",
	})
}
