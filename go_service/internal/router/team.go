package router

import (
	"go_service/internal/handlers"

	"github.com/gin-gonic/gin"
)

// TeamRoutes sets up routes for team-related operations
func TeamRoutes(rg *gin.RouterGroup, h *handlers.TeamHandler) {
	teams := rg.Group("/teams")
	{
		teams.POST("", h.CreateTeam)
		teams.POST("/:teamId/members", h.AddMemberToTeam)
		teams.DELETE("/:teamId/members/:memberId", h.RemoveMemberFromTeam)
		teams.POST("/:teamId/managers", h.AddManagerToTeam)
		teams.DELETE("/:teamId/managers/:managerId", h.RemoveManagerFromTeam)
		teams.GET("/:teamId/assets", h.GetTeamAssets)
	}

	// User assets route - manager only
	users := rg.Group("/users")
	{
		users.GET("/:userId/assets", h.GetUserAssets)
	}
}
