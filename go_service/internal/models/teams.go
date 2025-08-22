package models

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	ID        int       `gorm:"primary_key;auto_increment;column:teamId" json:"id"`
	TeamName  string    `gorm:"size:150;not null;unique;column:teamName" json:"teamName"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt" json:"updatedAt"`
}

func (Team) TableName() string {
	return "Teams"
}

// Roster represents the junction table for Team-User many-to-many relationship
type Roster struct {
	ID       int       `gorm:"primary_key;auto_increment;column:rosterId" json:"id"`
	TeamID   int       `gorm:"not null;column:teamId" json:"teamId"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;column:userId" json:"userId"`
	IsLeader bool      `gorm:"default:false;column:isLeader" json:"isLeader"` // Match Node.js field

	// Foreign key relationships
	Team Team `gorm:"foreignKey:TeamID"`
}

// TableName overrides the table name used by Roster to `Rosters`
func (Roster) TableName() string {
	return "Rosters"
}
