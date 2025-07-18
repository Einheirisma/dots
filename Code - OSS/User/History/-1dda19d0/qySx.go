package main

import (
	"encoding/json"
	"log"
	"net/smtp"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationMessage struct {
	Channel   string            `json:"channel"`
	Recipient string            `json:"recipient"`
	Subject   string            `json:"subject"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
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

	// Объявляем exchange
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

	// Создаем очередь
	q, err := ch.QueueDeclare(
		"email_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to declare a queue:", err)
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
		log.Fatal("Failed to bind a queue:", err)
	}

	// Регистрируем потребителя
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
		log.Fatal("Failed to register a consumer:", err)
	}

	// Обработка сообщений
	forever := make(chan bool)

	log.Printf("Email worker started. Waiting for messages...")

	go func() {
		for d := range msgs {
			log.Printf("Received email message: %s", d.Body)

			// Десериализация сообщения
			var msg NotificationMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("Failed to unmarshal message: %s", err)
				d.Nack(false, false)
				continue
			}

			// Отправка email
			if err := sendEmail(msg.Recipient, msg.Subject, msg.Body); err != nil {
				log.Printf("Failed to send email: %s", err)
				d.Nack(false, true)
				continue
			}

			log.Printf("Email sent to %s", msg.Recipient)
			d.Ack(false)
		}
	}()

	<-forever
}

func sendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
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
