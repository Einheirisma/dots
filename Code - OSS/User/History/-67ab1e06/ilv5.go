package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/skillissu3e/notify-platform/user-service/internal/handler"
	authmiddleware "github.com/skillissu3e/notify-platform/user-service/internal/middleware"
	"github.com/skillissu3e/notify-platform/user-service/pkg/common"
)

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
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

func main() {
	// Загружаем секреты из Vault перед инициализацией
	vault.InitializeSecretsFromVault()
	// Получаем параметры подключения из переменных окружения
	mysqlHost := os.Getenv("MYSQL_HOST")
	if mysqlHost == "" {
		mysqlHost = "localhost"
	}
	mysqlPort := os.Getenv("MYSQL_PORT")
	if mysqlPort == "" {
		mysqlPort = "3306"
	}
	mysqlUser := os.Getenv("MYSQL_USER")
	if mysqlUser == "" {
		mysqlUser = "app"
	}
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	if mysqlPassword == "" {
		mysqlPassword = "app_password"
	}
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")
	if mysqlDatabase == "" {
		mysqlDatabase = "notify"
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	// Инициализация MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDatabase)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("MySQL connection failed:", err)
	}
	defer db.Close()

	// Устанавливаем параметры пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверка подключения
	err = db.Ping()
	if err != nil {
		log.Fatal("MySQL ping failed:", err)
	}
	log.Println("Successfully connected to MySQL")

	// Инициализация Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "", // нет пароля
		DB:       0,  // дефолтная БД
	})

	// Проверка подключения к Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Redis connection failed:", err)
	}
	log.Println("Successfully connected to Redis")

	// Настройка Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

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
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.POST("/register", handler.RegisterHandler(db))
	e.POST("/login", handler.LoginHandler(db, rdb))
	e.GET("/verify-email", handler.VerifyEmailHandler(db))
	e.POST("/forgot-password", handler.ForgotPasswordHandler(db))
	e.POST("/reset-password", handler.ResetPasswordHandler(db)) // Исправленный обработчик

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
	}))

	// Защищенные маршруты
	secured := e.Group("/api")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	secured.Use(authmiddleware.JWTAuth(jwtSecret))
	secured.GET("/me", handler.UserInfoHandler(db))
	e.GET("/verify-email", handler.VerifyEmailHandler(db))
	e.POST("/resend-verification", handler.ResendVerificationHandler(db))

	// Rate Limiting Middleware для критических эндпоинтов
	rateLimited := e.Group("")
	rateLimited.Use(common.RateLimiter(rdb, "rl", 10, time.Minute))
	rateLimited.POST("/register", handler.RegisterHandler(db))

	// Метрики Prometheus
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

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

	log.Printf("Starting User Service on port %s", port)
	e.Logger.Fatal(e.StartTLS(":"+port, certFile, keyFile))
}
