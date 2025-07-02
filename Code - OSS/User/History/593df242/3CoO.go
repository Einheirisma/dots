package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/skillissu3e/notify-platform/notification-service/internal/repository"
)

type NotificationRequest struct {
	Channel    string            `json:"channel" validate:"required,oneof=email telegram whatsapp"`
	Recipient  string            `json:"recipient" validate:"required"`
	Subject    string            `json:"subject"`
	Body       string            `json:"body" validate:"required"`
	Data       map[string]string `json:"data"`
	TemplateID int               `json:"template_id"`
}

func NotifyHandler(ch *amqp.Channel, counter *prometheus.CounterVec, repo *repository.NotificationRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req NotificationRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		if err := c.Validate(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Извлечение userID из JWT
		userID, err := strconv.Atoi(c.Get("userID").(string))
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		}

		// Создание записи в истории
		notification := &repository.Notification{
			ID:        uuid.New().String(),
			UserID:    userID,
			Channel:   req.Channel,
			Recipient: req.Recipient,
			Subject:   req.Subject,
			Body:      req.Body,
			Status:    "queued",
		}
		if err := repo.Create(notification); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to record notification"})
		}

		// Отправка в RabbitMQ
		message, err := json.Marshal(req)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to serialize message"})
		}

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
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to publish message"})
		}

		counter.WithLabelValues("notifications", req.Channel).Inc()

		return c.JSON(http.StatusAccepted, map[string]string{
			"status":  "queued",
			"channel": req.Channel,
			"id":      notification.ID,
		})
	}
}

func NotificationHistoryHandler(repo *repository.NotificationRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, err := strconv.Atoi(c.Get("userID").(string))
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		}

		limit := 50
		if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 {
			limit = l
		}

		notifications, err := repo.FindByUser(userID, limit)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get history"})
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
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		}

		return c.JSON(http.StatusOK, stats)
	}
}
