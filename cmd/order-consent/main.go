package main

import (
	"log"
	"net/http"
	"os"

	consent "github.com/infrai-examples/order-consent-service"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := consent.NewClient(consent.DefaultBaseURL, key, nil)
	server := &http.Server{
		Addr:              ":8080",
		Handler:           consent.NewService(client),
		ReadHeaderTimeout: consentHeaderTimeout,
	}
	log.Printf("order consent service listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

const consentHeaderTimeout = 5_000_000_000
