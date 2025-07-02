package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/skillissu3e/notify-platform/user-service/pkg/common"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type RegisterRequest struct {
	Email    string
	Password string
}

func RegisterHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req RegisterRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		// Проверка уникальности email
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", req.Email).Scan(&exists)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
		}
		if exists {
			return c.JSON(http.StatusConflict, map[string]string{"error": "email already exists"})
		}

		// Хеширование пароля
		hashedPassword, err := common.HashPassword(req.Password)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "password hashing failed"})
		}

		// Сохранение пользователя
		_, err = db.Exec(
			"INSERT INTO users (email, password_hash) VALUES (?, ?)",
			req.Email,
			hashedPassword,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user creation failed"})
		}

		return c.JSON(http.StatusCreated, map[string]string{"status": "user created"})
	}
}

type LoginRequest struct {
	Email    string
	Password string
}

func LoginHandler(db *sql.DB, rdb *redis.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req LoginRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		// Получаем пользователя из БД
		var (
			id           int
			passwordHash string
		)
		err := db.QueryRow(
			"SELECT id, password_hash FROM users WHERE email = ?",
			req.Email,
		).Scan(&id, &passwordHash)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		}

		// Проверяем пароль
		if !common.VerifyPassword(req.Password, passwordHash) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		}

		// Генерируем JWT токен
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", id),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		})

		accessToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
		}

		// Сохраняем refresh токен в Redis
		refreshToken := uuid.New().String()
		err = rdb.Set(c.Request().Context(),
			fmt.Sprintf("refresh:%d", id),
			refreshToken,
			7*24*time.Hour,
		).Err()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store refresh token"})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}
