package handlers

import (
	"context"
	"log"
	"net/http"

	"go_service/internal/events"
	"go_service/internal/kafka"
	"go_service/internal/models"
	"go_service/internal/redis"
	"go_service/internal/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderHandler struct {
	db             *gorm.DB
	kafkaProducer  *kafka.Producer
	redisService   *redis.Service
}

func NewFolderHandler(db *gorm.DB, kafkaProducer *kafka.Producer, redisService *redis.Service) *FolderHandler {
	return &FolderHandler{
		db:             db,
		kafkaProducer:  kafkaProducer,
		redisService:   redisService,
	}
}

// CreateFolder creates a new folder for the authenticated user
func (h *FolderHandler) CreateFolder(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to create folder: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse request body
	var req struct {
		FolderName string `json:"folderName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid request format", err.Error()))
		return
	}

	// Create folder object
	folder := models.Folder{
		ID:         uuid.New(),
		FolderName: req.FolderName,
		OwnerID:    currentUserID.(uuid.UUID),
	}

	// Save to database
	if err := h.db.Create(&folder).Error; err != nil {
		log.Printf("Failed to create folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to create folder", ""))
		return
	}

	// Emit folder created event
	if h.kafkaProducer != nil {
		assetEvent := events.NewAssetEvent(events.FolderCreated, events.AssetTypeFolder, folder.ID, folder.OwnerID, folder.OwnerID)
		if err := h.kafkaProducer.PublishAssetEvent(context.Background(), assetEvent); err != nil {
			log.Printf("Failed to publish folder created event: %v", err)
		}
	}

	// Cache folder metadata
	if h.redisService != nil {
		if err := h.redisService.SetFolderMetadata(context.Background(), &folder); err != nil {
			log.Printf("Failed to cache folder metadata: %v", err)
		}
	}

	c.JSON(http.StatusCreated, responses.NewSuccessResponse("Folder created successfully", folder))
}

// GetFolderDetails retrieves details of a specific folder
func (h *FolderHandler) GetFolderDetails(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to access folder: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse folder ID
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		log.Printf("Invalid folder ID format: %s", folderIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid folder ID format", ""))
		return
	}

	// Try to get folder details from cache first
	var folder models.Folder
	var folderFromCache *models.Folder
	
	if h.redisService != nil {
		folderFromCache, err = h.redisService.GetFolderMetadata(context.Background(), folderID)
		if err != nil {
			log.Printf("Cache error when getting folder metadata: %v", err)
		}
	}
	
	if folderFromCache != nil {
		// Use cached data
		folder = *folderFromCache
		log.Printf("Retrieved folder %s from cache", folderID)
	} else {
		// Get folder details from database
		if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Printf("Folder not found: %s", folderID)
				c.JSON(http.StatusNotFound, responses.NewErrorResponse("Folder not found", ""))
				return
			}
			log.Printf("Database error when finding folder: %v", err)
			c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to retrieve folder details", ""))
			return
		}
		
		// Cache the folder for future requests
		if h.redisService != nil {
			if err := h.redisService.SetFolderMetadata(context.Background(), &folder); err != nil {
				log.Printf("Failed to cache folder metadata: %v", err)
			}
		}
	}

	// Check ownership or sharing permissions
	if folder.OwnerID != currentUserID.(uuid.UUID) {
		// Try to check access from cache first
		var hasAccess bool
		var accessLevel string
		var sharedByID uuid.UUID
		
		if h.redisService != nil {
			acl, err := h.redisService.GetAssetACL(context.Background(), folderID)
			if err != nil {
				log.Printf("Cache error when getting asset ACL: %v", err)
			} else if acl != nil {
				if level, exists := acl[currentUserID.(uuid.UUID).String()]; exists {
					hasAccess = true
					accessLevel = level
					log.Printf("Retrieved folder access from cache for user %s", currentUserID)
				}
			}
		}
		
		if !hasAccess {
			// Check if folder is shared with the user from database
			var folderShare models.FolderShare
			if err := h.db.Where("folder_id = ? AND user_id = ?", folderID, currentUserID).First(&folderShare).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					log.Printf("User %s attempted to access folder %s without permission", currentUserID, folderID)
					c.JSON(http.StatusForbidden, responses.NewErrorResponse("You don't have permission to access this folder", ""))
					return
				}
				log.Printf("Database error when checking folder access: %v", err)
				c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify folder access permission", ""))
				return
			}
			accessLevel = string(folderShare.AccessLevel)
			sharedByID = folderShare.SharedByID
			
			// Update cache with the access information
			if h.redisService != nil {
				if err := h.redisService.AddAssetAccess(context.Background(), folderID, currentUserID.(uuid.UUID), accessLevel); err != nil {
					log.Printf("Failed to cache access control: %v", err)
				}
			}
		}
		
		// Include sharing info in response
		c.JSON(http.StatusOK, responses.NewSuccessResponse("Folder details retrieved successfully", gin.H{
			"folder":      folder,
			"accessLevel": accessLevel,
			"sharedBy":    sharedByID,
		}))
		return
	}

	// Get notes in this folder
	var notes []models.Note
	if err := h.db.Where("folder_id = ?", folderID).Find(&notes).Error; err != nil {
		log.Printf("Error fetching notes for folder %s: %v", folderID, err)
	}

	// Get sharing info
	var shares []models.FolderShare
	if err := h.db.Where("folder_id = ?", folderID).Find(&shares).Error; err != nil {
		log.Printf("Error fetching sharing info for folder %s: %v", folderID, err)
	}

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Folder details retrieved successfully", gin.H{
		"folder": folder,
		"notes":  notes,
		"shared": len(shares) > 0,
		"shares": shares,
	}))
}

// UpdateFolder updates folder details
func (h *FolderHandler) UpdateFolder(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to update folder: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse folder ID
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		log.Printf("Invalid folder ID format: %s", folderIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid folder ID format", ""))
		return
	}

	// Parse request body
	var req struct {
		FolderName string `json:"folderName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid request format", err.Error()))
		return
	}

	// Get folder from database
	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Folder not found: %s", folderID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Folder not found", ""))
			return
		}
		log.Printf("Database error when finding folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to retrieve folder", ""))
		return
	}

	// Check ownership or write permission
	if folder.OwnerID != currentUserID.(uuid.UUID) {
		// Check if user has write permission
		var folderShare models.FolderShare
		if err := h.db.Where("folder_id = ? AND user_id = ? AND access_level = ?",
			folderID, currentUserID, models.Write).First(&folderShare).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Printf("User %s attempted to update folder %s without permission", currentUserID, folderID)
				c.JSON(http.StatusForbidden, responses.NewErrorResponse("You don't have permission to update this folder", ""))
				return
			}
			log.Printf("Database error when checking folder write permission: %v", err)
			c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify folder write permission", ""))
			return
		}
	}

	// Update folder
	folder.FolderName = req.FolderName
	if err := h.db.Save(&folder).Error; err != nil {
		log.Printf("Failed to update folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to update folder", ""))
		return
	}

	// Emit folder updated event
	if h.kafkaProducer != nil {
		assetEvent := events.NewAssetEvent(events.FolderUpdated, events.AssetTypeFolder, folder.ID, folder.OwnerID, currentUserID.(uuid.UUID))
		if err := h.kafkaProducer.PublishAssetEvent(context.Background(), assetEvent); err != nil {
			log.Printf("Failed to publish folder updated event: %v", err)
		}
	}

	// Invalidate and update cache with new data
	if h.redisService != nil {
		if err := h.redisService.SetFolderMetadata(context.Background(), &folder); err != nil {
			log.Printf("Failed to update folder cache: %v", err)
		}
	}

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Folder updated successfully", folder))
}

// DeleteFolder deletes a folder and its notes
func (h *FolderHandler) DeleteFolder(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to delete folder: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse folder ID
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		log.Printf("Invalid folder ID format: %s", folderIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid folder ID format", ""))
		return
	}

	// Get folder from database
	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Folder not found: %s", folderID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Folder not found", ""))
			return
		}
		log.Printf("Database error when finding folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to retrieve folder", ""))
		return
	}

	// Only the owner can delete a folder
	if folder.OwnerID != currentUserID.(uuid.UUID) {
		log.Printf("User %s attempted to delete folder %s without ownership", currentUserID, folderID)
		c.JSON(http.StatusForbidden, responses.NewErrorResponse("Only the owner can delete this folder", ""))
		return
	}

	// Begin transaction for cascading delete
	tx := h.db.Begin()

	// Delete shares first
	if err := tx.Where("folder_id = ?", folderID).Delete(&models.FolderShare{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to delete folder shares: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to delete folder shares", ""))
		return
	}

	// Delete note shares
	var notesInFolder []models.Note
	if err := tx.Where("folder_id = ?", folderID).Find(&notesInFolder).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to find notes in folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to process folder deletion", ""))
		return
	}

	// Delete note shares for each note
	for _, note := range notesInFolder {
		if err := tx.Where("note_id = ?", note.ID).Delete(&models.NoteShare{}).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to delete note shares: %v", err)
			c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to delete note shares", ""))
			return
		}
	}

	// Delete notes
	if err := tx.Where("folder_id = ?", folderID).Delete(&models.Note{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to delete notes in folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to delete notes in folder", ""))
		return
	}

	// Delete folder
	if err := tx.Delete(&folder).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to delete folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to delete folder", ""))
		return
	}

	tx.Commit()

	// Emit folder deleted event
	if h.kafkaProducer != nil {
		assetEvent := events.NewAssetEvent(events.FolderDeleted, events.AssetTypeFolder, folder.ID, folder.OwnerID, currentUserID.(uuid.UUID))
		if err := h.kafkaProducer.PublishAssetEvent(context.Background(), assetEvent); err != nil {
			log.Printf("Failed to publish folder deleted event: %v", err)
		}
	}

	// Invalidate cache
	if h.redisService != nil {
		if err := h.redisService.InvalidateFolderMetadata(context.Background(), folder.ID); err != nil {
			log.Printf("Failed to invalidate folder cache: %v", err)
		}
	}

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Folder and all its contents deleted successfully", nil))
}

// ShareFolder shares a folder with another user
func (h *FolderHandler) ShareFolder(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to share folder: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse folder ID
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		log.Printf("Invalid folder ID format: %s", folderIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid folder ID format", ""))
		return
	}

	// Parse request body
	var req struct {
		UserID      uuid.UUID          `json:"userId" binding:"required"`
		AccessLevel models.AccessLevel `json:"accessLevel" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid request format", err.Error()))
		return
	}

	// Validate access level
	if req.AccessLevel != models.Read && req.AccessLevel != models.Write {
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid access level. Must be 'read' or 'write'", ""))
		return
	}

	// Get folder from database
	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Folder not found: %s", folderID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Folder not found", ""))
			return
		}
		log.Printf("Database error when finding folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to retrieve folder", ""))
		return
	}

	// Check if user is the owner
	if folder.OwnerID != currentUserID.(uuid.UUID) {
		log.Printf("User %s attempted to share folder %s without ownership", currentUserID, folderID)
		c.JSON(http.StatusForbidden, responses.NewErrorResponse("Only the owner can share this folder", ""))
		return
	}

	// Check if already shared with this user
	var existingShare models.FolderShare
	if err := h.db.Where("folder_id = ? AND user_id = ?", folderID, req.UserID).First(&existingShare).Error; err == nil {
		// Update existing share
		existingShare.AccessLevel = req.AccessLevel
		if err := h.db.Save(&existingShare).Error; err != nil {
			log.Printf("Failed to update share: %v", err)
			c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to update share", ""))
			return
		}

		// Emit folder shared event for update
		if h.kafkaProducer != nil {
			assetEvent := events.NewAssetSharingEvent(events.FolderShared, events.AssetTypeFolder, folderID, folder.OwnerID, currentUserID.(uuid.UUID), req.UserID, string(req.AccessLevel))
			if err := h.kafkaProducer.PublishAssetEvent(context.Background(), assetEvent); err != nil {
				log.Printf("Failed to publish folder shared event: %v", err)
			}
		}

		// Update Redis access control cache
		if h.redisService != nil {
			if err := h.redisService.AddAssetAccess(context.Background(), folderID, req.UserID, string(req.AccessLevel)); err != nil {
				log.Printf("Failed to update access control cache: %v", err)
			}
		}

		c.JSON(http.StatusOK, responses.NewSuccessResponse("Folder sharing updated successfully", existingShare))
		return
	} else if err != gorm.ErrRecordNotFound {
		log.Printf("Database error when checking existing share: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify existing shares", ""))
		return
	}

	// Create new share
	share := models.FolderShare{
		ID:          uuid.New(),
		FolderID:    folderID,
		UserID:      req.UserID,
		AccessLevel: req.AccessLevel,
		SharedByID:  currentUserID.(uuid.UUID),
	}

	if err := h.db.Create(&share).Error; err != nil {
		log.Printf("Failed to create share: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to share folder", ""))
		return
	}

	// Emit folder shared event for new share
	if h.kafkaProducer != nil {
		assetEvent := events.NewAssetSharingEvent(events.FolderShared, events.AssetTypeFolder, folderID, folder.OwnerID, currentUserID.(uuid.UUID), req.UserID, string(req.AccessLevel))
		if err := h.kafkaProducer.PublishAssetEvent(context.Background(), assetEvent); err != nil {
			log.Printf("Failed to publish folder shared event: %v", err)
		}
	}

	// Update Redis access control cache
	if h.redisService != nil {
		if err := h.redisService.AddAssetAccess(context.Background(), folderID, req.UserID, string(req.AccessLevel)); err != nil {
			log.Printf("Failed to update access control cache: %v", err)
		}
	}

	if err := h.db.Preload("Folder").First(&share, "id = ?", share.ID).Error; err != nil {
		log.Printf("Warning: Could not load folder details: %v", err)
	}

	c.JSON(http.StatusCreated, responses.NewSuccessResponse("Folder shared successfully", share))
}

// RevokeSharing revokes folder sharing for a specific user
func (h *FolderHandler) RevokeSharing(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to revoke folder sharing: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse folder ID
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		log.Printf("Invalid folder ID format: %s", folderIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid folder ID format", ""))
		return
	}

	// Parse user ID
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("Invalid user ID format: %s", userIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid user ID format", ""))
		return
	}

	// Check if folder exists and user is owner
	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Folder not found: %s", folderID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Folder not found", ""))
			return
		}
		log.Printf("Database error when finding folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to retrieve folder", ""))
		return
	}

	// Only the owner can revoke sharing
	if folder.OwnerID != currentUserID.(uuid.UUID) {
		log.Printf("User %s attempted to revoke sharing for folder %s without ownership", currentUserID, folderID)
		c.JSON(http.StatusForbidden, responses.NewErrorResponse("Only the owner can revoke sharing", ""))
		return
	}

	// Find share
	var share models.FolderShare
	if err := h.db.Where("folder_id = ? AND user_id = ?", folderID, userID).First(&share).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Share not found for folder %s and user %s", folderID, userID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Sharing not found", ""))
			return
		}
		log.Printf("Database error when finding share: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify sharing", ""))
		return
	}

	// Delete share
	if err := h.db.Delete(&share).Error; err != nil {
		log.Printf("Failed to delete share: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to revoke sharing", ""))
		return
	}

	// Emit folder unshared event
	if h.kafkaProducer != nil {
		assetEvent := events.NewAssetSharingEvent(events.FolderUnshared, events.AssetTypeFolder, folderID, folder.OwnerID, currentUserID.(uuid.UUID), userID, "")
		if err := h.kafkaProducer.PublishAssetEvent(context.Background(), assetEvent); err != nil {
			log.Printf("Failed to publish folder unshared event: %v", err)
		}
	}

	// Update Redis access control cache
	if h.redisService != nil {
		if err := h.redisService.RemoveAssetAccess(context.Background(), folderID, userID); err != nil {
			log.Printf("Failed to update access control cache: %v", err)
		}
	}

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Folder sharing revoked successfully", nil))
}
