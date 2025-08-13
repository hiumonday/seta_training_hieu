package router

import (
	"go_service/internal/handlers"

	"github.com/gin-gonic/gin"
)

// ROutes cho team
func TeamRoutes(rg *gin.RouterGroup, h *handlers.TeamHandler) {
	teams := rg.Group("/teams")
	{
		teams.POST("", h.CreateTeam)

	}
}
