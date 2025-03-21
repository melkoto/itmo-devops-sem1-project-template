package main

import (
	"log"
	"net/http"
	"os"
	"project_sem/internal/handler"
	"project_sem/internal/repository"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Using system environment variables.")
	}

	repo, err := repository.NewPostgresRepo()
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	defer repo.Close()

	priceHandler := handler.NewPriceHandler(repo)

	router := mux.NewRouter()
	router.HandleFunc("/api/v0/prices", priceHandler.UploadPrices).Methods("POST")
	router.HandleFunc("/api/v0/prices", priceHandler.DownloadPrices).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
