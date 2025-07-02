package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/smtp"
	"os"

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
	// Получаем параметры подключения из переменных окружения
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
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open a channel:", err)
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
		log.Fatal("Failed to declare an exchange:", err)
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
		log.Fatal("Failed to declare a queue:", err)
	}

	log.Printf("Email queue declared: %s (%d messages, %d consumers)",
		q.Name, q.Messages, q.Consumers)

	// Привязываем очередь к exchange
	err = ch.QueueBind(
		q.Name,
		"email",
		"notifications",
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to bind a queue:", err)
	}

	// Инициализация базы данных
	mysqlDSN := os.Getenv("MYSQL_DSN")
	if mysqlDSN == "" {
		mysqlDSN = "app:app_password@tcp(localhost:3306)/notify?parseTime=true"
	}
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatal("Failed to connect to MySQL:", err)
	}
	defer db.Close()

	// Проверка подключения к БД
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
		log.Fatal("Failed to register a consumer:", err)
	}

	// Обработка сообщений
	log.Printf("Email worker started. Waiting for messages...")

	for d := range msgs {
		log.Printf("Received a message: %s", d.Body)

		// Десериализация сообщения
		var msg NotificationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %s", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, false) // Отбрасываем сообщение без повторной обработки
			continue
		}

		log.Printf("Processing notification ID: %s", msg.ID)

		// Отправка email
		if err := sendEmail(msg.Recipient, msg.Subject, msg.Body); err != nil {
			log.Printf("Failed to send email: %s", err)
			updateNotificationStatus(db, msg.ID, "failed")
			d.Nack(false, true) // Возвращаем в очередь для повторной попытки
			continue
		}

		log.Printf("Email sent to %s", msg.Recipient)
		updateNotificationStatus(db, msg.ID, "sent")
		d.Ack(false) // Подтверждаем успешную обработку
	}
}

func sendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "localhost"
	}
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "1026"
	}
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")

	// Формируем сообщение
	from := "notify@example.com"
	if smtpUser != "" {
		from = smtpUser
	}

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	// Аутентификация (если есть учетные данные)
	var auth smtp.Auth
	if smtpUser != "" && smtpPass != "" {
		auth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	}

	// Отправка письма
	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		from,
		[]string{to},
		[]byte(msg),
	)

	return err
}

func updateNotificationStatus(db *sql.DB, id, status string) {
	_, err := db.Exec(
		"UPDATE notifications SET status = ?, sent_at = NOW() WHERE id = ?",
		status,
		id,
	)
	if err != nil {
		log.Printf("Failed to update notification status: %v", err)
	}
}
