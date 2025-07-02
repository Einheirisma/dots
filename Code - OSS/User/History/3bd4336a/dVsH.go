package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationMessage struct {
	Channel   string            `json:"channel"`
	Recipient string            `json:"recipient"` // WhatsApp номер в формате 79123456789
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
}

type GreenAPIResponse struct {
	IdMessage string `json:"idMessage"` // Для отслеживания статуса
}

func main() {
	// Получаем учетные данные из .env
	idInstance := os.Getenv("GREEN_API_INSTANCE")
	apiToken := os.Getenv("GREEN_API_TOKEN")

	if idInstance == "" || apiToken == "" {
		log.Fatal("GREEN_API_INSTANCE or GREEN_API_TOKEN is not set")
	}

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
		"whatsapp_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatal("Failed to declare a queue:", err)
	}

	log.Printf("WhatsApp queue declared: %s (%d messages, %d consumers)",
		q.Name, q.Messages, q.Consumers)

	// Привязываем очередь к exchange
	err = ch.QueueBind(
		q.Name,
		"whatsapp",
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

	log.Printf("WhatsApp worker started. Waiting for messages...")

	// Обработка сообщений
	for d := range msgs {
		log.Printf("Received a message: %s", d.Body)

		var msg NotificationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %s", err)
			d.Nack(false, false) // Отбрасываем сообщение
			continue
		}

		// Отправка сообщения через Green-API
		if err := sendWhatsAppMessage(idInstance, apiToken, msg); err != nil {
			log.Printf("Failed to send WhatsApp message: %s", err)
			d.Nack(false, true) // Возвращаем в очередь для повторной попытки
			continue
		}

		log.Printf("Message sent to %s", msg.Recipient)
		d.Ack(false) // Подтверждаем обработку
	}
}

func sendWhatsAppMessage(idInstance, apiToken string, msg NotificationMessage) error {
	// Формируем URL для отправки сообщения
	url := fmt.Sprintf("https://api.green-api.com/waInstance%s/sendMessage/%s", idInstance, apiToken)

	// Формируем тело запроса
	payload := map[string]interface{}{
		"chatId":  msg.Recipient + "@c.us", // Преобразуем номер в chatId
		"message": msg.Body,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Отправляем POST-запрос
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return fmt.Errorf("green-api error (status %d): failed to decode error response", resp.StatusCode)
		}
		return fmt.Errorf("green-api error (status %d): %v", resp.StatusCode, errorResponse)
	}

	// Декодируем ответ
	var result GreenAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("Message ID: %s", result.IdMessage)
	return nil
}
