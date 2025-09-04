package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"go_service/internal/dto"
	"go_service/internal/kafka"
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
	service     *services.TeamService
	producer    *kafka.Producer
	redisClient *redisclient.TeamCache
}

func NewTeamHandler(db *gorm.DB, producer *kafka.Producer, redisClient *redisclient.TeamCache) *TeamHandler {
	return &TeamHandler{
		db:          db,
		userService: services.NewUserService(),
		service:     services.NewTeamService(db, producer, redisClient),
		producer:    producer,
		redisClient: redisClient,
	}
}

// CreateTeam creates a new team with members (only managers can create teams)
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || strings.ToUpper(role.(string)) != "MANAGER" {
		responses.Error(c, http.StatusForbidden, fmt.Errorf("insufficient permissions"), "Only managers can create teams")
		return
	}

	var req dto.CreateTeamReq
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Error(c, http.StatusBadRequest, err, "Invalid request format")
		return
	}

	userID, _ := c.Get("user_id")
	creatorID := userID.(uuid.UUID)

	team, failedMembers, err := h.service.CreateTeam(c.Request.Context(), req.TeamName, req.UserIDs, creatorID)
	if err != nil {
		responses.Error(c, http.StatusInternalServerError, err, "Failed to create team")
		return
	}

	response := gin.H{
		"success": true,
		"message": "Team created successfully",
		"data": gin.H{
			"team":          team,
			"failedMembers": failedMembers,
		},
	}
	responses.JSON(c, http.StatusCreated, response)
}

// POST /teams/:teamId/members
func (h *TeamHandler) AddMemberToTeam(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		responses.Error(c, http.StatusUnauthorized, fmt.Errorf("missing user_id"), "Authentication required")
		return
	}

	teamIDStr := c.Param("teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		responses.Error(c, http.StatusBadRequest, err, "Invalid teamId")
		return
	}

	var req dto.AddMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Error(c, http.StatusBadRequest, err, "Invalid request format")
		return
	}

	addedCount, failedCount, failedMembers, err := h.service.AddMemberToTeam(c.Request.Context(), teamID, req.UserIDs, currentUserID.(uuid.UUID))
	if err != nil {
		responses.Error(c, http.StatusBadRequest, err, "Failed to add members to team")
		return
	}

	status := http.StatusCreated
	if addedCount > 0 && failedCount > 0 {
		status = http.StatusPartialContent
	} else if addedCount == 0 {
		status = http.StatusBadRequest
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

	responses.JSON(c, status, gin.H{
		"success": addedCount > 0,
		"message": fmt.Sprintf("Added: %d, Failed: %d", addedCount, failedCount),
		"data": gin.H{
			"addedCount":   addedCount,
			"failedCount":  failedCount,
			"failedMembers": failedMembers,
		},
	})
}
			"totalCount": len(req.UserIDs),
			"results":    results,
		},
	})
}
