package handlers

import (
	"net/http"
	"strconv"

	"go_service/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	if !exists || role != string(models.Manager) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only managers can create teams"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	creatorID := userID.(uuid.UUID)

	var req struct {
		TeamName string `json:"teamName" binding:"required"`
		Managers []struct {
			ManagerID uuid.UUID `json:"managerId"`
		} `json:"managers"`
		Members []struct {
			MemberID uuid.UUID `json:"memberId"`
		} `json:"members"`
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

	// Add creator as team leader in Roster table
	roster := models.Roster{
		TeamID:   team.ID,
		UserID:   creatorID,
		IsLeader: true,
	}

	if err := tx.Create(&roster).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to add team leader"})
		return
	}

	// Add additional managers as leaders
	for _, mgr := range req.Managers {
		var manager models.User
		if err := tx.First(&manager, "userId = ? AND role = ?", mgr.ManagerID, models.Manager).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Manager not found or invalid role"})
			return
		}

		managerRoster := models.Roster{
			TeamID:   team.ID,
			UserID:   mgr.ManagerID,
			IsLeader: true,
		}

		if err := tx.Create(&managerRoster).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add manager"})
			return
		}
	}

	// Add members
	for _, mbr := range req.Members {
		var member models.User
		if err := tx.First(&member, "userId = ?", mbr.MemberID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Member not found"})
			return
		}

		memberRoster := models.Roster{
			TeamID:   team.ID,
			UserID:   mbr.MemberID,
			IsLeader: false,
		}

		if err := tx.Create(&memberRoster).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member"})
			return
		}
	}

	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Team created successfully",
		"team":    team,
	})
}

// GetTeam returns a specific team with members
func (h *TeamHandler) GetTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	var team models.Team
	if err := h.db.First(&team, "teamId = ?", teamID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}

	// Get team members through Roster junction table
	var rosters []models.Roster
	if err := h.db.Preload("User").Where("teamId = ?", teamID).Find(&rosters).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch team members"})
		return
	}

	// Separate managers and members
	var managers []models.User
	var members []models.User

	for _, roster := range rosters {
		if roster.IsLeader {
			managers = append(managers, roster.User)
		} else {
			members = append(members, roster.User)
		}
	}

	response := gin.H{
		"team":     team,
		"managers": managers,
		"members":  members,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateTeam updates team information (only team managers can update)
func (h *TeamHandler) UpdateTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	// Check if current user is a manager of this team
	if !h.isTeamManager(currentUserID, teamID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only team managers can update team"})
		return
	}

	var req struct {
		TeamName string `json:"teamName"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var team models.Team
	if err := h.db.First(&team, "teamId = ?", teamID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}

	if req.TeamName != "" {
		team.TeamName = req.TeamName
	}

	if err := h.db.Save(&team).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update team"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Team updated successfully",
		"team":    team,
	})
}

// DeleteTeam deletes a team (only team managers can delete)
func (h *TeamHandler) DeleteTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	// Check if current user is a manager of this team
	if !h.isTeamManager(currentUserID, teamID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only team managers can delete team"})
		return
	}

	tx := h.db.Begin()

	// Delete all rosters first (cascade delete)
	if err := tx.Where("teamId = ?", teamID).Delete(&models.Roster{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete team rosters"})
		return
	}

	// Delete the team
	if err := tx.Where("teamId = ?", teamID).Delete(&models.Team{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete team"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Team deleted successfully"})
}

// AddMemberToTeam adds a member to a team
func (h *TeamHandler) AddMemberToTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	// Check if current user is a manager of this team
	if !h.isTeamManager(currentUserID, teamID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only team managers can add members"})
		return
	}

	var req struct {
		MemberID uuid.UUID `json:"memberId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var member models.User
	if err := h.db.First(&member, "userId = ?", req.MemberID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Member not found"})
		return
	}

	// Check if already in team
	var existingRoster models.Roster
	if err := h.db.Where("teamId = ? AND userId = ?", teamID, req.MemberID).First(&existingRoster).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User is already in this team"})
		return
	}

	// Add member
	newRoster := models.Roster{
		TeamID:   teamID,
		UserID:   req.MemberID,
		IsLeader: false,
	}

	if err := h.db.Create(&newRoster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added successfully"})
}

// RemoveMemberFromTeam removes a member from a team
func (h *TeamHandler) RemoveMemberFromTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	// Check if current user is a manager of this team
	if !h.isTeamManager(currentUserID, teamID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only team managers can remove members"})
		return
	}

	// Remove member from team
	if err := h.db.Where("teamId = ? AND userId = ?", teamID, memberID).Delete(&models.Roster{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// AddManagerToTeam adds a manager to a team
func (h *TeamHandler) AddManagerToTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	// Check if current user is a manager of this team
	if !h.isTeamManager(currentUserID, teamID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only team managers can add other managers"})
		return
	}

	var req struct {
		ManagerID uuid.UUID `json:"managerId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists and is a manager
	var manager models.User
	if err := h.db.First(&manager, "userId = ? AND role = ?", req.ManagerID, models.Manager).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manager not found or user is not a manager"})
		return
	}

	// Check if already in team
	var existingRoster models.Roster
	if err := h.db.Where("teamId = ? AND userId = ?", teamID, req.ManagerID).First(&existingRoster).Error; err == nil {
		// Update to manager if already member
		existingRoster.IsLeader = true
		if err := h.db.Save(&existingRoster).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote to manager"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User promoted to manager successfully"})
		return
	}

	// Add as new manager
	newRoster := models.Roster{
		TeamID:   teamID,
		UserID:   req.ManagerID,
		IsLeader: true,
	}

	if err := h.db.Create(&newRoster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add manager"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Manager added successfully"})
}

// RemoveManagerFromTeam removes a manager from a team
func (h *TeamHandler) RemoveManagerFromTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	managerIDStr := c.Param("managerId")
	managerID, err := uuid.Parse(managerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid manager ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	// Check if current user is a manager of this team
	if !h.isTeamManager(currentUserID, teamID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only team managers can remove other managers"})
		return
	}

	// Prevent removing yourself if you're the only manager
	var managerCount int64
	h.db.Model(&models.Roster{}).Where("teamId = ? AND isLeader = ?", teamID, true).Count(&managerCount)

	if managerCount <= 1 && currentUserID == managerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove the last manager from team"})
		return
	}

	// Remove manager from team (or demote to member)
	var managerRoster models.Roster
	if err := h.db.Where("teamId = ? AND userId = ? AND isLeader = ?", teamID, managerID, true).First(&managerRoster).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Manager not found in this team"})
		return
	}

	// Demote to member instead of removing completely
	managerRoster.IsLeader = false
	if err := h.db.Save(&managerRoster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove manager"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Manager removed successfully"})
}

// Helper function to check if user is team manager
func (h *TeamHandler) isTeamManager(userID uuid.UUID, teamID int) bool {
	var roster models.Roster
	return h.db.Where("teamId = ? AND userId = ? AND isLeader = ?", teamID, userID, true).First(&roster).Error == nil
}
