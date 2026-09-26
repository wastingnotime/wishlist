package main

import (
	"log"
	"os"

	"github.com/wastingnotime/wishlist/apps/api/internal/infrastructure"
)

func main() {
	seed, err := infrastructure.SampleBoard()
	if err != nil {
		log.Fatal(err)
	}
	store, err := infrastructure.OpenPostgres(os.Getenv("WISHLIST_DATABASE_URL"), seed)
	if err != nil {
		log.Fatal(err)
	}
	if err := store.Close(); err != nil {
		log.Fatal(err)
	}
}
