package models

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null" json:"ownerId"`
	FolderID  uuid.UUID `gorm:"type:uuid;not null" json:"folderId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
