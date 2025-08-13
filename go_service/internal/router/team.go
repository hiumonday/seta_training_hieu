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
		teams.POST("/:teamId/members", h.AddMemberToTeam)
		teams.DELETE("/:teamId/members/:memberId", h.RemoveMemberFromTeam)
		teams.POST("/:teamId/managers", h.AddManagerToTeam)
		teams.DELETE("/:teamId/managers/:managerId", h.RemoveManagerFromTeam)
	}
}
