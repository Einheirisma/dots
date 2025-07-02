package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/emersion/go-smtp"
	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationMessage struct {
	Channel   string
	Recipient string
	Subject   string
	Body      string
	Data      map[string]string
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

	// Объявляем exchange (если еще не создан)
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
		"email_queue", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare a queue:", err)
	}

	// Привязываем очередь к exchange
	err = ch.QueueBind(
		q.Name,          // queue name
		"email",         // routing key
		"notifications", // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to bind a queue:", err)
	}

	// Регистрируем потребителя
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatal("Failed to register a consumer:", err)
	}

	// Обработка сообщений
	forever := make(chan bool)

	go func() {
		for d := range msgs {
			// Десериализация сообщения
			var msg NotificationMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("Failed to unmarshal message: %s", err)
				d.Nack(false, false) // Отбрасываем сообщение без повторной обработки
				continue
			}

			// Отправка email
			if err := sendEmail(msg.Recipient, msg.Subject, msg.Body); err != nil {
				log.Printf("Failed to send email: %s", err)
				d.Nack(false, true) // Повторная обработка
				continue
			}

			d.Ack(false)
		}
	}()

	log.Printf("Email worker started. Waiting for messages.")
	<-forever
}


func sendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")

	// Создаем SMTP клиент
	client, err := smtp.Dial(smtpHost + ":" + smtpPort)
	if err != nil {
		return err
	}
	defer client.Close()

	// Аутентификация (если есть учетные данные)
	if smtpUser != "" && smtpPass != "" {
		auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	// Создаем сообщение
	var msg bytes.Buffer
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	// Отправляем письмо
	err = client.SendMail(
		smtpUser,  // от кого
		[]string{to},  // кому
		bytes.NewReader(msg.Bytes()),
	if err != nil {
		return err
	}

	return nil
}