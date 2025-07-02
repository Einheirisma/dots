package common

import (
	"crypto/tls"
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

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpHost,
	}

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

	if err = client.Auth(auth); err != nil {
		log.Printf("SMTP auth error: %v", err)
		return err
	}

	if err = client.Mail(from); err != nil {
		log.Printf("Mail from error: %v", err)
		return err
	}

	if err = client.Rcpt(to); err != nil {
		log.Printf("Rcpt to error: %v", err)
		return err
	}

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
