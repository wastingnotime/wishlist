package infrastructure

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

const smtpSendTimeout = 10 * time.Second

// SMTPOTPSender delivers visitor codes through the platform SMTP contract.
// The password is read from the mounted Docker secret during startup and is
// never included in errors or logs.
type SMTPOTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
	replyTo  string
}

// NewSMTPOTPSenderFromEnv loads the infra-platform email contract from the
// process environment and the mounted secret file.
func NewSMTPOTPSenderFromEnv() (*SMTPOTPSender, error) {
	host := strings.TrimSpace(os.Getenv("WNT_EMAIL_SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("WNT_EMAIL_SMTP_PORT"))
	username := strings.TrimSpace(os.Getenv("WNT_EMAIL_SMTP_USERNAME"))
	passwordFile := strings.TrimSpace(os.Getenv("WNT_EMAIL_SMTP_PASSWORD_FILE"))
	from := strings.TrimSpace(os.Getenv("WNT_EMAIL_FROM"))
	replyTo := strings.TrimSpace(os.Getenv("WNT_EMAIL_REPLY_TO"))
	if host == "" || strings.ContainsAny(host, "\r\n") || port == "" || username == "" || passwordFile == "" || from == "" {
		return nil, fmt.Errorf("incomplete production SMTP configuration")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("invalid production SMTP port")
	}
	parsedUsername, err := parseMailbox(username)
	if err != nil {
		return nil, fmt.Errorf("invalid production SMTP username")
	}
	if _, err := parseMailbox(from); err != nil {
		return nil, fmt.Errorf("invalid production email sender")
	}
	if replyTo != "" {
		if _, err := parseMailbox(replyTo); err != nil {
			return nil, fmt.Errorf("invalid production reply-to address")
		}
	}
	password, err := os.ReadFile(passwordFile)
	if err != nil || len(password) == 0 {
		return nil, fmt.Errorf("could not read production SMTP password secret")
	}
	return &SMTPOTPSender{
		host: host, port: port, username: parsedUsername.Address,
		password: string(password), from: from, replyTo: replyTo,
	}, nil
}

func (sender *SMTPOTPSender) Send(ctx context.Context, recipient, code string) error {
	to, err := parseMailbox(recipient)
	if err != nil {
		return fmt.Errorf("invalid OTP recipient")
	}
	from, _ := parseMailbox(sender.from) // validated during construction
	ctx, cancel := context.WithTimeout(ctx, smtpSendTimeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(sender.host, sender.port))
	if err != nil {
		return fmt.Errorf("connect to OTP email provider: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	stopClose := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stopClose()

	client, err := smtp.NewClient(conn, sender.host)
	if err != nil {
		return fmt.Errorf("start OTP email session: %w", err)
	}
	defer client.Close()
	if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: sender.host}); err != nil {
		return fmt.Errorf("establish secure OTP email session: %w", err)
	}
	if err := client.Auth(smtp.PlainAuth("", sender.username, sender.password, sender.host)); err != nil {
		return fmt.Errorf("authenticate OTP email session: %w", err)
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("set OTP email sender: %w", err)
	}
	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("set OTP email recipient: %w", err)
	}
	body, err := client.Data()
	if err != nil {
		return fmt.Errorf("start OTP email message: %w", err)
	}
	if _, err := io.WriteString(body, sender.message(to, code)); err != nil {
		_ = body.Close()
		return fmt.Errorf("write OTP email message: %w", err)
	}
	if err := body.Close(); err != nil {
		return fmt.Errorf("send OTP email message: %w", err)
	}
	return nil
}

func (sender *SMTPOTPSender) message(to *mail.Address, code string) string {
	from, _ := parseMailbox(sender.from)
	var message strings.Builder
	fmt.Fprintf(&message, "From: %s\r\n", from.Address)
	if sender.replyTo != "" {
		replyTo, _ := parseMailbox(sender.replyTo)
		fmt.Fprintf(&message, "Reply-To: %s\r\n", replyTo.Address)
	}
	fmt.Fprintf(&message, "To: %s\r\n", to.Address)
	message.WriteString("Subject: Your Wishlist sign-in code\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	fmt.Fprintf(&message, "Your Wishlist sign-in code is %s. It expires in 10 minutes.\r\n", code)
	message.WriteString("If you did not request this code, you can ignore this email.\r\n")
	return message.String()
}

func parseMailbox(value string) (*mail.Address, error) {
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address == "" || strings.ContainsAny(address.Address, "\r\n") {
		return nil, fmt.Errorf("invalid email address")
	}
	return address, nil
}
