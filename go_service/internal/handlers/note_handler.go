package handlers

import (
	"log"
	"net/http"

	"go_service/internal/models"
	"go_service/internal/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NoteHandler struct {
	db *gorm.DB
}

func NewNoteHandler(db *gorm.DB) *NoteHandler {
	return &NoteHandler{db: db}
}

// CreateNote creates a new note inside a folder
func (h *NoteHandler) CreateNote(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to create note: missing user_id")
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
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid request format", err.Error()))
		return
	}

	// Check if folder exists and user has permissions
	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Folder not found: %s", folderID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Folder not found", ""))
			return
		}
		log.Printf("Database error when finding folder: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to verify folder", ""))
		return
	}

	// Check if user is owner or has write access
	hasWriteAccess := false
	if folder.OwnerID == currentUserID.(uuid.UUID) {
		hasWriteAccess = true
	} else {
		// Check for write permission
		var folderShare models.FolderShare
		if err := h.db.Where("folder_id = ? AND user_id = ? AND access_level = ?",
			folderID, currentUserID, models.Write).First(&folderShare).Error; err == nil {
			hasWriteAccess = true
		}
	}

	if !hasWriteAccess {
		log.Printf("User %s attempted to create note in folder %s without write permission", currentUserID, folderID)
		c.JSON(http.StatusForbidden, responses.NewErrorResponse("You don't have permission to create notes in this folder", ""))
		return
	}

	// Create note
	note := models.Note{
		ID:       uuid.New(),
		Title:    req.Title,
		Content:  req.Content,
		OwnerID:  currentUserID.(uuid.UUID),
		FolderID: folderID,
	}

	if err := h.db.Create(&note).Error; err != nil {
		log.Printf("Failed to create note: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to create note", ""))
		return
	}

	c.JSON(http.StatusCreated, responses.NewSuccessResponse("Note created successfully", note))
}

// GetNote retrieves a note
func (h *NoteHandler) GetNote(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to access note: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse note ID
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		log.Printf("Invalid note ID format: %s", noteIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid note ID format", ""))
		return
	}

	// Get note from database
	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Note not found: %s", noteID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Note not found", ""))
			return
		}
		log.Printf("Database error when finding note: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to retrieve note", ""))
		return
	}

	// Check permissions - owner has direct access
	if note.OwnerID == currentUserID.(uuid.UUID) {
		c.JSON(http.StatusOK, responses.NewSuccessResponse("Note retrieved successfully", note))
		return
	}

	// Check if there's direct note sharing
	var noteShare models.NoteShare
	if err := h.db.Where("note_id = ? AND user_id = ?", noteID, currentUserID).First(&noteShare).Error; err == nil {
		c.JSON(http.StatusOK, responses.NewSuccessResponse("Note retrieved successfully", gin.H{
			"note":        note,
			"accessLevel": noteShare.AccessLevel,
			"sharedBy":    noteShare.SharedByID,
		}))
		return
	}

	// Check folder sharing
	var folderShare models.FolderShare
	if err := h.db.Where("folder_id = ? AND user_id = ?", note.FolderID, currentUserID).First(&folderShare).Error; err == nil {
		c.JSON(http.StatusOK, responses.NewSuccessResponse("Note retrieved successfully", gin.H{
			"note":          note,
			"accessLevel":   folderShare.AccessLevel,
			"sharedBy":      folderShare.SharedByID,
			"folderSharing": true,
		}))
		return
	}

	// No access
	log.Printf("User %s attempted to access note %s without permission", currentUserID, noteID)
	c.JSON(http.StatusForbidden, responses.NewErrorResponse("You don't have permission to access this note", ""))
}

// UpdateNote updates a note
func (h *NoteHandler) UpdateNote(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to update note: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse note ID
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		log.Printf("Invalid note ID format: %s", noteIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid note ID format", ""))
		return
	}

	// Parse request body
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid request format", err.Error()))
		return
	}

	// Get note from database
	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Note not found: %s", noteID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Note not found", ""))
			return
		}
		log.Printf("Database error when finding note: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to retrieve note", ""))
		return
	}

	// Check write permissions
	hasWriteAccess := false

	// Owner has write access
	if note.OwnerID == currentUserID.(uuid.UUID) {
		hasWriteAccess = true
	} else {
		// Check direct note sharing
		var noteShare models.NoteShare
		if err := h.db.Where("note_id = ? AND user_id = ? AND access_level = ?",
			noteID, currentUserID, models.Write).First(&noteShare).Error; err == nil {
			hasWriteAccess = true
		} else {
			// Check folder sharing
			var folderShare models.FolderShare
			if err := h.db.Where("folder_id = ? AND user_id = ? AND access_level = ?",
				note.FolderID, currentUserID, models.Write).First(&folderShare).Error; err == nil {
				hasWriteAccess = true
			}
		}
	}

	if !hasWriteAccess {
		log.Printf("User %s attempted to update note %s without write permission", currentUserID, noteID)
		c.JSON(http.StatusForbidden, responses.NewErrorResponse("You don't have permission to update this note", ""))
		return
	}

	// Update note
	if req.Title != "" {
		note.Title = req.Title
	}
	if req.Content != "" {
		note.Content = req.Content
	}

	if err := h.db.Save(&note).Error; err != nil {
		log.Printf("Failed to update note: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to update note", ""))
		return
	}

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Note updated successfully", note))
}

// DeleteNote deletes a note
func (h *NoteHandler) DeleteNote(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to delete note: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse note ID
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		log.Printf("Invalid note ID format: %s", noteIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid note ID format", ""))
		return
	}

	// Get note from database
	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Note not found: %s", noteID)
			c.JSON(http.StatusNotFound, responses.NewErrorResponse("Note not found", ""))
			return
		}
		log.Printf("Database error when finding note: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to retrieve note", ""))
		return
	}

	// Check if user is owner
	if note.OwnerID != currentUserID.(uuid.UUID) {
		// Check if user has write access to the folder
		var folderShare models.FolderShare
		if err := h.db.Where("folder_id = ? AND user_id = ? AND access_level = ?",
			note.FolderID, currentUserID, models.Write).First(&folderShare).Error; err != nil {
			log.Printf("User %s attempted to delete note %s without ownership or write permission", currentUserID, noteID)
			c.JSON(http.StatusForbidden, responses.NewErrorResponse("You don't have permission to delete this note", ""))
			return
		}
	}

	// Begin transaction
	tx := h.db.Begin()

	// Delete note shares first
	if err := tx.Where("note_id = ?", noteID).Delete(&models.NoteShare{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to delete note shares: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to delete note shares", ""))
		return
	}

	// Delete the note
	if err := tx.Delete(&note).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to delete note: %v", err)
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse("Failed to delete note", ""))
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, responses.NewSuccessResponse("Note deleted successfully", nil))
}

// ShareNote shares a note with another user
func (h *NoteHandler) ShareNote(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to share note: missing user_id")
		c.JSON(http.StatusUnauthorized, responses.NewErrorResponse("Authentication required", ""))
		return
	}

	// Parse note ID
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		log.Printf("Invalid note ID format: %s", noteIDStr)
		c.JSON(http.StatusBadRequest, responses.NewErrorResponse("Invalid note ID format", ""))
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

	// Get note from database
	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Note not found: %s", noteID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Note not found",
			})
			return
		}
		log.Printf("Database error when finding note: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve note",
		})
		return
	}

	// Check if user is the owner
	if note.OwnerID != currentUserID.(uuid.UUID) {
		log.Printf("User %s attempted to share note %s without ownership", currentUserID, noteID)
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only the owner can share this note",
		})
		return
	}

	// Check if already shared with this user
	var existingShare models.NoteShare
	if err := h.db.Where("note_id = ? AND user_id = ?", noteID, req.UserID).First(&existingShare).Error; err == nil {
		// Update existing share
		existingShare.AccessLevel = req.AccessLevel
		if err := h.db.Save(&existingShare).Error; err != nil {
			log.Printf("Failed to update share: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to update share",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Note sharing updated successfully",
			"data":    existingShare,
		})
		return
	} else if err != gorm.ErrRecordNotFound {
		log.Printf("Database error when checking existing share: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify existing shares",
		})
		return
	}

	// Create new share
	share := models.NoteShare{
		ID:          uuid.New(),
		NoteID:      noteID,
		UserID:      req.UserID,
		AccessLevel: req.AccessLevel,
		SharedByID:  currentUserID.(uuid.UUID),
	}

	if err := h.db.Create(&share).Error; err != nil {
		log.Printf("Failed to create share: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to share note",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Note shared successfully",
		"data":    share,
	})
}

// RevokeNoteSharing revokes note sharing for a specific user
func (h *NoteHandler) RevokeNoteSharing(c *gin.Context) {
	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		log.Println("Unauthorized attempt to revoke note sharing: missing user_id")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Parse note ID
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		log.Printf("Invalid note ID format: %s", noteIDStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid note ID format",
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

	// Check if note exists and user is owner
	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Note not found: %s", noteID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Note not found",
			})
			return
		}
		log.Printf("Database error when finding note: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve note",
		})
		return
	}

	// Only the owner can revoke sharing
	if note.OwnerID != currentUserID.(uuid.UUID) {
		log.Printf("User %s attempted to revoke sharing for note %s without ownership", currentUserID, noteID)
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only the owner can revoke sharing",
		})
		return
	}

	// Find share
	var share models.NoteShare
	if err := h.db.Where("note_id = ? AND user_id = ?", noteID, userID).First(&share).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Share not found for note %s and user %s", noteID, userID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Sharing not found",
			})
			return
		}
		log.Printf("Database error when finding share: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify sharing",
		})
		return
	}

	// Delete share
	if err := h.db.Delete(&share).Error; err != nil {
		log.Printf("Failed to delete share: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to revoke sharing",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Note sharing revoked successfully",
	})
}
