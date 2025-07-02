package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationRequest struct {
	Channel   string            `json:"channel" validate:"required,oneof=email telegram whatsapp"`
	Recipient string            `json:"recipient" validate:"required"`
	Subject   string            `json:"subject"` // Убрали валидацию required
	Body      string            `json:"body" validate:"required"`
	Data      map[string]string `json:"data"`
}

func NotifyHandler(ch *amqp.Channel, counter *prometheus.CounterVec) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req NotificationRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		// Создаем кастомный валидатор
		validate := validator.New()
		err := validate.StructExcept(req, "Subject") // Явно исключаем Subject из валидации

		// Кастомная валидация для Subject
		if strings.ToLower(req.Channel) == "email" && req.Subject == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject is required for email notifications"})
		}

		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

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
			},
		)
		if err != nil {
			// Добавляем детали ошибки
			return c.JSON(http.StatusInternalServerError,
				map[string]string{"error": "failed to publish message: " + err.Error()})
		}

		// Инкрементируем счетчик для RabbitMQ
		counter.WithLabelValues("notifications", req.Channel).Inc()

		return c.JSON(http.StatusAccepted, map[string]string{"status": "queued"})
	}
}
