package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/skillissu3e/notify-platform/internal/common"

	"github.com/skillissu3e/notify-platform/notification-service/internal/handler"
	authmiddleware "github.com/skillissu3e/notify-platform/notification-service/internal/middleware"
	"github.com/skillissu3e/notify-platform/notification-service/internal/repository"
	"github.com/skillissu3e/notify-platform/notification-service/internal/service"
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
	notificationProcessingTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notification_processing_seconds",
			Help:    "Time taken to process notification",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"channel"},
	)
	queueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rabbitmq_queue_size",
			Help: "Current size of RabbitMQ queues",
		},
		[]string{"queue"},
	)
	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of database queries",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query"},
	)
	notificationStatusTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_status_total",
			Help: "Total notifications by status",
		},
		[]string{"status", "channel"},
	)
	auditActionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audit_actions_total",
			Help: "Total audit actions",
		},
		[]string{"action", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(rabbitmqPublishedTotal)
	prometheus.MustRegister(notificationProcessingTime)
	prometheus.MustRegister(queueSize)
	prometheus.MustRegister(dbQueryDuration)
	prometheus.MustRegister(notificationStatusTotal)
	prometheus.MustRegister(auditActionsTotal)
}

func main() {
	// Загружаем секреты из Vault перед инициализацией
	common.InitializeSecretsFromVault()

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
		rabbitMQHost = "rabbitmq"
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
		mysqlDSN = "app:app_password@tcp(mysql:3306)/notify?parseTime=true"
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

	// Инициализация Redis для rate limiting
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: "", // no password
		DB:       0,  // default DB
	})

	// Проверка подключения к Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis: ", err)
	}
	log.Println("Successfully connected to Redis")

	// Инициализация репозиториев
	notificationRepo := repository.NewNotificationRepository(db)
	rateLimitRepo := repository.NewRateLimitRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	auditService := service.NewAuditService(auditLogRepo)

	// Инициализация Echo
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	// Middleware для восстановления после паники
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll:   false,
		DisablePrintStack: false,
	}))

	// Настройка структурированного логгирования
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			requestID, ok := c.Get("requestID").(string)
			if !ok {
				requestID = "unknown"
			}
			log.Printf(`{"time":"%v","id":"%s","method":"%s","uri":"%s","status":%d,"remote_ip":"%s","user_agent":"%s","latency":%d,"error":"%s"}`,
				time.Now().Format(time.RFC3339),
				requestID,
				v.Method,
				v.URI,
				v.Status,
				v.RemoteIP,
				v.UserAgent,
				v.Latency.Nanoseconds(),
				v.Error,
			)
			return nil
		},
	}))

	// Middleware для генерации Request ID
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()

			// Генерация уникального ID запроса
			requestID := req.Header.Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			res.Header().Set(echo.HeaderXRequestID, requestID)
			c.Set("requestID", requestID)

			return next(c)
		}
	})

	// Middleware для сбора метрик и обработки паники
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC RECOVERED: %v\n", r)
					log.Println("Stack trace:")
					debug.PrintStack()
					c.Error(echo.NewHTTPError(http.StatusInternalServerError, "internal server error"))
				}
			}()

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

	// Middleware для rate limiting
	e.Use(authmiddleware.RateLimiter(authmiddleware.RateLimiterConfig{
		RedisClient:   rdb,
		RateLimitRepo: rateLimitRepo,
		ErrorHandler: func(c echo.Context, err error) error {
			if he, ok := err.(*echo.HTTPError); ok {
				return c.JSON(he.Code, map[string]interface{}{
					"error":   he.Message,
					"details": "Too many requests, please try again later",
				})
			}
			return err
		},
	}))

	// Public routes
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// JWT Middleware
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Группа с аутентификацией
	authGroup := e.Group("")
	authGroup.Use(authmiddleware.JWTAuth(jwtSecret))

	// Защищенные маршруты
	authGroup.POST("/notify", handler.NotifyHandler(ch, rabbitmqPublishedTotal, notificationRepo, auditService))
	authGroup.GET("/history", handler.NotificationHistoryHandler(notificationRepo, auditService))
	authGroup.GET("/stats", handler.NotificationStatsHandler(notificationRepo))
	authGroup.GET("/audit", handler.AuditLogHandler(auditService))

	// Запуск сервера с HTTPS
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8443"
	}

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	if certFile == "" || keyFile == "" {
		log.Fatal("TLS_CERT_FILE and TLS_KEY_FILE environment variables are required")
	}

	log.Printf("Starting Notification Service on port %s", port)
	e.Logger.Fatal(e.StartTLS(":"+port, certFile, keyFile))
}
