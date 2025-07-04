package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"deadline-aggregator/internal/handlers"
	"deadline-aggregator/internal/scheduler"
	"deadline-aggregator/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, continuing with system env")
	}

	db, err := store.NewPostgres()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := store.RunMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	go scheduler.StartReminderScheduler(db)

	r := gin.Default()
	handlers.SetupRoutes(r, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on http://localhost:%s", port)
	
	log.Fatal(r.Run(":" + port))
}
