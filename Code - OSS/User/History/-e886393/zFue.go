package common

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
)

func SendEmail(to, subject, body string) error {
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

	msg := "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body

	// Создаем настраиваемую TLS конфигурацию
	tlsconfig := &tls.Config{
		ServerName:         smtpHost,
		MinVersion:         tls.VersionTLS12, // Минимальная версия TLS 1.2
		InsecureSkipVerify: false,            // Отключаем небезопасную проверку сертификатов
	}

	// Добавляем системные корневые сертификаты
	certPool, err := x509.SystemCertPool()
	if err != nil {
		log.Printf("Failed to get system cert pool: %v", err)
		certPool = x509.NewCertPool()
	}

	// Загружаем пользовательские сертификаты из переменной окружения
	sslCertFile := os.Getenv("SSL_CERT_FILE")
	if sslCertFile != "" {
		certPEM, err := ioutil.ReadFile(sslCertFile)
		if err != nil {
			log.Printf("Failed to read cert file: %v", err)
		} else {
			// Парсим PEM-сертификаты
			for block, rest := pem.Decode(certPEM); block != nil; block, rest = pem.Decode(rest) {
				if block.Type == "CERTIFICATE" {
					cert, err := x509.ParseCertificate(block.Bytes)
					if err != nil {
						log.Printf("Failed to parse certificate: %v", err)
						continue
					}
					certPool.AddCert(cert)
				}
			}
		}
	}

	tlsconfig.RootCAs = certPool

	// Подключение с TLS
	conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, tlsconfig)
	if err != nil {
		log.Printf("TLS connection error: %v", err)
		return err
	}
	defer conn.Close()

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
