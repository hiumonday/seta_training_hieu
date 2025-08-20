package responses

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Details string      `json:"details,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse(message string, data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(error string, details string) APIResponse {
	return APIResponse{
		Success: false,
		Error:   error,
		Details: details,
	}
}
