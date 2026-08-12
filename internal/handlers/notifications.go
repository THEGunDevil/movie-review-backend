package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
)

type NotificationsHandler struct {
	Queries *gen.Queries
}

func (n *NotificationsHandler) NotificationsHandler(c *gin.Context) {
	notifs, err := n.Queries.GetNotifications(c.Request.Context(), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, notifs)
}
func (n *NotificationsHandler) MarkReadHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	err = n.Queries.MarkNotificationRead(c.Request.Context(), service.UUIDToPGType(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "marked read"})
}

// helper
func (n *NotificationsHandler) notificationFromEvent(eventType string, payload map[string]interface{}) (string, string) {
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