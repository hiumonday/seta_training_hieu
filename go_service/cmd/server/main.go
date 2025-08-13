package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"go_service/internal/database"
	"go_service/internal/handlers"
	"go_service/internal/middleware"
	"go_service/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"), // Match Node.js service env var name
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)
	log.Printf("dsn: %s", dsn)
	db, err := database.Connect(dsn)
	if err != nil {
		log.Fatalf("Fail to connect DB: %v", err)
	}

	teamHandler := handlers.NewTeamHandler(db)

	// Setup Gin router
	r := gin.Default()
	r.Use(middleware.AuthMiddleware(db))
	router.SetupRouter(r, db, teamHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	// add graceful shutdown
	log.Fatal(http.ListenAndServe(":"+port, r))
}
