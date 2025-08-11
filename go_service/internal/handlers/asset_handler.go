package handlers

import (
	"net/http"
	"strconv"

	"go_service/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssetHandler struct {
	db *gorm.DB
}

func NewAssetHandler(db *gorm.DB) *AssetHandler {
	return &AssetHandler{db: db}
}

// =============================================================================
// FOLDER MANAGEMENT
// =============================================================================

// CreateFolder creates a new folder
func (h *AssetHandler) CreateFolder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		// use custom error to response
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	ownerID := userID.(uuid.UUID)

	var req struct {
		FolderName string `json:"folderName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	folder := models.Folder{
		FolderName: req.FolderName,
		OwnerID:    ownerID,
	}

	// pass ctx to database.DB.Create
	if err := h.db.Create(&folder).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create folder"})
		return
	}

	c.JSON(http.StatusCreated, folder)
}

// GetFolder gets folder details
func (h *AssetHandler) GetFolder(c *gin.Context) {
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	// Check if user has access to this folder
	hasAccess := h.checkFolderAccess(currentUserID, folderID)
	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this folder"})
		return
	}

	c.JSON(http.StatusOK, folder)
}

// UpdateFolder updates folder details (only owner can update)
func (h *AssetHandler) UpdateFolder(c *gin.Context) {
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	// Only owner can update folder
	if folder.OwnerID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only folder owner can update folder"})
		return
	}

	var req struct {
		FolderName string `json:"folderName"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FolderName != "" {
		folder.FolderName = req.FolderName
	}

	if err := h.db.Save(&folder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder"})
		return
	}

	c.JSON(http.StatusOK, folder)
}

// DeleteFolder deletes a folder and all its notes (only owner can delete)
func (h *AssetHandler) DeleteFolder(c *gin.Context) {
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	// Only owner can delete folder
	if folder.OwnerID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only folder owner can delete folder"})
		return
	}

	tx := h.db.Begin()

	// Delete all notes in the folder
	if err := tx.Where("folder_id = ?", folderID).Delete(&models.Note{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notes"})
		return
	}

	// Delete all folder shares
	if err := tx.Where("folder_id = ?", folderID).Delete(&models.FolderShare{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete folder shares"})
		return
	}

	// Delete the folder
	if err := tx.Delete(&folder).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete folder"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Folder and all its contents deleted successfully"})
}

// =============================================================================
// NOTE MANAGEMENT
// =============================================================================

// CreateNote creates a new note inside a folder
func (h *AssetHandler) CreateNote(c *gin.Context) {
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	// Check if user has write access to the folder
	hasWriteAccess := h.checkFolderWriteAccess(currentUserID, folderID)
	if !hasWriteAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have write access to this folder"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	note := models.Note{
		Title:    req.Title,
		Content:  req.Content,
		OwnerID:  currentUserID,
		FolderID: folderID,
	}

	if err := h.db.Create(&note).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create note"})
		return
	}

	c.JSON(http.StatusCreated, note)
}

// GetNote gets note details
func (h *AssetHandler) GetNote(c *gin.Context) {
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	// Check if user has access to this note
	hasAccess := h.checkNoteAccess(currentUserID, noteID)
	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this note"})
		return
	}

	c.JSON(http.StatusOK, note)
}

// UpdateNote updates note content (only owner or those with write access)
func (h *AssetHandler) UpdateNote(c *gin.Context) {
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	// Check if user has write access to this note
	hasWriteAccess := h.checkNoteWriteAccess(currentUserID, noteID)
	if !hasWriteAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have write access to this note"})
		return
	}

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title != "" {
		note.Title = req.Title
	}
	if req.Content != "" {
		note.Content = req.Content
	}

	if err := h.db.Save(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update note"})
		return
	}

	c.JSON(http.StatusOK, note)
}

// DeleteNote deletes a note (only owner can delete)
func (h *AssetHandler) DeleteNote(c *gin.Context) {
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	// Only owner can delete note
	if note.OwnerID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only note owner can delete note"})
		return
	}

	tx := h.db.Begin()

	// Delete all note shares
	if err := tx.Where("note_id = ?", noteID).Delete(&models.NoteShare{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete note shares"})
		return
	}

	// Delete the note
	if err := tx.Delete(&note).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete note"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Note deleted successfully"})
}

// =============================================================================
// SHARING FUNCTIONALITY
// =============================================================================

// ShareFolder shares a folder with another user
func (h *AssetHandler) ShareFolder(c *gin.Context) {
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	// Only owner can share folder
	if folder.OwnerID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only folder owner can share folder"})
		return
	}

	var req struct {
		UserID      uuid.UUID          `json:"userId" binding:"required"`
		AccessLevel models.AccessLevel `json:"accessLevel" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if target user exists
	var targetUser models.User
	if err := h.db.First(&targetUser, "id = ?", req.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target user not found"})
		return
	}

	// Check if share already exists
	var existingShare models.FolderShare
	if err := h.db.Where("folder_id = ? AND user_id = ?", folderID, req.UserID).First(&existingShare).Error; err == nil {
		// Update existing share
		existingShare.AccessLevel = req.AccessLevel
		if err := h.db.Save(&existingShare).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder share"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Folder share updated successfully"})
		return
	}

	// Create new share
	folderShare := models.FolderShare{
		FolderID:    folderID,
		UserID:      req.UserID,
		AccessLevel: req.AccessLevel,
		SharedByID:  currentUserID,
	}

	if err := h.db.Create(&folderShare).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to share folder"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Folder shared successfully"})
}

// RevokeFolderShare revokes folder sharing
func (h *AssetHandler) RevokeFolderShare(c *gin.Context) {
	folderIDStr := c.Param("folderId")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	targetUserIDStr := c.Param("userId")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var folder models.Folder
	if err := h.db.First(&folder, "id = ?", folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	// Only owner can revoke folder sharing
	if folder.OwnerID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only folder owner can revoke folder sharing"})
		return
	}

	if err := h.db.Where("folder_id = ? AND user_id = ?", folderID, targetUserID).Delete(&models.FolderShare{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke folder share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder share revoked successfully"})
}

// ShareNote shares a note with another user
func (h *AssetHandler) ShareNote(c *gin.Context) {
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	// Only owner can share note
	if note.OwnerID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only note owner can share note"})
		return
	}

	var req struct {
		UserID      uuid.UUID          `json:"userId" binding:"required"`
		AccessLevel models.AccessLevel `json:"accessLevel" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if target user exists
	var targetUser models.User
	if err := h.db.First(&targetUser, "id = ?", req.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target user not found"})
		return
	}

	// Check if share already exists
	var existingShare models.NoteShare
	if err := h.db.Where("note_id = ? AND user_id = ?", noteID, req.UserID).First(&existingShare).Error; err == nil {
		// Update existing share
		existingShare.AccessLevel = req.AccessLevel
		if err := h.db.Save(&existingShare).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update note share"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Note share updated successfully"})
		return
	}

	// Create new share
	noteShare := models.NoteShare{
		NoteID:      noteID,
		UserID:      req.UserID,
		AccessLevel: req.AccessLevel,
		SharedByID:  currentUserID,
	}

	if err := h.db.Create(&noteShare).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to share note"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Note shared successfully"})
}

// RevokeNoteShare revokes note sharing
func (h *AssetHandler) RevokeNoteShare(c *gin.Context) {
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	targetUserIDStr := c.Param("userId")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	currentUserID := userID.(uuid.UUID)

	var note models.Note
	if err := h.db.First(&note, "id = ?", noteID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	// Only owner can revoke note sharing
	if note.OwnerID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only note owner can revoke note sharing"})
		return
	}

	if err := h.db.Where("note_id = ? AND user_id = ?", noteID, targetUserID).Delete(&models.NoteShare{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke note share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Note share revoked successfully"})
}

// =============================================================================
// MANAGER-ONLY ENDPOINTS
// =============================================================================

// GetTeamAssets returns all assets that team members own or can access (managers only)
func (h *AssetHandler) GetTeamAssets(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || role != string(models.Manager) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only managers can access team assets"})
		return
	}

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

	// Check if current user is a manager of this team using Roster table
	var currentUserRoster models.Roster
	if err := h.db.Where("teamId = ? AND userId = ? AND isLeader = ?", teamID, currentUserID, true).First(&currentUserRoster).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a manager of this team"})
		return
	}

	// Get all team members through Roster table
	var rosters []models.Roster
	if err := h.db.Where("teamId = ?", teamID).Find(&rosters).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch team members"})
		return
	}

	// Extract member IDs
	memberIDs := make([]uuid.UUID, len(rosters))
	for i, roster := range rosters {
		memberIDs[i] = roster.UserID
	}

	// Get folders owned by team members
	var folders []models.Folder
	if err := h.db.Where("owner_id IN ?", memberIDs).Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch folders"})
		return
	}

	// Get notes owned by team members
	var notes []models.Note
	if err := h.db.Where("owner_id IN ?", memberIDs).Find(&notes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notes"})
		return
	}

	// Get shared folders accessible by team members
	var sharedFolders []models.FolderShare
	if err := h.db.Preload("Folder").Where("user_id IN ?", memberIDs).Find(&sharedFolders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shared folders"})
		return
	}

	// Get shared notes accessible by team members
	var sharedNotes []models.NoteShare
	if err := h.db.Preload("Note").Where("user_id IN ?", memberIDs).Find(&sharedNotes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shared notes"})
		return
	}

	response := gin.H{
		"ownedFolders":  folders,
		"ownedNotes":    notes,
		"sharedFolders": sharedFolders,
		"sharedNotes":   sharedNotes,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserAssets returns all assets owned by or shared with a specific user (managers only)
func (h *AssetHandler) GetUserAssets(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || role != string(models.Manager) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only managers can access user assets"})
		return
	}

	targetUserIDStr := c.Param("userId")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if target user exists
	var targetUser models.User
	if err := h.db.First(&targetUser, "userId = ?", targetUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get folders owned by user
	var folders []models.Folder
	if err := h.db.Where("owner_id = ?", targetUserID).Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch folders"})
		return
	}

	// Get notes owned by user
	var notes []models.Note
	if err := h.db.Where("owner_id = ?", targetUserID).Find(&notes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notes"})
		return
	}

	// Get shared folders accessible by user
	var sharedFolders []models.FolderShare
	if err := h.db.Preload("Folder").Where("user_id = ?", targetUserID).Find(&sharedFolders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shared folders"})
		return
	}

	// Get shared notes accessible by user
	var sharedNotes []models.NoteShare
	if err := h.db.Preload("Note").Where("user_id = ?", targetUserID).Find(&sharedNotes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shared notes"})
		return
	}

	response := gin.H{
		"user":          targetUser,
		"ownedFolders":  folders,
		"ownedNotes":    notes,
		"sharedFolders": sharedFolders,
		"sharedNotes":   sharedNotes,
	}

	c.JSON(http.StatusOK, response)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// checkFolderAccess checks if user has any access to a folder
func (h *AssetHandler) checkFolderAccess(userID, folderID uuid.UUID) bool {
	// Check if user is the owner
	var folder models.Folder
	if err := h.db.First(&folder, "id = ? AND owner_id = ?", folderID, userID).Error; err == nil {
		return true
	}

	// Check if folder is shared with user
	var share models.FolderShare
	if err := h.db.First(&share, "folder_id = ? AND user_id = ?", folderID, userID).Error; err == nil {
		return true
	}

	return false
}

// checkFolderWriteAccess checks if user has write access to a folder
func (h *AssetHandler) checkFolderWriteAccess(userID, folderID uuid.UUID) bool {
	// Check if user is the owner
	var folder models.Folder
	if err := h.db.First(&folder, "id = ? AND owner_id = ?", folderID, userID).Error; err == nil {
		return true
	}

	// Check if folder is shared with write access
	var share models.FolderShare
	if err := h.db.First(&share, "folder_id = ? AND user_id = ? AND access_level = ?", folderID, userID, models.Write).Error; err == nil {
		return true
	}

	return false
}

// checkNoteAccess checks if user has any access to a note
func (h *AssetHandler) checkNoteAccess(userID, noteID uuid.UUID) bool {
	// Check if user is the owner
	var note models.Note
	if err := h.db.First(&note, "id = ? AND owner_id = ?", noteID, userID).Error; err == nil {
		return true
	}

	// Check if note is shared with user
	var share models.NoteShare
	if err := h.db.First(&share, "note_id = ? AND user_id = ?", noteID, userID).Error; err == nil {
		return true
	}

	// Check if note's folder is shared with user
	if err := h.db.First(&note, "id = ?", noteID).Error; err == nil {
		return h.checkFolderAccess(userID, note.FolderID)
	}

	return false
}

// checkNoteWriteAccess checks if user has write access to a note
func (h *AssetHandler) checkNoteWriteAccess(userID, noteID uuid.UUID) bool {
	// Check if user is the owner
	var note models.Note
	if err := h.db.First(&note, "id = ? AND owner_id = ?", noteID, userID).Error; err == nil {
		return true
	}

	// Check if note is shared with write access
	var share models.NoteShare
	if err := h.db.First(&share, "note_id = ? AND user_id = ? AND access_level = ?", noteID, userID, models.Write).Error; err == nil {
		return true
	}

	// Check if note's folder is shared with write access
	if err := h.db.First(&note, "id = ?", noteID).Error; err == nil {
		return h.checkFolderWriteAccess(userID, note.FolderID)
	}

	return false
}
