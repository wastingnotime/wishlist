package infrastructure

import (
	"context"
	"log"
)

// LocalLoggingOTPSender is only for development. It never logs the recipient address.
type LocalLoggingOTPSender struct{ Logger *log.Logger }

func (sender LocalLoggingOTPSender) Send(ctx context.Context, _ string, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sender.Logger.Printf("Wishlist local OTP code: %s", code)
	return nil
}
