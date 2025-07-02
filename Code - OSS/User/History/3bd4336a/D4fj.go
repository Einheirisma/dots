package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
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

type GreenAPIResponse struct {
	IdMessage string `json:"idMessage"`
}

func main() {
	idInstance := os.Getenv("GREEN_API_INSTANCE")
	apiToken := os.Getenv("GREEN_API_TOKEN")

	if idInstance == "" || apiToken == "" {
		log.Fatal("GREEN_API credentials not set")
	}

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

	_, _ = ch.QueueDelete("whatsapp_queue", false, false, false)

	// Создаем очередь
	q, err := ch.QueueDeclare(
		"whatsapp_queue",
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
		"whatsapp",
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
		log.Println("Starting metrics server on :8083")
		log.Fatal(http.ListenAndServe(":8083", nil))
	}()

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

	log.Printf("WhatsApp worker started. Waiting for messages...")

	for d := range msgs {
		var msg NotificationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, false)
			continue
		}

		if err := sendWhatsAppMessage(idInstance, apiToken, msg); err != nil {
			log.Printf("Failed to send WhatsApp message: %v", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, true)
			continue
		}

		updateNotificationStatus(db, msg.ID, "sent")
		d.Ack(false)
		log.Printf("Message sent to %s", msg.Recipient)
	}
}

func sendWhatsAppMessage(idInstance, apiToken string, msg NotificationMessage) error {
	url := fmt.Sprintf("https://api.green-api.com/waInstance%s/sendMessage/%s", idInstance, apiToken)
	payload := map[string]interface{}{
		"chatId":  fmt.Sprintf("%s@c.us", msg.Recipient),
		"message": msg.Body,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("green-api error: %s", resp.Status)
	}

	var result GreenAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	log.Printf("Message ID: %s", result.IdMessage)
	return nil
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
