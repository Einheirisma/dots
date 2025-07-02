package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationMessage struct {
	Channel   string            `json:"channel"`
	Recipient string            `json:"recipient"` // Telegram Chat ID
	Subject   string            `json:"subject"`   // Не используется в Telegram
	Body      string            `json:"body"`      // Текст сообщения
	Data      map[string]string `json:"data"`      // Дополнительные данные
}

func main() {
	// Получаем токен бота из переменных окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	log.Printf("Initializing Telegram bot with token: %s", botToken)

	// Создаем бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Получаем параметры подключения к RabbitMQ
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

	amqpURI := "amqp://" + rabbitMQUser + ":" + rabbitMQPass + "@" + rabbitMQHost + ":5672/"

	// Подключение к RabbitMQ
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Объявляем exchange
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
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	// Создаем очередь
	q, err := ch.QueueDeclare(
		"telegram_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	log.Printf("Telegram queue declared: %s (%d messages, %d consumers)",
		q.Name, q.Messages, q.Consumers)

	// Привязываем очередь к exchange
	err = ch.QueueBind(
		q.Name,
		"telegram",
		"notifications",
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	// Регистрируем потребителя
	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Printf("Telegram worker started. Waiting for messages...")

	// Обработка сообщений
	for d := range msgs {
		log.Printf("Received raw message: %s", d.Body)

		var msg NotificationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			d.Nack(false, false) // Отбрасываем сообщение
			continue
		}

		log.Printf("Processing message: %+v", msg)

		// Отправляем сообщение в Telegram
		if err := sendTelegramMessage(bot, msg); err != nil {
			log.Printf("Failed to send Telegram message: %v", err)
			d.Nack(false, true) // Возвращаем в очередь для повторной попытки
			continue
		}

		log.Printf("Message successfully sent to %s", msg.Recipient)
		d.Ack(false) // Подтверждаем обработку
	}
}

func sendTelegramMessage(bot *tgbotapi.BotAPI, msg NotificationMessage) error {
	// Преобразуем Chat ID в int64
	chatID, err := strconv.ParseInt(msg.Recipient, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID format: %w", err)
	}

	log.Printf("Sending message to chat %d: %s", chatID, msg.Body)

	// Создаем сообщение
	message := tgbotapi.NewMessage(chatID, msg.Body)

	// Добавляем форматирование, если указано
	if parseMode, ok := msg.Data["parse_mode"]; ok {
		message.ParseMode = parseMode
	}

	// Отправляем сообщение
	_, err = bot.Send(message)
	if err != nil {
		return fmt.Errorf("telegram API error: %w", err)
	}

	return nil
}
