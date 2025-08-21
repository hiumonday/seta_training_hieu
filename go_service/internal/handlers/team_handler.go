package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"go_service/internal/kafka"
	"go_service/internal/models"
	"go_service/internal/redisclient"
	"go_service/internal/services"
	"go_service/pkg/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamHandler struct {
	db          *gorm.DB
	userService *services.UserService
	producer    *kafka.Producer
	redisClient *redisclient.TeamCache
}

func NewTeamHandler(db *gorm.DB, producer *kafka.Producer, redisClient *redisclient.TeamCache) *TeamHandler {
	return &TeamHandler{
		db:          db,
		userService: services.NewUserService(),
		producer:    producer,
		redisClient: redisClient,
	}
}

// CreateTeam creates a new team with members (only managers can create teams)
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	// Lấy vai trò từ context để kiểm tra quyền
	role, exists := c.Get("role")
	roleStr, ok := role.(string)
	if !exists || !ok || strings.ToUpper(roleStr) != "MANAGER" {
		log.Println("Non-manager user tried to create a team")
		c.JSON(http.StatusForbidden, responses.NewErrorResponse("Only managers can create teams", ""))
		return
	}

	var req struct {
		TeamName string      `json:"teamName" binding:"required"`
		UserIDs  []uuid.UUID `json:"userIds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("Invalid request body for CreateTeam")
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse(err.Error(), "Invalid request body"))
		return
	}

	tx := h.db.Begin()

	// Tạo team mới
	team := models.Team{
		TeamName: req.TeamName,
	}

	if err := tx.Create(&team).Error; err != nil {
		tx.Rollback()
		log.Println("Failed to create team")
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Failed to create team", ""))
		return
	}

	// Thêm người tạo vào team và đặt làm leader
	userID, exists := c.Get("user_id")
	if !exists {
		tx.Rollback()
		log.Println("User ID not found in context")
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("User not authenticated", "Missing user ID in context"))
		return
	}

	creatorID := userID.(uuid.UUID)

	// Thêm người tạo làm leader
	leaderRoster := models.Roster{
		TeamID:   team.ID,
		UserID:   creatorID,
		IsLeader: true,
	}

	if err := tx.Create(&leaderRoster).Error; err != nil {
		tx.Rollback()
		log.Println("Failed to create team leader roster entry")
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to set team leader", ""))
		return
	}

	// Thêm các thành viên khác vào team
	addedMembers := []uuid.UUID{creatorID} // Đã thêm người tạo
	failedMembers := []uuid.UUID{}

	for _, memberID := range req.UserIDs {
		// Kiểm tra nếu userID đã được thêm vào (tránh thêm trùng)
		alreadyAdded := false
		for _, id := range addedMembers {
			if id == memberID {
				alreadyAdded = true
				break
			}
		}

		if alreadyAdded {
			continue
		}

		if _, err := h.userService.GetUserByID(memberID.String()); err != nil {
			// User không tồn tại hoặc có lỗi khi gọi service
			log.Printf("Failed to verify user %s: %v", memberID, err)
			failedMembers = append(failedMembers, memberID)
			continue
		}

		// Thêm user vào team
		roster := models.Roster{
			TeamID:   team.ID,
			UserID:   memberID,
			IsLeader: false,
		}

		if err := tx.Create(&roster).Error; err != nil {
			failedMembers = append(failedMembers, memberID)
			log.Printf("Failed to add member %s to team: %v", memberID, err)
			continue
		}

		addedMembers = append(addedMembers, memberID)
	}

	tx.Commit()

	response := gin.H{
		"message": "Team created successfully",
		"team":    team,
	}

	// Nếu có thành viên không được thêm vào, báo cáo lại
	if len(failedMembers) > 0 {
		response["warning"] = "Some members could not be added to the team"
		response["failedMembers"] = failedMembers
	}

	c.JSON(http.StatusCreated, responses.NewSuccessResponse("Team created successfully", response))
}

func (h *TeamHandler) AddMemberToTeam(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to add team member: missing user_id")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid team ID format: %s", teamIDStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid team ID format",
		})
		return
	}

	// Kiểm tra team tồn tại
	var team models.Team
	if err := h.db.First(&team, teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Team not found: %d", teamID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Team not found",
			})
			return
		}
		log.Printf("Database error when finding team: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify team",
		})
		return
	}

	// Kiểm tra xem người dùng hiện tại có phải là leader của team hay không
	var leaderRoster models.Roster
	if err := h.db.Where("\"teamId\" = ? AND \"userId\" = ? AND \"isLeader\" = ?", teamID, currentUserID, true).First(&leaderRoster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("User %s attempted to add member to team %d without leadership permission", currentUserID, teamID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Only team leaders can add members",
			})
			return
		}
		log.Printf("Database error when checking leadership: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify team leadership",
		})
		return
	}

	var req struct {
		UserIDs []uuid.UUID `json:"userIds" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Bắt đầu transaction
	tx := h.db.Begin()

	type AddResult struct {
		UserID     uuid.UUID `json:"userId"`
		Username   string    `json:"username"`
		Status     string    `json:"status"`
		StatusCode int       `json:"-"`
	}

	results := make([]AddResult, 0, len(req.UserIDs))
	addedCount := 0
	existingCount := 0
	failedCount := 0

	processedUsers := make(map[uuid.UUID]bool)

	for _, userID := range req.UserIDs {
		// Bỏ qua nếu đã xử lý userID này trước đó
		if processedUsers[userID] {
			results = append(results, AddResult{
				UserID:     userID,
				Status:     "duplicate_in_request",
				StatusCode: http.StatusBadRequest,
			})
			failedCount++
			continue
		}
		processedUsers[userID] = true

		// Kiểm tra user tồn tại qua service
		userResp, err := h.userService.GetUserByID(userID.String())
		if err != nil {
			results = append(results, AddResult{
				UserID:     userID,
				Status:     "user_not_found",
				StatusCode: http.StatusBadRequest,
			})
			failedCount++
			continue
		}

		// Kiểm tra xem user đã là thành viên của team chưa
		var existingMember models.Roster
		if err := h.db.Where("\"teamId\" = ? AND \"userId\" = ?", teamID, userID).First(&existingMember).Error; err == nil {
			results = append(results, AddResult{
				UserID:     userID,
				Username:   userResp.User.Username,
				Status:     "already_member",
				StatusCode: http.StatusConflict,
			})
			existingCount++
			continue
		} else if err != gorm.ErrRecordNotFound {
			log.Printf("Database error when checking existing membership: %v", err)
			results = append(results, AddResult{
				UserID:     userID,
				Status:     "error_verifying",
				StatusCode: http.StatusInternalServerError,
			})
			failedCount++
			continue
		}

		// Thêm user vào team
		roster := models.Roster{
			TeamID:   team.ID,
			UserID:   userID,
			IsLeader: false, // Mặc định là false
		}

		if err := tx.Create(&roster).Error; err != nil {
			log.Printf("Failed to add user %s to team %d: %v", userID, teamID, err)
			results = append(results, AddResult{
				UserID:     userID,
				Status:     "error_adding",
				StatusCode: http.StatusInternalServerError,
			})
			failedCount++
			continue
		}

		// Thêm thành công
		results = append(results, AddResult{
			UserID:     userID,
			Username:   userResp.User.Username,
			Status:     "added_successfully",
			StatusCode: http.StatusCreated,
		})
		addedCount++
	}

	// Rollback nếu không thêm được thành viên nào
	if addedCount == 0 {
		tx.Rollback()
		log.Printf("No members were added to team %d", teamID)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No members were added to the team",
			"results": results,
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save team members data",
		})
		return
	}
	//emit MEMBER_ADDED
	if addedCount > 0 && h.producer != nil {
		for _, result := range results {
			if result.Status == "added_successfully" {
				err := h.producer.SendTeamEvent(
					kafka.EventMemberAdded,
					teamID,
					currentUserID.(uuid.UUID),
					result.UserID,
				)
				if err != nil {
					log.Printf("Failed to send Kafka event for member addition: %v", err)
					// Continue processing even if Kafka event fails
				}
			}
		}
	}

	var responseStatus int
	var successValue bool

	if failedCount == 0 && existingCount == 0 {
		// Tất cả đều thành công
		responseStatus = http.StatusCreated
		successValue = true
	} else if addedCount > 0 {
		// Thêm một phần thành công
		responseStatus = http.StatusPartialContent // 206
		successValue = true
	} else {
		// Không thành công
		responseStatus = http.StatusBadRequest
		successValue = false
	}

	//  Trả về response
	c.JSON(responseStatus, gin.H{
		"success": successValue,
		"message": fmt.Sprintf("Processed %d members: %d added, %d already existing, %d failed",
			len(req.UserIDs), addedCount, existingCount, failedCount),
		"data": gin.H{
			"teamId":     team.ID,
			"teamName":   team.TeamName,
			"addedCount": addedCount,
			"totalCount": len(req.UserIDs),
			"results":    results,
		},
	})
}

// RemoveMemberFromTeam removes a member from a team
func (h *TeamHandler) RemoveMemberFromTeam(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to remove team member: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", "Missing user ID"))
		return
	}

	// Parse team ID
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid team ID format: %s", teamIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid team ID format", ""))
		return
	}

	// Parse member ID
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		log.Printf("Invalid member ID format: %s", memberIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid member ID format", ""))
		return
	}

	// Check if team exists
	var team models.Team
	if err := h.db.First(&team, teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Team not found: %d", teamID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Team not found", ""))
			return
		}
		log.Printf("Database error when finding team: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team", err.Error()))
		return
	}

	// Check if current user is team leader
	var leaderRoster models.Roster
	if err := h.db.Where("\"teamId\" = ? AND \"userId\" = ? AND \"isLeader\" = ?", teamID, currentUserID, true).First(&leaderRoster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("User %s attempted to remove member from team %d without leadership permission", currentUserID, teamID)
			c.JSON(http.StatusForbidden, responses.NewErrorResponse("Only team leaders can remove members", ""))
			return
		}
		log.Printf("Database error when checking leadership: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team leadership", err.Error()))
		return
	}

	// Check if member exists in the team
	var memberRoster models.Roster
	if err := h.db.Where("\"teamId\" = ? AND \"userId\" = ?", teamID, memberID).First(&memberRoster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Member %s not found in team %d", memberID, teamID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Member not found in this team", ""))
			return
		}
		log.Printf("Database error when checking member: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team membership", err.Error()))
		return
	}

	// Prevent removing yourself if you're the only leader
	if memberID == currentUserID.(uuid.UUID) && memberRoster.IsLeader {
		// Count team leaders
		var leaderCount int64
		if err := h.db.Model(&models.Roster{}).Where("\"teamId\" = ? AND \"isLeader\" = ?", teamID, true).Count(&leaderCount).Error; err != nil {
			log.Printf("Error counting team leaders: %v", err)
			c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team leadership status", err.Error()))
			return
		}

		if leaderCount <= 1 {
			log.Printf("User %s attempted to remove self as the only leader of team %d", currentUserID, teamID)
			c.JSON(http.StatusForbidden, responses.NewErrorResponse("Cannot remove yourself as the only team leader. Assign another leader first.", ""))
			return
		}
	}

	// Remove member
	if err := h.db.Delete(&memberRoster).Error; err != nil {
		log.Printf("Failed to remove member %s from team %d: %v", memberID, teamID, err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to remove member from team", err.Error()))
		return
	}

	if err := h.producer.SendTeamEvent(
		kafka.EventMemberRemoved,
		teamID,
		currentUserID.(uuid.UUID),
		memberID,
	); err != nil {
		log.Printf("Failed to send Kafka event for member removed: %v", err)
		// Continue processing even if Kafka event fails
	}

	// Fetch username for response
	userResp, _ := h.userService.GetUserByID(memberID.String())
	username := ""
	if userResp != nil && userResp.User.Username != "" {
		username = userResp.User.Username
	}

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Member removed from team successfully", gin.H{
		"teamId":    team.ID,
		"teamName":  team.TeamName,
		"memberId":  memberID,
		"username":  username,
		"wasLeader": memberRoster.IsLeader,
	}))
}

// AddManagerToTeam promotes a member to team manager/leader
func (h *TeamHandler) AddManagerToTeam(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to add team manager: missing user_id")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Parse team ID
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid team ID format: %s", teamIDStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid team ID format",
		})
		return
	}

	// Check if team exists
	var team models.Team
	if err := h.db.First(&team, teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Team not found: %d", teamID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Team not found",
			})
			return
		}
		log.Printf("Database error when finding team: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify team",
		})
		return
	}

	// Check if current user is team leader
	var leaderRoster models.Roster
	if err := h.db.Where("\"teamId\" = ? AND \"userId\" = ? AND \"isLeader\" = ?", teamID, currentUserID, true).First(&leaderRoster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("User %s attempted to add manager to team %d without leadership permission", currentUserID, teamID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Only team leaders can add managers",
			})
			return
		}
		log.Printf("Database error when checking leadership: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify team leadership",
		})
		return
	}

	// Parse request body
	var req struct {
		UserIDs []uuid.UUID `json:"userIds" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Process each user
	type PromoteResult struct {
		UserID   uuid.UUID `json:"userId"`
		Username string    `json:"username"`
		Status   string    `json:"status"`
	}

	results := make([]PromoteResult, 0, len(req.UserIDs))
	successCount := 0
	failedCount := 0

	for _, userID := range req.UserIDs {
		// Verify user exists
		userResp, err := h.userService.GetUserByID(userID.String())
		if err != nil {
			results = append(results, PromoteResult{
				UserID: userID,
				Status: "user_not_found",
			})
			failedCount++
			continue
		}

		// Check if user is in team
		var memberRoster models.Roster
		if err := h.db.Where("\"teamId\" = ? AND \"userId\" = ?", teamID, userID).First(&memberRoster).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				results = append(results, PromoteResult{
					UserID:   userID,
					Username: userResp.User.Username,
					Status:   "not_team_member",
				})
				failedCount++
				continue
			}

			log.Printf("Database error when checking team membership: %v", err)
			results = append(results, PromoteResult{
				UserID:   userID,
				Username: userResp.User.Username,
				Status:   "database_error",
			})
			failedCount++
			continue
		}

		// Check if already a leader
		if memberRoster.IsLeader {
			results = append(results, PromoteResult{
				UserID:   userID,
				Username: userResp.User.Username,
				Status:   "already_leader",
			})
			failedCount++
			continue
		}

		// Update to make the user a leader
		memberRoster.IsLeader = true
		if err := h.db.Save(&memberRoster).Error; err != nil {
			log.Printf("Failed to promote user %s to team leader: %v", userID, err)
			results = append(results, PromoteResult{
				UserID:   userID,
				Username: userResp.User.Username,
				Status:   "update_failed",
			})
			failedCount++
			continue
		}

		// Success
		results = append(results, PromoteResult{
			UserID:   userID,
			Username: userResp.User.Username,
			Status:   "promoted_to_leader",
		})
		successCount++
	}

	var responseStatus int
	var successValue bool

	if failedCount == 0 {
		responseStatus = http.StatusOK
		successValue = true
	} else if successCount > 0 {
		responseStatus = http.StatusPartialContent
		successValue = true
	} else {
		responseStatus = http.StatusBadRequest
		successValue = false
	}

	c.JSON(responseStatus, gin.H{
		"success": successValue,
		"message": fmt.Sprintf("Processed %d promotion requests: %d successful, %d failed",
			len(req.UserIDs), successCount, failedCount),
		"data": gin.H{
			"teamId":       team.ID,
			"teamName":     team.TeamName,
			"successCount": successCount,
			"totalCount":   len(req.UserIDs),
			"results":      results,
		},
	})
}

// RemoveManagerFromTeam demotes a team manager/leader to a regular member
func (h *TeamHandler) RemoveManagerFromTeam(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to remove team manager: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", "Missing user ID"))
		return
	}

	// Parse team ID
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid team ID format: %s", teamIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid team ID format", ""))
		return
	}

	// Parse manager ID
	managerIDStr := c.Param("managerId")
	managerID, err := uuid.Parse(managerIDStr)
	if err != nil {
		log.Printf("Invalid manager ID format: %s", managerIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid manager ID format", ""))
		return
	}

	// Check if team exists
	var team models.Team
	if err := h.db.First(&team, teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Team not found: %d", teamID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Team not found", ""))
			return
		}
		log.Printf("Database error when finding team: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team", err.Error()))
		return
	}

	// Check if current user is team leader
	var leaderRoster models.Roster
	if err := h.db.Where("\"teamId\" = ? AND \"userId\" = ? AND \"isLeader\" = ?", teamID, currentUserID, true).First(&leaderRoster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("User %s attempted to remove manager from team %d without leadership permission", currentUserID, teamID)
			c.JSON(http.StatusForbidden, responses.NewErrorResponse("Only team leaders can remove managers", ""))
			return
		}
		log.Printf("Database error when checking leadership: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team leadership", err.Error()))
		return
	}

	// Check if manager exists and is a leader
	var managerRoster models.Roster
	if err := h.db.Where("\"teamId\" = ? AND \"userId\" = ?", teamID, managerID).First(&managerRoster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Manager %s not found in team %d", managerID, teamID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Manager not found in this team", ""))
			return
		}
		log.Printf("Database error when checking manager: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team membership", err.Error()))
		return
	}

	if !managerRoster.IsLeader {
		log.Printf("User %s is not a manager of team %d", managerID, teamID)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("User is not a manager of this team", ""))
		return
	}

	// Count team leaders
	var leaderCount int64
	if err := h.db.Model(&models.Roster{}).Where("\"teamId\" = ? AND \"isLeader\" = ?", teamID, true).Count(&leaderCount).Error; err != nil {
		log.Printf("Error counting team leaders: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team leadership status", err.Error()))
		return
	}

	if leaderCount <= 1 {
		log.Printf("Attempted to remove the only leader from team %d", teamID)
		c.JSON(http.StatusForbidden, responses.NewErrorResponse("Cannot remove the only team leader. Assign another leader first.", ""))
		return
	}

	// Update to demote the manager
	managerRoster.IsLeader = false
	if err := h.db.Save(&managerRoster).Error; err != nil {
		log.Printf("Failed to demote manager %s from team %d: %v", managerID, teamID, err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to demote manager", err.Error()))
		return
	}

	// Fetch username for response
	userResp, _ := h.userService.GetUserByID(managerID.String())
	username := ""
	if userResp != nil && userResp.User.Username != "" {
		username = userResp.User.Username
	}

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Manager demoted successfully", gin.H{
		"teamId":   team.ID,
		"teamName": team.TeamName,
		"userId":   managerID,
		"username": username,
	}))
}

// GetTeamMembers returns all members of a team
func (h *TeamHandler) GetTeamMembers(c *gin.Context) {

	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid team ID format: %s", teamIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid team ID format", ""))
		return
	}

	// Check if team exists
	var team models.Team
	if err := h.db.First(&team, teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Team not found: %d", teamID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Team not found", ""))
			return
		}
		log.Printf("Database error when finding team: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team", err.Error()))
		return
	}

	// Try to get team members from Redis cache first
	var members []models.Roster
	var userInfos []map[string]interface{}
	var cacheHit bool

	if h.redisClient != nil {
		ctx := c.Request.Context()

		// Get team members from cache
		userIDs, err := h.redisClient.GetMembers(ctx, teamID)
		if err == nil && len(userIDs) > 0 {
			log.Printf("Cache hit for team %d members", teamID)

			if err := h.db.Where("\"teamId\" = ? AND \"userId\" IN ?", teamID, userIDs).Find(&members).Error; err != nil {
				log.Printf("Error fetching roster details: %v", err)
				// Will continue to full DB fetch
			} else {
				cacheHit = true
			}
		}
	}

	// Fallback to database if cache miss
	if !cacheHit {
		if err := h.db.Where("\"teamId\" = ?", teamID).Find(&members).Error; err != nil {
			log.Printf("Failed to fetch team members: %v", err)
			c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to fetch team members", err.Error()))
			return
		}

		// Update cache with DB results
		if h.redisClient != nil && len(members) > 0 {
			ctx := c.Request.Context()

			// Convert roster entries to user IDs for cache
			userIDs := make([]interface{}, len(members))
			for i, member := range members {
				userIDs[i] = member.UserID.String()
			}

			// Store in Redis
			if err := h.redisClient.StoreMembers(ctx, teamID, userIDs); err != nil {
				log.Printf("Failed to update cache: %v", err)
				// Non-critical error, continue
			}
		}
	}

	// Prepare response data
	userInfos = make([]map[string]interface{}, 0, len(members))
	for _, member := range members {
		// Get user details from user service
		userResp, err := h.userService.GetUserByID(member.UserID.String())
		username := ""
		if err == nil && userResp != nil && userResp.User != nil {
			username = userResp.User.Username
		}

		userInfo := map[string]interface{}{
			"userId":   member.UserID,
			"username": username,
			"isLeader": member.IsLeader,
		}
		userInfos = append(userInfos, userInfo)
	}

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Team members retrieved successfully", gin.H{
		"teamId":   team.ID,
		"teamName": team.TeamName,
		"members":  userInfos,
	}))
}

// GetTeamAssets retrieves all assets (folders and notes) that team members own or can access
func (h *TeamHandler) GetTeamAssets(c *gin.Context) {
	// Check if user has MANAGER role
	role, exists := c.Get("role")
	if !exists || role != "MANAGER" {
		log.Printf("Unauthorized access attempt: role required is MANAGER, got %v", role)
		c.JSON(http.StatusForbidden, responses.NewErrorResponse("Only managers can access team assets", ""))
		return
	}
	// Parse team ID
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid team ID format: %s", teamIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid team ID format", ""))
		return
	}

	// Check if team exists
	var team models.Team
	if err := h.db.First(&team, teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Team not found: %d", teamID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Team not found", ""))
			return
		}
		log.Printf("Database error when finding team: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify team", err.Error()))
		return
	}

	// Get all team members
	var rosters []models.Roster
	if err := h.db.Where("\"teamId\" = ?", teamID).Find(&rosters).Error; err != nil {
		log.Printf("Failed to fetch team members: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to fetch team members", err.Error()))
		return
	}

	if len(rosters) == 0 {
		c.JSON(http.StatusOK, responses.NewSuccessResponse("Team has no members", gin.H{
			"teamId":   team.ID,
			"teamName": team.TeamName,
			"folders":  []interface{}{},
			"notes":    []interface{}{},
		}))
		return
	}

	// Extract user IDs for query
	userIDs := make([]uuid.UUID, len(rosters))
	for i, roster := range rosters {
		userIDs[i] = roster.UserID
	}

	// Get folders owned by team members
	var folders []models.Folder
	if err := h.db.Where("owner_id IN ?", userIDs).Find(&folders).Error; err != nil {
		log.Printf("Failed to fetch folders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch folders",
		})
		return
	}

	// Get folders shared with team members
	var folderShares []models.FolderShare
	if err := h.db.Preload("Folder").Where("user_id IN ?", userIDs).Find(&folderShares).Error; err != nil {
		log.Printf("Failed to fetch folder shares: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch folder shares",
		})
		return
	}

	// Add shared folders to the folders list, avoiding duplicates
	folderMap := make(map[uuid.UUID]models.Folder)
	for _, folder := range folders {
		folderMap[folder.ID] = folder
	}

	for _, share := range folderShares {
		if _, exists := folderMap[share.Folder.ID]; !exists {
			folderMap[share.Folder.ID] = share.Folder
		}
	}

	// Get notes owned by team members
	var notes []models.Note
	if err := h.db.Where("owner_id IN ?", userIDs).Find(&notes).Error; err != nil {
		log.Printf("Failed to fetch notes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch notes",
		})
		return
	}

	// Get notes shared with team members
	var noteShares []models.NoteShare
	if err := h.db.Preload("Note").Where("user_id IN ?", userIDs).Find(&noteShares).Error; err != nil {
		log.Printf("Failed to fetch note shares: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch note shares",
		})
		return
	}

	// Add shared notes to the notes list, avoiding duplicates
	noteMap := make(map[uuid.UUID]models.Note)
	for _, note := range notes {
		noteMap[note.ID] = note
	}

	for _, share := range noteShares {
		if _, exists := noteMap[share.Note.ID]; !exists {
			noteMap[share.Note.ID] = share.Note
		}
	}

	// Convert maps back to slices for response
	resultFolders := make([]models.Folder, 0, len(folderMap))
	for _, folder := range folderMap {
		resultFolders = append(resultFolders, folder)
	}

	resultNotes := make([]models.Note, 0, len(noteMap))
	for _, note := range noteMap {
		resultNotes = append(resultNotes, note)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Team assets retrieved successfully",
		"data": gin.H{
			"teamId":   team.ID,
			"teamName": team.TeamName,
			"folders":  resultFolders,
			"notes":    resultNotes,
		},
	})
}

// GetUserAssets retrieves all assets (folders and notes) owned by or shared with a user
func (h *TeamHandler) GetUserAssets(c *gin.Context) {

	// Check if user has MANAGER role
	role, exists := c.Get("role")
	if !exists || role != "MANAGER" {
		log.Printf("Unauthorized access attempt: role required is MANAGER, got %v", role)
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only managers can access user assets",
		})
		return
	}

	// Parse user ID
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("Invalid user ID format: %s", userIDStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid user ID format",
		})
		return
	}

	// Verify user exists through the user service
	userResp, err := h.userService.GetUserByID(userID.String())
	if err != nil || userResp == nil || userResp.User == nil {
		log.Printf("User not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	// Get folders owned by the user
	var folders []models.Folder
	if err := h.db.Where("owner_id = ?", userID).Find(&folders).Error; err != nil {
		log.Printf("Failed to fetch folders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch folders",
		})
		return
	}

	// Get folders shared with the user
	var folderShares []models.FolderShare
	if err := h.db.Preload("Folder").Where("user_id = ?", userID).Find(&folderShares).Error; err != nil {
		log.Printf("Failed to fetch folder shares: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch folder shares",
		})
		return
	}

	// Get notes owned by the user
	var notes []models.Note
	if err := h.db.Where("owner_id = ?", userID).Find(&notes).Error; err != nil {
		log.Printf("Failed to fetch notes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch notes",
		})
		return
	}

	// Get notes shared with the user
	var noteShares []models.NoteShare
	if err := h.db.Preload("Note").Where("user_id = ?", userID).Find(&noteShares).Error; err != nil {
		log.Printf("Failed to fetch note shares: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch note shares",
		})
		return
	}

	// Add shared folders to response
	sharedFolders := make([]models.Folder, len(folderShares))
	for i, share := range folderShares {
		sharedFolders[i] = share.Folder
	}

	// Add shared notes to response
	sharedNotes := make([]models.Note, len(noteShares))
	for i, share := range noteShares {
		sharedNotes[i] = share.Note
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User assets retrieved successfully",
		"data": gin.H{
			"userId":        userID,
			"username":      userResp.User.Username,
			"ownedFolders":  folders,
			"sharedFolders": sharedFolders,
			"ownedNotes":    notes,
			"sharedNotes":   sharedNotes,
		},
	})
}
