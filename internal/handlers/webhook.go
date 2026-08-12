package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
)
type WebhookHandler struct {
	Queries *gen.Queries
}
// helper function – maps webhook event type to notification title & message
func notificationFromEvent(eventType string, payload map[string]interface{}) (string, string) {
	switch eventType {
	case "reply.created":
		user, _ := payload["user"].(string)
		parentTitle, _ := payload["parentTitle"].(string)
		return "New Reply", user + " replied to \"" + parentTitle + "\""
	case "rating.updated":
		movie, _ := payload["movie"].(string)
		rating, _ := payload["rating"].(string)
		return "Rating Changed", "The rating for " + movie + " is now " + rating
	default:
		return "", ""
	}
}
func (w *WebhookHandler)WebhookHandler(c *gin.Context) {
	var rawPayload map[string]interface{}
	if err := c.ShouldBindJSON(&rawPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	eventType, ok := rawPayload["type"].(string)
	if !ok || eventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'type' field"})
		return
	}

	payloadBytes, _ := json.Marshal(rawPayload)

	// sqlc generated function
	event, err := w.Queries.CreateWebhookEvent(c.Request.Context(), gen.CreateWebhookEventParams{
		EventType: eventType,
		Payload:   payloadBytes,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save event"})
		return
	}

	// Business logic: create a notification
	notifTitle, notifMsg := notificationFromEvent(eventType, rawPayload)
	if notifTitle != "" {
		_, err := w.Queries.CreateNotification(c.Request.Context(), gen.CreateNotificationParams{
			Title:   notifTitle,
			Message: notifMsg,
			EventID: service.UUIDToPGType(event.ID.Bytes),
		})
		if err != nil {
			log.Println("Failed to create notification:", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
