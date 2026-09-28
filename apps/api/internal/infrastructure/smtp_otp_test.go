package infrastructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSMTPOTPSenderFromEnvReadsMountedSecret(t *testing.T) {
	passwordFile := filepath.Join(t.TempDir(), "smtp-password")
	if err := os.WriteFile(passwordFile, []byte("test-password"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WNT_EMAIL_SMTP_HOST", "smtp.example.test")
	t.Setenv("WNT_EMAIL_SMTP_PORT", "587")
	t.Setenv("WNT_EMAIL_SMTP_USERNAME", "noreply@example.test")
	t.Setenv("WNT_EMAIL_SMTP_PASSWORD_FILE", passwordFile)
	t.Setenv("WNT_EMAIL_FROM", "noreply@example.test")
	t.Setenv("WNT_EMAIL_REPLY_TO", "")

	sender, err := NewSMTPOTPSenderFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if sender.password != "test-password" || sender.host != "smtp.example.test" || sender.port != "587" {
		t.Fatalf("unexpected sender configuration: host=%q port=%q", sender.host, sender.port)
	}
}

func TestNewSMTPOTPSenderFromEnvRejectsIncompleteConfiguration(t *testing.T) {
	for _, key := range []string{
		"WNT_EMAIL_SMTP_HOST", "WNT_EMAIL_SMTP_PORT", "WNT_EMAIL_SMTP_USERNAME",
		"WNT_EMAIL_SMTP_PASSWORD_FILE", "WNT_EMAIL_FROM", "WNT_EMAIL_REPLY_TO",
	} {
		t.Setenv(key, "")
	}
	if _, err := NewSMTPOTPSenderFromEnv(); err == nil {
		t.Fatal("expected incomplete SMTP configuration to fail")
	}
}

func TestSMTPMessageUsesContractSenderRecipientAndExpiresCode(t *testing.T) {
	sender := &SMTPOTPSender{from: "noreply@example.test", replyTo: "support@example.test"}
	to, err := parseMailbox("person@example.test")
	if err != nil {
		t.Fatal(err)
	}
	message := sender.message(to, "123456")
	for _, expected := range []string{
		"From: noreply@example.test\r\n",
		"Reply-To: support@example.test\r\n",
		"To: person@example.test\r\n",
		"Subject: Your Wishlist sign-in code\r\n",
		"Your Wishlist sign-in code is 123456. It expires in 10 minutes.",
	} {
		if !strings.Contains(message, expected) {
			t.Errorf("message is missing %q", expected)
		}
	}
}
