package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/skillissu3e/notify-platform/notification-service/internal/handler"
	"github.com/skillissu3e/notify-platform/notification-service/internal/repository"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return err
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

	// Инициализация базы данных
	mysqlDSN := os.Getenv("MYSQL_DSN")
	if mysqlDSN == "" {
		mysqlDSN = "app:app_password@tcp(localhost:3306)/notify?parseTime=true"
	}
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatal("Failed to connect to MySQL:", err)
	}
	defer db.Close()

	// Проверка подключения к БД
	if err := db.Ping(); err != nil {
		log.Fatal("MySQL ping failed:", err)
	}
	log.Println("Successfully connected to MySQL")

	// Инициализация репозиториев
	notificationRepo := repository.NewNotificationRepository(db)
	//templateRepo := repository.NewTemplateRepository(db)

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

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
	}))

	// Маршруты
	e.POST("/notify", handler.NotifyHandler(ch, rabbitmqPublishedTotal, notificationRepo))
	e.GET("/history", handler.NotificationHistoryHandler(notificationRepo))
	e.GET("/stats", handler.NotificationStatsHandler(notificationRepo))
	e.POST("/templates", handler.CreateTemplateHandler(db))
	e.GET("/templates", handler.ListTemplatesHandler(db))
	e.GET("/templates/:id", handler.GetTemplateHandler(db))
	e.PUT("/templates/:id", handler.UpdateTemplateHandler(db))
	e.DELETE("/templates/:id", handler.DeleteTemplateHandler(db))

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
