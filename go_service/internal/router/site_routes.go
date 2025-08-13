package router

import (
	"go_service/internal/handlers"
	"go_service/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(router *gin.Engine, db *gorm.DB, teamHandler *handlers.TeamHandler) {

	//v1 api
	v1 := router.Group("/api/v1")

	protectedRoutes := v1.Group("/")
	protectedRoutes.Use(middleware.AuthMiddleware(db))

	TeamRoutes(protectedRoutes, teamHandler)

}
