package responses

import "github.com/gin-gonic/gin"

type Response struct {
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

// // NewSuccessResponse creates a successful API response
// func NewSuccessResponse(message string, data interface{}) APIResponse {
// 	return APIResponse{
// 		Success: true,
// 		Message: message,
// 		Data:    data,
// 	}
// }

// // NewErrorResponse creates an error API response
// func NewErrorResponse(error string, details string) APIResponse {
// 	return APIResponse{
// 		Success: false,
// 		Error:   error,
// 		Details: details,
// 	}
// }

func JSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, Response{
		Data: data,
	})
}
