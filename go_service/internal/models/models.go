package models

import (
	"time"

	"github.com/google/uuid"
)

type AccessLevel string

const (
	Read  AccessLevel = "read"
	Write AccessLevel = "write"
)

type Team struct {
	ID        int       `gorm:"primary_key;auto_increment;column:teamId" json:"id"`
	TeamName  string    `gorm:"size:150;not null;unique;column:teamName" json:"teamName"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt" json:"updatedAt"`
}

// TableName overrides the table name used by Team to `Teams`
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

type Folder struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FolderName string    `gorm:"size:150;not null" json:"folderName"`
	OwnerID    uuid.UUID `gorm:"type:uuid;not null" json:"ownerId"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Note struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null" json:"ownerId"`
	FolderID  uuid.UUID `gorm:"type:uuid;not null" json:"folderId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// FolderShare represents sharing permissions for folders
type FolderShare struct {
	ID          uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FolderID    uuid.UUID   `gorm:"type:uuid;not null" json:"folderId"`
	UserID      uuid.UUID   `gorm:"type:uuid;not null" json:"userId"`
	AccessLevel AccessLevel `gorm:"type:access_level;not null" json:"accessLevel"`
	SharedByID  uuid.UUID   `gorm:"type:uuid;not null" json:"sharedById"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`

	// Foreign key relationships
	Folder Folder `gorm:"foreignKey:FolderID"`
}

// NoteShare represents sharing permissions for individual notes
type NoteShare struct {
	ID          uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	NoteID      uuid.UUID   `gorm:"type:uuid;not null" json:"noteId"`
	UserID      uuid.UUID   `gorm:"type:uuid;not null" json:"userId"`
	AccessLevel AccessLevel `gorm:"type:access_level;not null" json:"accessLevel"`
	SharedByID  uuid.UUID   `gorm:"type:uuid;not null" json:"sharedById"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`

	// Foreign key relationships
	Note Note `gorm:"foreignKey:NoteID"`
}
