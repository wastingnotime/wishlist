package main

import (
	"log"
	"net/http"
	"os"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/httpapi"
	"github.com/wastingnotime/wishlist/apps/api/internal/infrastructure"
)

func main() {
	board, err := infrastructure.SampleBoard()
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("WISHLIST_API_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	server := &http.Server{Addr: addr, Handler: httpapi.New(application.NewBoard(board)).Handler()}
	log.Printf("Wishlist API listening on http://%s", addr)
	log.Fatal(server.ListenAndServe())
}
