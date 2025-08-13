package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/machinebox/graphql"
)

// UserResponse represents the GraphQL user response
type UserResponse struct {
	User struct {
		UserId   string `json:"userId"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	} `json:"user"`
}

// UserService handles communication with the user service
type UserService struct {
	client  *graphql.Client
	baseURL string
}

// NewUserService creates a new user service client
func NewUserService() *UserService {
	baseURL := os.Getenv("USER_SERVICE_URL")

	log.Printf("Initializing UserService with URL: %s", baseURL)
	client := graphql.NewClient(baseURL)

	return &UserService{
		client:  client,
		baseURL: baseURL,
	}
}

// GetUserByID fetches a user by their ID
func (s *UserService) GetUserByID(userID string) (*UserResponse, error) {
	log.Printf("Fetching user with ID: %s", userID)

	// Create GraphQL request
	req := graphql.NewRequest(`
        query GetUser($userId: ID!) {
            user(userId: $userId) {
                userId
                username
                email
                role
            }
        }
    `)

	// Set variables
	req.Var("userId", userID)

	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute request
	var response UserResponse
	log.Printf("Sending GraphQL request to %s", s.baseURL)

	if err := s.client.Run(ctx, req, &response); err != nil {
		log.Printf("GraphQL request failed: %v", err)
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	log.Printf("GraphQL response received: %+v", response)

	// Check if user was found
	if response.User.UserId == "" {
		log.Printf("User not found with ID: %s", userID)
		return nil, fmt.Errorf("user not found with ID: %s", userID)
	}

	return &response, nil
}
