package main

import (
	"log"
	"net/http"
	"os"

	"go_service/internal/database"
	"go_service/internal/handlers"
	"go_service/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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

	// Setup Gin router
	r := gin.Default()

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

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware(db))
	{

		// Team routes
		protected.GET("/teams", teamHandler.GetTeams)
		protected.POST("/teams", teamHandler.CreateTeam)
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
