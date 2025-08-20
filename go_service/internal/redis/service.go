package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go_service/internal/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	client *redis.Client
}

// NewService creates a new Redis service
func NewService(addr, password string, db int) *Service {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil
	}

	log.Println("Successfully connected to Redis")
	return &Service{client: client}
}

// Team Member Cache Methods

// SetTeamMembers sets the list of team members in cache
func (s *Service) SetTeamMembers(ctx context.Context, teamID int, userIDs []uuid.UUID) error {
	key := fmt.Sprintf("team:%d:members", teamID)
	
	// Convert UUIDs to strings
	members := make([]interface{}, len(userIDs))
	for i, userID := range userIDs {
		members[i] = userID.String()
	}

	// Clear existing members and set new ones
	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)
	if len(members) > 0 {
		pipe.LPush(ctx, key, members...)
	}
	pipe.Expire(ctx, key, 24*time.Hour) // Set expiration
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Failed to set team members for team %d: %v", teamID, err)
		return err
	}

	log.Printf("Updated team members cache for team %d", teamID)
	return nil
}

// AddTeamMember adds a member to the team cache
func (s *Service) AddTeamMember(ctx context.Context, teamID int, userID uuid.UUID) error {
	key := fmt.Sprintf("team:%d:members", teamID)
	
	err := s.client.LPush(ctx, key, userID.String()).Err()
	if err != nil {
		log.Printf("Failed to add member %s to team %d: %v", userID, teamID, err)
		return err
	}

	// Refresh expiration
	s.client.Expire(ctx, key, 24*time.Hour)
	
	log.Printf("Added member %s to team %d cache", userID, teamID)
	return nil
}

// RemoveTeamMember removes a member from the team cache
func (s *Service) RemoveTeamMember(ctx context.Context, teamID int, userID uuid.UUID) error {
	key := fmt.Sprintf("team:%d:members", teamID)
	
	err := s.client.LRem(ctx, key, 0, userID.String()).Err()
	if err != nil {
		log.Printf("Failed to remove member %s from team %d: %v", userID, teamID, err)
		return err
	}

	log.Printf("Removed member %s from team %d cache", userID, teamID)
	return nil
}

// GetTeamMembers retrieves team members from cache
func (s *Service) GetTeamMembers(ctx context.Context, teamID int) ([]uuid.UUID, error) {
	key := fmt.Sprintf("team:%d:members", teamID)
	
	members, err := s.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		log.Printf("Failed to get team members for team %d: %v", teamID, err)
		return nil, err
	}

	userIDs := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		userID, err := uuid.Parse(member)
		if err != nil {
			log.Printf("Invalid UUID in team members cache: %s", member)
			continue
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}

// Asset Metadata Cache Methods

// SetFolderMetadata caches folder metadata
func (s *Service) SetFolderMetadata(ctx context.Context, folder *models.Folder) error {
	key := fmt.Sprintf("folder:%s", folder.ID.String())
	
	data, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Failed to marshal folder metadata: %v", err)
		return err
	}

	err = s.client.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to cache folder metadata for %s: %v", folder.ID, err)
		return err
	}

	log.Printf("Cached folder metadata for %s", folder.ID)
	return nil
}

// GetFolderMetadata retrieves folder metadata from cache
func (s *Service) GetFolderMetadata(ctx context.Context, folderID uuid.UUID) (*models.Folder, error) {
	key := fmt.Sprintf("folder:%s", folderID.String())
	
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		log.Printf("Failed to get folder metadata for %s: %v", folderID, err)
		return nil, err
	}

	var folder models.Folder
	err = json.Unmarshal([]byte(data), &folder)
	if err != nil {
		log.Printf("Failed to unmarshal folder metadata: %v", err)
		return nil, err
	}

	return &folder, nil
}

// InvalidateFolderMetadata removes folder metadata from cache
func (s *Service) InvalidateFolderMetadata(ctx context.Context, folderID uuid.UUID) error {
	key := fmt.Sprintf("folder:%s", folderID.String())
	return s.client.Del(ctx, key).Err()
}

// SetNoteMetadata caches note metadata
func (s *Service) SetNoteMetadata(ctx context.Context, note *models.Note) error {
	key := fmt.Sprintf("note:%s", note.ID.String())
	
	data, err := json.Marshal(note)
	if err != nil {
		log.Printf("Failed to marshal note metadata: %v", err)
		return err
	}

	err = s.client.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to cache note metadata for %s: %v", note.ID, err)
		return err
	}

	log.Printf("Cached note metadata for %s", note.ID)
	return nil
}

// GetNoteMetadata retrieves note metadata from cache
func (s *Service) GetNoteMetadata(ctx context.Context, noteID uuid.UUID) (*models.Note, error) {
	key := fmt.Sprintf("note:%s", noteID.String())
	
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		log.Printf("Failed to get note metadata for %s: %v", noteID, err)
		return nil, err
	}

	var note models.Note
	err = json.Unmarshal([]byte(data), &note)
	if err != nil {
		log.Printf("Failed to unmarshal note metadata: %v", err)
		return nil, err
	}

	return &note, nil
}

// InvalidateNoteMetadata removes note metadata from cache
func (s *Service) InvalidateNoteMetadata(ctx context.Context, noteID uuid.UUID) error {
	key := fmt.Sprintf("note:%s", noteID.String())
	return s.client.Del(ctx, key).Err()
}

// Access Control Cache Methods

// SetAssetACL caches asset access control list
func (s *Service) SetAssetACL(ctx context.Context, assetID uuid.UUID, acl map[string]string) error {
	key := fmt.Sprintf("asset:%s:acl", assetID.String())
	
	data, err := json.Marshal(acl)
	if err != nil {
		log.Printf("Failed to marshal asset ACL: %v", err)
		return err
	}

	err = s.client.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to cache asset ACL for %s: %v", assetID, err)
		return err
	}

	log.Printf("Cached asset ACL for %s", assetID)
	return nil
}

// GetAssetACL retrieves asset access control list from cache
func (s *Service) GetAssetACL(ctx context.Context, assetID uuid.UUID) (map[string]string, error) {
	key := fmt.Sprintf("asset:%s:acl", assetID.String())
	
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		log.Printf("Failed to get asset ACL for %s: %v", assetID, err)
		return nil, err
	}

	var acl map[string]string
	err = json.Unmarshal([]byte(data), &acl)
	if err != nil {
		log.Printf("Failed to unmarshal asset ACL: %v", err)
		return nil, err
	}

	return acl, nil
}

// AddAssetAccess adds or updates access for a user to an asset
func (s *Service) AddAssetAccess(ctx context.Context, assetID, userID uuid.UUID, accessLevel string) error {
	key := fmt.Sprintf("asset:%s:acl", assetID.String())
	
	// Get existing ACL or create new one
	acl, err := s.GetAssetACL(ctx, assetID)
	if err != nil {
		return err
	}
	if acl == nil {
		acl = make(map[string]string)
	}

	// Add/update access
	acl[userID.String()] = accessLevel

	// Save back to cache
	return s.SetAssetACL(ctx, assetID, acl)
}

// RemoveAssetAccess removes access for a user from an asset
func (s *Service) RemoveAssetAccess(ctx context.Context, assetID, userID uuid.UUID) error {
	key := fmt.Sprintf("asset:%s:acl", assetID.String())
	
	// Get existing ACL
	acl, err := s.GetAssetACL(ctx, assetID)
	if err != nil {
		return err
	}
	if acl == nil {
		return nil // Nothing to remove
	}

	// Remove access
	delete(acl, userID.String())

	// Save back to cache
	return s.SetAssetACL(ctx, assetID, acl)
}

// Close closes the Redis connection
func (s *Service) Close() error {
	return s.client.Close()
}