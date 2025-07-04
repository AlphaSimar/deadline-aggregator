package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/AlphaSimar/deadline-aggregator/internal/handlers"
	"github.com/AlphaSimar/deadline-aggregator/internal/scheduler"
	"github.com/AlphaSimar/deadline-aggregator/internal/store"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, continuing with system env")
	}

	// Connect to DB
	db, err := store.NewPostgres()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Run DB migrations
	if err := store.RunMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Start scheduler
	go scheduler.StartReminderScheduler(db)

	// Setup HTTP server
	r := gin.Default()
	handlers.SetupRoutes(r, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on http://localhost:%s", port)
	log.Fatal(r.Run(":" + port))
}
