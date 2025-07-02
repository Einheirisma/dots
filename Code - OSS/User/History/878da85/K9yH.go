package handler

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"github.com/skillissu3e/notify-platform/user-service/pkg/common"
)

func RegisterHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", req.Email).Scan(&exists)
		if err != nil {
			log.Printf("Database error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
		}
		if exists {
			return c.JSON(http.StatusConflict, map[string]string{"error": "email already exists"})
		}

		hashedPassword, err := common.HashPassword(req.Password)
		if err != nil {
			log.Printf("Password hashing failed: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "password hashing failed"})
		}

		verificationToken := uuid.New().String()

		_, err = db.Exec(
			"INSERT INTO users (email, password_hash, verification_token) VALUES (?, ?, ?)",
			req.Email,
			hashedPassword,
			verificationToken,
		)
		if err != nil {
			log.Printf("User creation failed: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user creation failed"})
		}

		go func() {
			subject := "Подтверждение регистрации"
			body := fmt.Sprintf(`
Для подтверждения email перейдите по ссылке:
https://localhost:8443/verify-email?token=%s

Ваш токен подтверждения: %s
			`, verificationToken, verificationToken)

			log.Printf("Sending verification email to %s", req.Email)
			if err := common.SendEmail(req.Email, subject, body); err != nil {
				log.Printf("Failed to send verification email: %v", err)
			} else {
				log.Printf("Verification email sent to %s", req.Email)
			}
		}()

		return c.JSON(http.StatusCreated, map[string]string{
			"status": "user created, check your email for verification instructions",
		})
	}
}

func VerifyEmailHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.QueryParam("token")
		if token == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "verification token is required"})
		}

		result, err := db.Exec(
			"UPDATE users SET verified = true, verification_token = NULL "+
				"WHERE verification_token = ?", token)
		if err != nil {
			log.Printf("Database error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Rows affected error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify token"})
		}

		if rowsAffected == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid or expired token"})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "email verified"})
	}
}

func LoginHandler(db *sql.DB, rdb *redis.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		var (
			id           int
			passwordHash string
			verified     bool
		)

		err := db.QueryRow(
			"SELECT id, password_hash, verified FROM users WHERE email = ?",
			req.Email,
		).Scan(&id, &passwordHash, &verified)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			}
			log.Printf("Database error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
		}

		if !common.VerifyPassword(req.Password, passwordHash) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		}

		if !verified {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error":      "email not verified",
				"verify_url": "/verify-email.html",
				"resend_url": "/resend-verification",
			})
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", id),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    os.Getenv("JWT_ISSUER"),
			Audience:  []string{"notify-platform"},
		})

		accessToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			log.Printf("Token generation failed: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
		}

		refreshToken := uuid.New().String()
		err = rdb.Set(c.Request().Context(),
			fmt.Sprintf("refresh:%d", id),
			refreshToken,
			7*24*time.Hour,
		).Err()
		if err != nil {
			log.Printf("Redis error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store refresh token"})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

func ForgotPasswordHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		type Request struct {
			Email string `json:"email"`
		}

		var req Request
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		log.Printf("Password reset request for: %s", req.Email)

		var userID int
		err := db.QueryRow("SELECT id FROM users WHERE email = ?", req.Email).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("User not found: %s", req.Email)
				return c.JSON(http.StatusOK, map[string]string{"status": "reset instructions sent if email exists"})
			}
			log.Printf("Database error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
		}

		resetToken := uuid.New().String()
		expiresAt := time.Now().Add(1 * time.Hour)

		_, err = db.Exec(
			"UPDATE users SET reset_token = ?, reset_expires = ? WHERE id = ?",
			resetToken,
			expiresAt,
			userID,
		)
		if err != nil {
			log.Printf("Failed to set reset token: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to set reset token"})
		}

		go func() {
			resetLink := fmt.Sprintf("https://localhost:8443/reset-password?token=%s", resetToken)
			subject := "Сброс пароля"
			body := fmt.Sprintf(`
Для сброса пароля перейдите по ссылке:
%s

Ваш токен сброса: %s
Действителен до: %s
			`, resetLink, resetToken, expiresAt.Format(time.RFC3339))

			log.Printf("Sending password reset email to %s", req.Email)
			if err := common.SendEmail(req.Email, subject, body); err != nil {
				log.Printf("Failed to send password reset email: %v", err)
			} else {
				log.Printf("Password reset email sent to %s", req.Email)
			}
		}()

		return c.JSON(http.StatusOK, map[string]string{"status": "reset instructions sent if email exists"})
	}
}

func ResetPasswordHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		type Request struct {
			Token           string `json:"token"`
			Password        string `json:"password"`
			ConfirmPassword string `json:"confirm_password"`
		}

		var req Request
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		if req.Password != req.ConfirmPassword {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "passwords do not match"})
		}

		if len(req.Password) < 8 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		}

		var userID int
		var expiresAtStr string
		err := db.QueryRow(
			"SELECT id, reset_expires FROM users WHERE reset_token = ?",
			req.Token,
		).Scan(&userID, &expiresAtStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid token"})
			}
			log.Printf("Database error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
		}

		expiresAt, err := time.Parse("2006-01-02 15:04:05", expiresAtStr)
		if err != nil {
			log.Printf("Time parse error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid time format"})
		}

		if time.Now().After(expiresAt) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "token expired"})
		}

		hashedPassword, err := common.HashPassword(req.Password)
		if err != nil {
			log.Printf("Password hashing error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "password hashing failed"})
		}

		_, err = db.Exec(
			"UPDATE users SET password_hash = ?, reset_token = NULL, reset_expires = NULL WHERE id = ?",
			hashedPassword,
			userID,
		)
		if err != nil {
			log.Printf("Password update error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update password"})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "password updated"})
	}
}

func ResendVerificationHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Email string `json:"email"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		log.Printf("Resending verification code to: %s", req.Email)

		var (
			userID            int
			verified          bool
			verificationToken string
		)
		err := db.QueryRow(
			"SELECT id, verified, verification_token FROM users WHERE email = ?",
			req.Email,
		).Scan(&userID, &verified, &verificationToken)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("User not found: %s", req.Email)
				return c.JSON(http.StatusOK, map[string]string{"status": "if email exists, code will be sent"})
			}
			log.Printf("Database error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
		}

		if verified {
			log.Printf("Email already verified: %s", req.Email)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "email already verified"})
		}

		if verificationToken == "" {
			verificationToken = uuid.New().String()
			_, err = db.Exec(
				"UPDATE users SET verification_token = ? WHERE id = ?",
				verificationToken,
				userID,
			)
			if err != nil {
				log.Printf("Failed to update verification token: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate new token"})
			}
		}

		go func() {
			subject := "Подтверждение регистрации"
			body := fmt.Sprintf(`
Ваш новый код подтверждения: %s
			`, verificationToken)

			log.Printf("Sending verification email to %s", req.Email)
			if err := common.SendEmail(req.Email, subject, body); err != nil {
				log.Printf("Failed to resend verification email: %v", err)
			} else {
				log.Printf("Verification email resent to %s", req.Email)
			}
		}()

		return c.JSON(http.StatusOK, map[string]string{"status": "verification code resent"})
	}
}

func UserInfoHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Get("userID").(string)
		if userID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		var (
			email    string
			verified bool
		)

		err := db.QueryRow(
			"SELECT email, verified FROM users WHERE id = ?",
			userID,
		).Scan(&email, &verified)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
			}
			log.Printf("Database error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":       userID,
			"email":    email,
			"verified": verified,
		})
	}
}

func VerifyResetTokenHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Token string `json:"token"`
		}

		if err := c.Bind(&req); err != nil || req.Token == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid token"})
		}

		var expiresAt time.Time
		err := db.QueryRow("SELECT reset_expires FROM users WHERE reset_token = ?", req.Token).Scan(&expiresAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid or expired token"})
			}
			log.Printf("DB error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server error"})
		}

		if time.Now().After(expiresAt) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "token expired"})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "valid"})
	}
}
