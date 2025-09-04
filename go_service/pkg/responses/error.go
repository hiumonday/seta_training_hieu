package responses

import "github.com/gin-gonic/gin"

type ErrorResponse struct {
	Data interface{} `json:"error,omitempty"`
}

func Error(c *gin.Context, status int, err error, message string) {
	errorRes := map[string]interface{}{
		"message": message,
		"error":   err.Error(),
	}

	c.JSON(status, ErrorResponse{Data: errorRes})
}
