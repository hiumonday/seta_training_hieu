package database

import (
	"fmt"
	"go_service/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(dsn string) (*gorm.DB, error) {
	DB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = DB.AutoMigrate(&models.Team{})

	if err != nil {

		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return DB, nil
}
