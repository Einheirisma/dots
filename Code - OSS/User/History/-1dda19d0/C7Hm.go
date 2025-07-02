package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
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

	// Создаем SMTP клиент
	addr := smtpHost + ":" + smtpPort
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	// Аутентификация
	if smtpUser != "" && smtpPass != "" {
		auth := sasl.NewPlainClient("", smtpUser, smtpPass)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	// Создаем сообщение
	var msg bytes.Buffer
	msg.WriteString("From: " + smtpUser + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	// Отправляем письмо
	if err := client.Mail(smtpUser, nil); err != nil {
		return err
	}
	if err := client.Rcpt(to, nil); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	if _, err := w.Write(msg.Bytes()); err != nil {
		return err
	}

	return nil
}
