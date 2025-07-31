package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go_service/internal/database"
	"go_service/internal/handlers"
	"go_service/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"math/rand"

	ginprometheus "github.com/zsais/go-gin-prometheus"
)

var logger zerolog.Logger

func setupLogger() zerolog.Logger {
	// Đảm bảo thư mục tồn tại
	os.MkdirAll("logs", os.ModePerm)

	file, err := os.OpenFile("logs/server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	multi := zerolog.MultiLevelWriter(os.Stdout, file)

	logger := zerolog.New(multi).With().Timestamp().Logger()
	return logger
}

func genRandom() time.Duration {
	rand.Seed(time.Now().UnixNano())

	// Generate a random number between 1 and 5
	randomNumber := rand.Intn(300) + 1 // 1 to 300

	return time.Duration(randomNumber) * time.Millisecond
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	logger = setupLogger()

	// Setup Gin router
	r := gin.Default()

	// NewWithConfig is the recommended way to initialize the middleware
	p := ginprometheus.NewWithConfig(ginprometheus.Config{
		Subsystem: "gin",
	})

	p.Use(r)

	// Ghi log mọi request
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Msg("Incoming request")
	})

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Initialize handlers
	teamHandler := handlers.NewTeamHandler(db)
	assetHandler := handlers.NewAssetHandler(db)

	// Test routes
	r.GET("/", func(c *gin.Context) {
		start := time.Now()
		time.Sleep(genRandom())
		c.JSON(200, "Hello world!")
		end := time.Now()
		fmt.Printf("Path: /, Request duration: %f seconds\n", end.Sub(start).Seconds())
	})
	r.GET("/ping", func(c *gin.Context) {
		start := time.Now()
		time.Sleep(genRandom())
		c.JSON(200, gin.H{
			"message": "pong",
		})
		end := time.Now()
		fmt.Printf("Path: /ping, Request duration: %f seconds\n", end.Sub(start).Seconds())
	})
	r.GET("/health", func(c *gin.Context) {
		start := time.Now()
		time.Sleep(genRandom())
		c.JSON(200, gin.H{
			"status": "ok",
		})
		end := time.Now()
		fmt.Printf("Path: /health, Request duration: %f seconds\n", end.Sub(start).Seconds())
	})

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware(db))
	{

		// Team routes
		protected.GET("/teams", func(c *gin.Context) {
			start := time.Now()
			time.Sleep(genRandom())
			logger.Info().Msg("Get teams endpoint called")
			teamHandler.GetTeams(c)
			end := time.Now()
			fmt.Printf("Path: /, Request duration: %f seconds\n", end.Sub(start).Seconds())
		})

		protected.POST("/teams", func(c *gin.Context) {
			start := time.Now()
			time.Sleep(genRandom())
			logger.Info().Msg("Create team endpoint called")
			teamHandler.CreateTeam(c)
			end := time.Now()
			fmt.Printf("Path: /, Request duration: %f seconds\n", end.Sub(start).Seconds())
		})
		protected.GET("/teams/:teamId", teamHandler.GetTeam)
		protected.PUT("/teams/:teamId", teamHandler.UpdateTeam)
		protected.DELETE("/teams/:teamId", teamHandler.DeleteTeam)

		// Team member management
		protected.POST("/teams/:teamId/members", teamHandler.AddMemberToTeam)
		protected.DELETE("/teams/:teamId/members/:memberId", teamHandler.RemoveMemberFromTeam)
		protected.POST("/teams/:teamId/managers", teamHandler.AddManagerToTeam)
		protected.DELETE("/teams/:teamId/managers/:managerId", teamHandler.RemoveManagerFromTeam)

		// Folder management
		protected.POST("/folders", assetHandler.CreateFolder)
		protected.GET("/folders/:folderId", assetHandler.GetFolder)
		protected.PUT("/folders/:folderId", assetHandler.UpdateFolder)
		protected.DELETE("/folders/:folderId", assetHandler.DeleteFolder)

		// Note management
		protected.POST("/folders/:folderId/notes", assetHandler.CreateNote)
		protected.GET("/notes/:noteId", assetHandler.GetNote)
		protected.PUT("/notes/:noteId", assetHandler.UpdateNote)
		protected.DELETE("/notes/:noteId", assetHandler.DeleteNote)

		// Sharing functionality
		protected.POST("/folders/:folderId/share", assetHandler.ShareFolder)
		protected.DELETE("/folders/:folderId/share/:userId", assetHandler.RevokeFolderShare)
		protected.POST("/notes/:noteId/share", assetHandler.ShareNote)
		protected.DELETE("/notes/:noteId/share/:userId", assetHandler.RevokeNoteShare)

		// Manager-only routes
		protected.GET("/teams/:teamId/assets", assetHandler.GetTeamAssets)
		protected.GET("/users/:userId/assets", assetHandler.GetUserAssets)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	// add graceful shutdown
	log.Fatal(http.ListenAndServe(":"+port, r))
}
