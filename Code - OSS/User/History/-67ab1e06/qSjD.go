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
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/skillissu3e/notify-platform/user-service/internal/handler"
	"github.com/skillissu3e/notify-platform/user-service/internal/middleware/auth"
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
	// Инициализация MySQL
	db, err := sql.Open("mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			os.Getenv("MYSQL_USER"),
			os.Getenv("MYSQL_PASSWORD"),
			"localhost",
			"3306",
			os.Getenv("MYSQL_DATABASE")))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("MySQL connection failed:", err)
	}

	// Инициализация Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	// Настройка Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

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

	// Защищенные маршруты
	secured := e.Group("/api")
	secured.Use(auth.JWTAuth(os.Getenv("JWT_SECRET")))
	secured.GET("/me", func(c echo.Context) error {
		userID := c.Get("userID").(string)
		return c.JSON(http.StatusOK, map[string]string{"user_id": userID})
	})

	// Метрики Prometheus
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Запуск сервера
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8081"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
