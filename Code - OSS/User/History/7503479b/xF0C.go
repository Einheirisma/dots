package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationMessage struct {
	ID        string            `json:"id"`
	Channel   string            `json:"channel"`
	Recipient string            `json:"recipient"`
	Subject   string            `json:"subject"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

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

	amqpURI := "amqp://" + rabbitMQUser + ":" + rabbitMQPass + "@" + rabbitMQHost + ":5672/"
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

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
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	_, _ = ch.QueueDelete("telegram_queue", false, false, false)

	q, err := ch.QueueDeclare(
		"telegram_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare queue:", err)
	}

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

	mysqlDSN := os.Getenv("MYSQL_DSN")
	if mysqlDSN == "" {
		mysqlDSN = "app:app_password@tcp(mysql:3306)/notify?parseTime=true"
	}
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("MySQL ping failed: %v", err)
	}
	log.Println("MySQL connection established")

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Starting metrics server on :8084")
		if err := http.ListenAndServe(":8084", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()
	time.Sleep(1 * time.Second)
	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Printf("Telegram worker started. Waiting for messages...")

	for d := range msgs {
		var msg NotificationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, false)
			continue
		}

		chatID, err := strconv.ParseInt(msg.Recipient, 10, 64)
		if err != nil {
			log.Printf("Invalid chat ID: %v", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, false)
			continue
		}

		message := tgbotapi.NewMessage(chatID, msg.Body)
		if _, err := bot.Send(message); err != nil {
			log.Printf("Failed to send message: %v", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, true)
			continue
		}

		updateNotificationStatus(db, msg.ID, "sent")
		d.Ack(false)
		log.Printf("Message sent to %d", chatID)
	}
}

func updateNotificationStatus(db *sql.DB, id, status string) {
	_, err := db.Exec(
		"UPDATE notifications SET status = ?, sent_at = ? WHERE id = ?",
		status,
		time.Now(),
		id,
	)
	if err != nil {
		log.Printf("Failed to update status: %v", err)
	}
}
