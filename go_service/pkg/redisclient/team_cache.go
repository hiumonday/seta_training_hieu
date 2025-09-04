package redisclient

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TeamCache provides Redis caching functions for team data
type TeamCache struct {
	client *redis.Client
}

// NewTeamCache creates a new TeamCache instance
func NewTeamCache(client *redis.Client) *TeamCache {
	return &TeamCache{
		client: client,
	}
}

// GetTeamMembersKey returns the Redis key for team members
func (tc *TeamCache) GetTeamMembersKey(teamID uint64) string {
	return fmt.Sprintf("team:%d:members", teamID)
}

// GetMembers retrieves team members from Redis cache
func (tc *TeamCache) GetMembers(ctx context.Context, teamID uint64) ([]uuid.UUID, error) {
	if tc.client == nil {
		return nil, fmt.Errorf("Redis client not initialized")
	}

	key := tc.GetTeamMembersKey(teamID)

	// Get all members from Redis set
	memberStrings, err := tc.client.SMembers(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Cache miss
			return nil, nil
		}
		return nil, err
	}

	// Convert string IDs to UUID
	userIDs := make([]uuid.UUID, 0, len(memberStrings))
	for _, memberStr := range memberStrings {
		memberID, err := uuid.Parse(memberStr)
		if err != nil {
			log.Printf("Invalid UUID in cache: %s", memberStr)
			continue
		}
		userIDs = append(userIDs, memberID)
	}

	// Refresh expiration
	tc.client.Expire(ctx, key, 24*time.Hour)

	return userIDs, nil
}

// StoreMembers stores team members in Redis
func (tc *TeamCache) StoreMembers(ctx context.Context, teamID uint64, userIDs []interface{}) error {
	if tc.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	key := tc.GetTeamMembersKey(teamID)

	// Use pipeline for efficiency
	pipe := tc.client.Pipeline()
	pipe.Del(ctx, key)

	if len(userIDs) > 0 {
		pipe.SAdd(ctx, key, userIDs...)
		pipe.Expire(ctx, key, 24*time.Hour)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

// AddMember adds a member to the team members cache
func (tc *TeamCache) AddMember(ctx context.Context, teamID uint64, userID uuid.UUID) error {
	if tc.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	key := tc.GetTeamMembersKey(teamID)

	// Add member to set
	err := tc.client.SAdd(ctx, key, userID.String()).Err()
	if err != nil {
		return err
	}

	// Refresh expiration
	return tc.client.Expire(ctx, key, 24*time.Hour).Err()
}

// RemoveMember removes a member from the team members cache
func (tc *TeamCache) RemoveMember(ctx context.Context, teamID uint64, userID uuid.UUID) error {
	if tc.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	key := tc.GetTeamMembersKey(teamID)

	// Remove member from set
	return tc.client.SRem(ctx, key, userID.String()).Err()
}

// SMembers is a wrapper around the Redis SMembers command
func (tc *TeamCache) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return tc.client.SMembers(ctx, key)
}

// Expire is a wrapper around the Redis Expire command
func (tc *TeamCache) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return tc.client.Expire(ctx, key, expiration)
}

// Pipeline returns a Redis pipeline
func (tc *TeamCache) Pipeline() redis.Pipeliner {
	return tc.client.Pipeline()
}
