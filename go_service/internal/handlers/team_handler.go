package handlers

import (
	"net/http"
	"strings"

	"go_service/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TeamHandler struct {
	db *gorm.DB
}

func NewTeamHandler(db *gorm.DB) *TeamHandler {
	return &TeamHandler{db: db}
}

// GetTeams returns all teams (for authenticated users)
func (h *TeamHandler) GetTeams(c *gin.Context) {
	var teams []models.Team
	if err := h.db.Find(&teams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
		return
	}

	c.JSON(http.StatusOK, teams)
}

// CreateTeam creates a new team (only managers can create teams)
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	// Check if user is a manager
	role, exists := c.Get("role")
	roleStr, ok := role.(string)
	if !exists || !ok || strings.ToUpper(roleStr) != "MANAGER" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only managers can create teams"})
		return
	}

	var req struct {
		TeamName string `json:"teamName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := h.db.Begin()

	// Create team (teamId will be auto-generated as INTEGER)
	team := models.Team{
		TeamName: req.TeamName,
	}

	if err := tx.Create(&team).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create team"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Team created successfully",
		"team":    team,
	})
}
