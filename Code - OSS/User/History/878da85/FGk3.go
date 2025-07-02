package handler

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/skillissu3e/notify-platform/pkg/common"
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
