package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/skillissu3e/notify-platform/notification-service/internal/repository"
	"github.com/skillissu3e/notify-platform/notification-service/internal/service"
)

type NotificationRequest struct {
	Channel    string            `json:"channel" validate:"required,oneof=email telegram whatsapp"`
	Recipient  string            `json:"recipient" validate:"required"`
	Subject    string            `json:"subject" validate:"max=255"`
	Body       string            `json:"body"`
	Data       map[string]string `json:"data"`
	TemplateID int               `json:"template_id"`
}

type NotificationMessage struct {
	ID        string            `json:"id"`
	Channel   string            `json:"channel"`
	Recipient string            `json:"recipient"`
	Subject   string            `json:"subject"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
}

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^[0-9]{11,15}$`)
)

func NotifyHandler(ch *amqp.Channel, counter *prometheus.CounterVec, repo *repository.NotificationRepository, auditService *service.AuditService) echo.HandlerFunc {
	return func(c echo.Context) error {
		log.Println("Received notification request")

		var req NotificationRequest
		if err := c.Bind(&req); err != nil {
			log.Printf("Error binding request: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		validate := validator.New()
		if err := validate.Struct(req); err != nil {
			var errorDetails []string
			for _, err := range err.(validator.ValidationErrors) {
				switch err.Tag() {
				case "required":
					errorDetails = append(errorDetails, fmt.Sprintf("%s is required", err.Field()))
				case "oneof":
					errorDetails = append(errorDetails, fmt.Sprintf("%s must be one of: %s", err.Field(), err.Param()))
				case "max":
					errorDetails = append(errorDetails, fmt.Sprintf("%s is too long (max %s characters)", err.Field(), err.Param()))
				default:
					errorDetails = append(errorDetails, fmt.Sprintf("%s: %s", err.Field(), err.Tag()))
				}
			}
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error":   "validation failed",
				"details": errorDetails,
			})
		}

		if req.TemplateID == 0 && req.Body == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "body is required when template is not used",
			})
		}

		switch req.Channel {
		case "email":
			if !emailRegex.MatchString(req.Recipient) {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
			}
		case "whatsapp":
			if !phoneRegex.MatchString(req.Recipient) {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid phone number format"})
			}
		case "telegram":
			if _, err := strconv.ParseInt(req.Recipient, 10, 64); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "telegram chat ID must be integer"})
			}
		}

		if len(req.Subject) > 255 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject too long (max 255 characters)"})
		}

		log.Printf("Notification request: %+v", req)

		userIDStr := c.Get("userID").(string)
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			log.Printf("Invalid userID: %s", userIDStr)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		}

		log.Printf("User ID: %d", userID)

		notification := &repository.Notification{
			ID:        uuid.New().String(),
			UserID:    userID,
			Channel:   req.Channel,
			Recipient: req.Recipient,
			Subject:   req.Subject,
			Body:      req.Body,
			Status:    "queued",
		}

		log.Println("Creating notification record")
		if err := repo.Create(notification); err != nil {
			log.Printf("Failed to create notification record: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to record notification"})
		}

		if err := auditService.LogNotificationSent(userID, notification.ID, req.Channel, "queued"); err != nil {
			log.Printf("Failed to log audit event: %v", err)
		}

		messageBody := NotificationMessage{
			ID:        notification.ID,
			Channel:   req.Channel,
			Recipient: req.Recipient,
			Subject:   req.Subject,
			Body:      req.Body,
			Data:      req.Data,
		}

		message, err := json.Marshal(messageBody)
		if err != nil {
			log.Printf("Failed to serialize message: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to serialize message"})
		}

		log.Println("Publishing to RabbitMQ")
		err = ch.Publish(
			"notifications",
			req.Channel,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        message,
				Headers: amqp.Table{
					"notification_id": notification.ID,
				},
			},
		)
		if err != nil {
			log.Printf("Failed to publish message: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to publish message"})
		}

		log.Println("Message published successfully")
		counter.WithLabelValues("notifications", req.Channel).Inc()

		return c.JSON(http.StatusAccepted, map[string]string{
			"status":  "queued",
			"channel": req.Channel,
			"id":      notification.ID,
		})
	}
}

func NotificationHistoryHandler(repo *repository.NotificationRepository, auditService *service.AuditService) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, err := strconv.Atoi(c.Get("userID").(string))
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		}

		limit := 50
		if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 {
			limit = l
		}

		channel := c.QueryParam("channel")

		notifications, err := repo.FindByUser(userID, limit, channel)
		if err != nil {
			log.Printf("Failed to get history: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get history"})
		}

		if err := auditService.LogUserAction(userID, "HISTORY_VIEW", "notification", "", "success"); err != nil {
			log.Printf("Failed to log audit event: %v", err)
		}

		return c.JSON(http.StatusOK, notifications)
	}
}

func NotificationStatsHandler(repo *repository.NotificationRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, err := strconv.Atoi(c.Get("userID").(string))
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		}

		stats, err := repo.GetStats(userID)
		if err != nil {
			log.Printf("Failed to get stats: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		}

		return c.JSON(http.StatusOK, stats)
	}
}

func AuditLogHandler(auditService *service.AuditService) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, err := strconv.Atoi(c.Get("userID").(string))
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		}

		limit := 50
		if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 {
			limit = l
		}

		logs, err := auditService.GetUserHistory(userID, limit)
		if err != nil {
			log.Printf("Failed to get audit log: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get audit log"})
		}

		return c.JSON(http.StatusOK, logs)
	}
}
