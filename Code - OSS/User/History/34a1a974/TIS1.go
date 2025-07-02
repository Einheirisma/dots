package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"github.com/skillissu3e/notify-platform/notification-service/internal/repository"
)

type RateLimiterConfig struct {
	RedisClient   *redis.Client
	RateLimitRepo *repository.RateLimitRepository
	ErrorHandler  func(c echo.Context, err error) error
}

func RateLimiter(config RateLimiterConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Извлекаем userID из контекста (установлено в JWTAuth middleware)
			userIDStr := c.Get("userID").(string)
			if userIDStr == "" {
				return config.ErrorHandler(c, echo.NewHTTPError(http.StatusUnauthorized, "User ID not found"))
			}

			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return config.ErrorHandler(c, echo.NewHTTPError(http.StatusInternalServerError, "Invalid user ID"))
			}

			// Извлекаем канал из запроса
			var req struct {
				Channel string `json:"channel"`
			}
			if err := c.Bind(&req); err != nil {
				return config.ErrorHandler(c, echo.NewHTTPError(http.StatusBadRequest, "Invalid request format"))
			}

			// Получаем лимиты из БД
			limit, err := config.RateLimitRepo.FindByUserAndChannel(userID, req.Channel)
			if err != nil {
				return config.ErrorHandler(c, echo.NewHTTPError(http.StatusInternalServerError, "Rate limit service unavailable"))
			}

			// Формируем ключ для Redis
			key := fmt.Sprintf("rl:%d:%s", userID, req.Channel)
			ctx := context.Background()

			// Проверяем текущий счетчик
			current, err := config.RedisClient.Get(ctx, key).Int()
			if err != nil && err != redis.Nil {
				return config.ErrorHandler(c, echo.NewHTTPError(http.StatusInternalServerError, "Rate limit error"))
			}

			// Проверяем превышение лимита
			if current >= limit.MaxRequests {
				retryAfter := config.RedisClient.TTL(ctx, key).Val()
				if retryAfter <= 0 {
					retryAfter = time.Duration(limit.Interval) * time.Second
				}

				c.Response().Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				return config.ErrorHandler(c, echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded"))
			}

			// Увеличиваем счетчик
			pipe := config.RedisClient.TxPipeline()
			pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, time.Duration(limit.Interval)*time.Second)
			_, err = pipe.Exec(ctx)
			if err != nil {
				return config.ErrorHandler(c, echo.NewHTTPError(http.StatusInternalServerError, "Rate limit error"))
			}

			return next(c)
		}
	}
}
