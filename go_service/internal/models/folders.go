package models

import (
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FolderName string    `gorm:"size:150;not null" json:"folderName"`
	OwnerID    uuid.UUID `gorm:"type:uuid;not null" json:"ownerId"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
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
