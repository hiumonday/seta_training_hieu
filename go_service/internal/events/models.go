package events

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TeamEvent represents events related to team operations
type TeamEvent struct {
	EventType    string    `json:"eventType"`
	TeamID       string    `json:"teamId"`
	PerformedBy  string    `json:"performedBy"`
	TargetUserID string    `json:"targetUserId,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// AssetEvent represents events related to asset operations
type AssetEvent struct {
	EventType string    `json:"eventType"`
	AssetType string    `json:"assetType"`
	AssetID   string    `json:"assetId"`
	OwnerID   string    `json:"ownerId"`
	ActionBy  string    `json:"actionBy"`
	Timestamp time.Time `json:"timestamp"`
	// Additional fields for sharing events
	SharedWithUserID *string `json:"sharedWithUserId,omitempty"`
	AccessLevel      *string `json:"accessLevel,omitempty"`
}

// NewTeamEvent creates a new team event
func NewTeamEvent(eventType string, teamID int, performedBy uuid.UUID, targetUserID *uuid.UUID) *TeamEvent {
	event := &TeamEvent{
		EventType:   eventType,
		TeamID:      fmt.Sprintf("%d", teamID), // Convert int to string
		PerformedBy: performedBy.String(),
		Timestamp:   time.Now(),
	}
	if targetUserID != nil {
		targetStr := targetUserID.String()
		event.TargetUserID = targetStr
	}
	return event
}

// NewAssetEvent creates a new asset event
func NewAssetEvent(eventType, assetType string, assetID, ownerID, actionBy uuid.UUID) *AssetEvent {
	return &AssetEvent{
		EventType: eventType,
		AssetType: assetType,
		AssetID:   assetID.String(),
		OwnerID:   ownerID.String(),
		ActionBy:  actionBy.String(),
		Timestamp: time.Now(),
	}
}

// NewAssetSharingEvent creates a new asset sharing event
func NewAssetSharingEvent(eventType, assetType string, assetID, ownerID, actionBy, sharedWithUserID uuid.UUID, accessLevel string) *AssetEvent {
	event := NewAssetEvent(eventType, assetType, assetID, ownerID, actionBy)
	sharedWithStr := sharedWithUserID.String()
	event.SharedWithUserID = &sharedWithStr
	event.AccessLevel = &accessLevel
	return event
}