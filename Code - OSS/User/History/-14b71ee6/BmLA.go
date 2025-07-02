package main

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/skillissu3e/notify-platform/notification-service/internal/handler"
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

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	rabbitmqPublishedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_published_total",
			Help: "Total number of messages published to RabbitMQ",
		},
		[]string{"exchange", "routing_key"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(rabbitmqPublishedTotal)
}

func main() {
	// Получаем параметры подключения из переменных окружения
	rabbitMQUser := os.Getenv("RABBITMQ_USER")
	if rabbitMQUser == "" {
		rabbitMQUser = "guest"
	}
	rabbitMQPass := os.Getenv("RABBITMQ_PASSWORD")
	if rabbitMQPass == "" {
		rabbitMQPass = "guest"
	}
	rabbitMQHost := os.Getenv("RABBITMQ_HOST")
	if rabbitMQHost == "" {
		rabbitMQHost = "localhost"
	}

	// Формируем строку подключения
	amqpURI := "amqp://" + rabbitMQUser + ":" + rabbitMQPass + "@" + rabbitMQHost + ":5672/"

	// Подключение к RabbitMQ
	var conn *amqp.Connection
	var err error
	for i := 0; i < 5; i++ {
		conn, err = amqp.Dial(amqpURI)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ (attempt %d): %v", i+1, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ after 5 attempts:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open a channel:", err)
	}
	defer ch.Close()

	// Создание Exchange
	err = ch.ExchangeDeclare(
		"notifications",
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatal("Failed to declare an exchange:", err)
	}

	// Инициализация Echo
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	// Middleware для сбора метрик
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			duration := time.Since(start).Seconds()

			status := strconv.Itoa(c.Response().Status)
			httpRequestsTotal.WithLabelValues(c.Request().Method, c.Path(), status).Inc()
			httpRequestDuration.WithLabelValues(c.Request().Method, c.Path()).Observe(duration)

			return err
		}
	})

	// Маршруты
	e.POST("/notify", handler.NotifyHandler(ch, rabbitmqPublishedTotal))

	// Метрики Prometheus
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Запуск сервера
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Starting Notification Service on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
