package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		os.Getenv("DB_PASS"),
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

	// Create a new HTTP server with the Gin router
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	//add graceful shutdown
	// Start the server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	// Listen for SIGINT and SIGTERM signals
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to do a graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
