package middleware

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token format"})
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(secret), nil
			})

			if err != nil || token == nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token claims"})
			}

			exp, err := claims.GetExpirationTime()
			if err != nil || exp == nil || exp.Before(time.Now()) {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token expired"})
			}

			iss, err := claims.GetIssuer()
			if err != nil || iss != os.Getenv("JWT_ISSUER") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token issuer"})
			}

			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user ID not found in token"})
			}

			c.Set("userID", userID)
			return next(c)
		}
	}
}
