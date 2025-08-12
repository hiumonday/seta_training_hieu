package database

import (
	"fmt"
	"go_service/internal/models"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// avoid use global variable
var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"), // Match Node.js service env var name
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	// close conection when shutdown app
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	// use log instead of fmt
	fmt.Println("Database connection successful.")

	// Create ENUM types first (if not exists)
	DB.Exec("DO $$ BEGIN CREATE TYPE access_level AS ENUM ('read', 'write'); EXCEPTION WHEN duplicate_object THEN null; END $$;")
	// DB.Exec("DO $$ BEGIN CREATE TYPE user_role AS ENUM ('manager', 'member'); EXCEPTION WHEN duplicate_object THEN null; END $$;")

	// Auto-migrate ONLY asset-related tables that Go service owns
	// Not migrate User, Team, or Roster models - they're managed by Node.js service
	// handle error
	DB.AutoMigrate(&models.Folder{}, &models.Note{}, &models.FolderShare{}, &models.NoteShare{})
}

// InitDB initializes the database connection and returns the DB instance
func InitDB() (*gorm.DB, error) {
	Connect()
	return DB, nil
}
