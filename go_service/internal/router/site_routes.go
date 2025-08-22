package router

import (
	"go_service/internal/handlers"
	"go_service/internal/kafka"
	"go_service/internal/middleware"
	"go_service/internal/redisclient"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
)

func SetupRouter(router *gin.Engine, db *gorm.DB, producer *kafka.Producer, redis_client *redisclient.TeamCache) {
	// Create handlers
	teamHandler := handlers.NewTeamHandler(db, producer, redis_client)
	folderHandler := handlers.NewFolderHandler(db)
	noteHandler := handlers.NewNoteHandler(db)
	importHandler := handlers.NewImportHandler()

	//v1 api
	v1 := router.Group("/api/v1")

	protectedRoutes := v1.Group("/")
	protectedRoutes.Use(middleware.AuthMiddleware(db))

	// Set up all routes
	TeamRoutes(protectedRoutes, teamHandler)
	FolderRoutes(protectedRoutes, folderHandler, noteHandler)
	NoteRoutes(protectedRoutes, noteHandler)
	ImportRoutes(protectedRoutes, importHandler)
}
