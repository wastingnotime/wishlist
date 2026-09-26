package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/httpapi"
	"github.com/wastingnotime/wishlist/apps/api/internal/infrastructure"
)

func main() {
	seed, err := infrastructure.SampleBoard()
	if err != nil {
		log.Fatal(err)
	}
	databaseURL := os.Getenv("WISHLIST_DATABASE_URL")
	store, err := infrastructure.OpenPostgres(databaseURL, seed)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	production := os.Getenv("APP_ENV") == "production"
	if production {
		log.Fatal("production OTP email delivery is not configured; refuse to start with the development code logger")
	}
	if mode := os.Getenv("WISHLIST_OTP_MODE"); mode != "" && mode != "log" {
		log.Fatalf("unsupported WISHLIST_OTP_MODE %q; configure a production email adapter before using another mode", mode)
	}
	secret := os.Getenv("WISHLIST_OTP_SECRET")
	if secret == "" {
		secret = "local-development-secret-change-before-deploy"
		log.Print("using local OTP digest secret; set WISHLIST_OTP_SECRET outside development")
	}
	visitor := application.NewVisitor(store, infrastructure.LocalLoggingOTPSender{Logger: log.Default()}, systemClock{}, secret)
	board := application.NewBoard(store)
	addr := os.Getenv("WISHLIST_API_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
		if production {
			addr = ":8080"
		}
	}
	secureCookies := production
	adminToken := os.Getenv("WISHLIST_ADMIN_TOKEN")
	if secureCookies && adminToken == "" {
		log.Fatal("WISHLIST_ADMIN_TOKEN is required in production")
	}
	if adminToken == "" {
		adminToken = "local-development-admin-token"
		_ = os.Setenv("WISHLIST_ADMIN_TOKEN", adminToken)
		log.Print("using local development admin token; configure WISHLIST_ADMIN_TOKEN before deployment")
	}
	server := &http.Server{Addr: addr, Handler: httpapi.New(board, visitor, secureCookies, store).Handler()}
	log.Printf("Wishlist API listening on http://%s", addr)
	log.Fatal(server.ListenAndServe())
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }
