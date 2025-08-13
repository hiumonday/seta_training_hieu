package router

import (
	"go_service/internal/handlers"

	"github.com/gin-gonic/gin"
)

// ImportRoutes defines routes for importing data
func ImportRoutes(rg *gin.RouterGroup, importHandler *handlers.ImportHandler) {
	rg.POST("/import-users", importHandler.ImportUsers)

}
