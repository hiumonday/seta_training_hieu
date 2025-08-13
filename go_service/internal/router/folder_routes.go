package router

import (
	"go_service/internal/handlers"

	"github.com/gin-gonic/gin"
)

// FolderRoutes defines routes for folder management
func FolderRoutes(rg *gin.RouterGroup, folderHandler *handlers.FolderHandler, noteHandler *handlers.NoteHandler) {
	folders := rg.Group("/folders")
	{
		folders.POST("", folderHandler.CreateFolder)
		folders.GET("/:folderId", folderHandler.GetFolderDetails)
		folders.PUT("/:folderId", folderHandler.UpdateFolder)
		folders.DELETE("/:folderId", folderHandler.DeleteFolder)

		// Note creation within folder
		folders.POST("/:folderId/notes", noteHandler.CreateNote)

		// Sharing
		folders.POST("/:folderId/share", folderHandler.ShareFolder)
		folders.DELETE("/:folderId/share/:userId", folderHandler.RevokeSharing)
	}
}
