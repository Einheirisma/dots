package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	amqp "github.com/rabbitmq/amqp091-go"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	// Базовая валидация
	if err := cv.validator.Struct(i); err != nil {
		return err
	}

	// Кастомная валидация для NotificationRequest
	if req, ok := i.(handler.NotificationRequest); ok {
		if strings.ToLower(req.Channel) == "email" && req.Subject == "" {
			return errors.New("subject is required for email notifications")
		}
	}
	return nil
}

type NotificationRequest struct {
	Channel   string            `json:"channel" validate:"required,oneof=email telegram whatsapp"`
	Recipient string            `json:"recipient" validate:"required"`
	Subject   string            `json:"subject"` // Не обязательное поле
	Body      string            `json:"body" validate:"required"`
	Data      map[string]string `json:"data"`
}

func NotifyHandler(ch *amqp.Channel, counter *prometheus.CounterVec) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req NotificationRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		// Кастомная валидация для Subject
		if strings.ToLower(req.Channel) == "email" && req.Subject == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject is required for email notifications"})
		}

		if err := c.Validate(req); err != nil {
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
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to publish message"})
		}

		// Инкрементируем счетчик для RabbitMQ
		counter.WithLabelValues("notifications", req.Channel).Inc()

		return c.JSON(http.StatusAccepted, map[string]string{"status": "queued"})
	}
}
