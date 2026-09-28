package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
	"github.com/wastingnotime/wishlist/apps/api/internal/httpapi"
	"github.com/wastingnotime/wishlist/apps/api/internal/infrastructure"
)

func main() {
	production := os.Getenv("APP_ENV") == "production"
	var otpSender application.OTPSender
	if production {
		var err error
		otpSender, err = infrastructure.NewSMTPOTPSenderFromEnv()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if mode := os.Getenv("WISHLIST_OTP_MODE"); mode != "" && mode != "log" {
			log.Fatalf("unsupported WISHLIST_OTP_MODE %q", mode)
		}
		otpSender = infrastructure.LocalLoggingOTPSender{Logger: log.Default()}
	}
	seed, err := startupSeed(production)
	if err != nil {
		log.Fatal(err)
	}
	databaseURL := os.Getenv("WISHLIST_DATABASE_URL")
	store, err := infrastructure.OpenPostgres(databaseURL, seed)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	casdoorAdmin := production || os.Getenv("WISHLIST_ADMIN_AUTH_MODE") == "casdoor"
	var adminOIDC *httpapi.AdminOIDC
	if casdoorAdmin {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		adminOIDC, err = httpapi.NewAdminOIDC(ctx,
			os.Getenv("WISHLIST_OIDC_DISCOVERY_URL"),
			os.Getenv("WISHLIST_OIDC_ISSUER"),
			os.Getenv("WISHLIST_OIDC_AUDIENCE"),
			os.Getenv("WISHLIST_ADMIN_SUBJECTS"))
		cancel()
		if err != nil {
			log.Fatal(err)
		}
	}
	secret := os.Getenv("WISHLIST_OTP_SECRET")
	if secret == "" {
		if production {
			log.Fatal("WISHLIST_OTP_SECRET is required in production")
		}
		secret = "local-development-secret-change-before-deploy"
		log.Print("using local OTP digest secret; set WISHLIST_OTP_SECRET outside development")
	}
	visitor := application.NewVisitor(store, otpSender, systemClock{}, secret)
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
	if !casdoorAdmin && adminToken == "" {
		adminToken = "local-development-admin-token"
		_ = os.Setenv("WISHLIST_ADMIN_TOKEN", adminToken)
		log.Print("using local development admin token; configure WISHLIST_ADMIN_TOKEN before deployment")
	}
	apiServer := httpapi.New(board, visitor, secureCookies, store)
	if casdoorAdmin {
		apiServer.UseAdminOIDC(adminOIDC)
	}
	server := &http.Server{Addr: addr, Handler: apiServer.Handler()}
	log.Printf("Wishlist API listening on http://%s", addr)
	log.Fatal(server.ListenAndServe())
}

func startupSeed(production bool) (domain.Board, error) {
	if production {
		return domain.NewBoard(nil, nil)
	}
	return infrastructure.SampleBoard()
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }
