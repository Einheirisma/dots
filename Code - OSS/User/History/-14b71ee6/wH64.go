package main

import (
	"log"
	"os"

	"notify-platform/notification-service/internal/handler"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	// Подключение к RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
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
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to declare an exchange:", err)
	}

	// Инициализация Echo
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()} // Регистрируем валидатор

	// Маршруты
	e.POST("/notify", handler.NotifyHandler(ch))

	// Запуск сервера
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8082"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
