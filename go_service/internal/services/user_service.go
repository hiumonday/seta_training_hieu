package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/machinebox/graphql"
)

// User represents a user from the user service
type User struct {
	ID       string `json:"userId"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// UserResponse represents the GraphQL user response
type UserResponse struct {
	User *User `json:"user"`
}

// CreateUserResponse represents the response from creating a user
type CreateUserResponse struct {
	Code    string `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *User  `json:"user"`
}

// UserService handles communication with the user service
type UserService struct {
	client  *graphql.Client
	baseURL string
}

// NewUserService creates a new user service client
func NewUserService() *UserService {
	baseURL := "http://localhost:4000/users"

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
	if response.User == nil || response.User.ID == "" {
		log.Printf("User not found with ID: %s", userID)
		return nil, fmt.Errorf("user not found with ID: %s", userID)
	}

	return &response, nil
}

// CreateUser creates a new user using GraphQL mutation
func (s *UserService) CreateUser(username, email, password, role string) (*UserResponse, error) {
	// Create GraphQL request
	req := graphql.NewRequest(`
        mutation CreateUser($input: CreateUserInput!) {
            createUser(input: $input) {
                code
                success
                message
                user {
                    userId
                    username
                    email
                    role
                }
            }
        }
    `)

	// Set variables
	req.Var("input", map[string]interface{}{
		"username": username,
		"email":    email,
		"password": password,
		"role":     role,
	})

	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute request
	var response struct {
		CreateUser CreateUserResponse `json:"createUser"`
	}

	if err := s.client.Run(ctx, req, &response); err != nil {
		log.Printf("GraphQL request failed: %v", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Check if user was created successfully
	if !response.CreateUser.Success {
		log.Printf("Failed to create user: %s", response.CreateUser.Message)
		return nil, fmt.Errorf("user creation failed: %s", response.CreateUser.Message)
	}

	// Create UserResponse from the createUser response
	return &UserResponse{
		User: response.CreateUser.User,
	}, nil
}
