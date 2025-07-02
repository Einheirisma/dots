package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"log"
	"net/smtp"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
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
	// Получаем параметры подключения
	rabbitMQHost := os.Getenv("RABBITMQ_HOST")
	if rabbitMQHost == "" {
		rabbitMQHost = "rabbitmq"
	}
	rabbitMQUser := os.Getenv("RABBITMQ_USER")
	if rabbitMQUser == "" {
		rabbitMQUser = "guest"
	}
	rabbitMQPass := os.Getenv("RABBITMQ_PASSWORD")
	if rabbitMQPass == "" {
		rabbitMQPass = "guest"
	}

	amqpURI := "amqp://" + rabbitMQUser + ":" + rabbitMQPass + "@" + rabbitMQHost + ":5672/"
	log.Printf("Connecting to RabbitMQ at: %s", amqpURI)

	// Подключение к RabbitMQ
	var conn *amqp.Connection
	var err error
	for i := 0; i < 5; i++ {
		conn, err = amqp.Dial(amqpURI)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ (attempt %d): %v", i+1, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ after 5 attempts:", err)
	}
	defer conn.Close()
	log.Println("Successfully connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open a channel:", err)
	}
	defer ch.Close()

	// Проверка существования Exchange
	err = ch.ExchangeDeclarePassive(
		"notifications",
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Printf("Exchange 'notifications' does not exist, declaring a new one: %v", err)
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
			log.Fatal("Failed to declare an exchange:", err)
		}
		log.Println("Created new exchange 'notifications'")
	} else {
		log.Println("Exchange 'notifications' verified")
	}

	// Создаем очередь
	q, err := ch.QueueDeclare(
		"email_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatal("Failed to declare queue:", err)
	}

	// Привязываем очередь к exchange
	err = ch.QueueBind(
		q.Name,
		"email",
		"notifications",
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to bind queue:", err)
	}
	log.Printf("Queue '%s' bound to exchange 'notifications' with routing key 'email'", q.Name)

	// Инициализация базы данных
	mysqlDSN := os.Getenv("MYSQL_DSN")
	if mysqlDSN == "" {
		mysqlDSN = "app:app_password@tcp(mysql:3306)/notify?parseTime=true"
	}
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatal("MySQL connection failed:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("MySQL ping failed:", err)
	}
	log.Println("Successfully connected to MySQL")

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
		log.Fatal("Failed to register consumer:", err)
	}

	log.Printf("Email worker started. Waiting for messages...")

	for d := range msgs {
		log.Printf("Received a message: %s", d.Body)

		var msg NotificationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %s", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, false)
			continue
		}

		log.Printf("Processing notification ID: %s for %s", msg.ID, msg.Recipient)

		// Отправка email
		if err := sendEmail(msg.Recipient, msg.Subject, msg.Body); err != nil {
			log.Printf("Failed to send email: %s", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, true)
			continue
		}

		log.Printf("Email sent to %s", msg.Recipient)
		updateNotificationStatus(db, msg.ID, "sent")
		d.Ack(false)
	}
}

func sendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.yandex.ru"
	}
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "465"
	}
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")

	log.Printf("Sending email to %s via %s:%s", to, smtpHost, smtpPort)

	from := smtpUser
	if from == "" {
		from = "noreply@yandex.ru"
	}

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	// Настройка TLS
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpHost,
	}

	// Подключение с TLS
	conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, tlsconfig)
	if err != nil {
		log.Printf("TLS connection error: %v", err)
		return err
	}

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		log.Printf("SMTP client error: %v", err)
		return err
	}
	defer client.Close()

	// Аутентификация
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	if err = client.Auth(auth); err != nil {
		log.Printf("SMTP auth error: %v", err)
		return err
	}

	// Установка отправителя
	if err = client.Mail(from); err != nil {
		log.Printf("Mail from error: %v", err)
		return err
	}

	// Установка получателя
	if err = client.Rcpt(to); err != nil {
		log.Printf("Rcpt to error: %v", err)
		return err
	}

	// Отправка данных
	w, err := client.Data()
	if err != nil {
		log.Printf("Data error: %v", err)
		return err
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		log.Printf("Write error: %v", err)
		return err
	}

	err = w.Close()
	if err != nil {
		log.Printf("Close writer error: %v", err)
		return err
	}

	client.Quit()

	log.Println("Email sent successfully")
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
		log.Printf("Failed to update status for %s: %v", id, err)
	} else {
		log.Printf("Updated status for %s to %s", id, status)
	}
}
