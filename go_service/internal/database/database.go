package database

import (
	"fmt"
	"go_service/internal/models"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func CreateDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)
}

func Connect(dsn string) (*gorm.DB, error) {
	DB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = DB.AutoMigrate(&models.Team{}, &models.Roster{}, &models.Folder{}, &models.Note{}, &models.FolderShare{}, &models.NoteShare{})

	if err != nil {

		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return DB, nil
}
