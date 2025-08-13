package router

import (
	"go_service/internal/handlers"

	"github.com/gin-gonic/gin"
)

// NoteRoutes defines routes for note management
func NoteRoutes(rg *gin.RouterGroup, noteHandler *handlers.NoteHandler) {
	notes := rg.Group("/notes")
	{
		notes.GET("/:noteId", noteHandler.GetNote)
		notes.PUT("/:noteId", noteHandler.UpdateNote)
		notes.DELETE("/:noteId", noteHandler.DeleteNote)

		// Sharing
		notes.POST("/:noteId/share", noteHandler.ShareNote)
		notes.DELETE("/:noteId/share/:userId", noteHandler.RevokeNoteSharing)
	}
}
