package mailing

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

func Send(recipient string, subject string, body string) (bool, error) {
	settings, settingsError := Settings()
	if settingsError != nil {
		return false, settingsError
	}
	if !Configured(settings) {
		return false, nil
	}
	address, parseError := mail.ParseAddress(recipient)
	if parseError != nil {
		return false, parseError
	}
	password, passwordError := decrypt(settings.PasswordCipher)
	if passwordError != nil {
		return false, passwordError
	}
	target := net.JoinHostPort(settings.Host, strconv.Itoa(settings.Port))
	tlsSettings := &tls.Config{ServerName: settings.Host, MinVersion: tls.VersionTLS12}
	var connection net.Conn
	var connectError error
	if settings.Security == "tls" {
		connection, connectError = tls.DialWithDialer(&net.Dialer{Timeout: Timeout}, "tcp", target, tlsSettings)
	} else {
		connection, connectError = net.DialTimeout("tcp", target, Timeout)
	}
	if connectError != nil {
		return false, DeliveryFailed
	}
	defer connection.Close()
	connection.SetDeadline(time.Now().Add(Timeout))
	client, clientError := smtp.NewClient(connection, settings.Host)
	if clientError != nil {
		return false, DeliveryFailed
	}
	defer client.Close()
	if settings.Security == "starttls" {
		if startError := client.StartTLS(tlsSettings); startError != nil {
			return false, DeliveryFailed
		}
	}
	if settings.Username != "" {
		if authError := client.Auth(smtp.PlainAuth("", settings.Username, password, settings.Host)); authError != nil {
			return false, DeliveryFailed
		}
	}
	if sendError := client.Mail(settings.FromEmail); sendError != nil {
		return false, DeliveryFailed
	}
	if sendError := client.Rcpt(address.Address); sendError != nil {
		return false, DeliveryFailed
	}
	writer, writerError := client.Data()
	if writerError != nil {
		return false, DeliveryFailed
	}
	from := (&mail.Address{Name: settings.FromName, Address: settings.FromEmail}).String()
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", from, address.String(), mime.QEncoding.Encode("UTF-8", subject), strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"))
	if _, writeError := writer.Write([]byte(message)); writeError != nil {
		writer.Close()
		return false, DeliveryFailed
	}
	if closeError := writer.Close(); closeError != nil {
		return false, DeliveryFailed
	}
	if quitError := client.Quit(); quitError != nil {
		return false, DeliveryFailed
	}
	return true, nil
}
