package repositories

import (
	"go_service/internal/models"

	"gorm.io/gorm"
)

type TeamRepository struct {
	db *gorm.DB
}

func NewTeamRepository(db *gorm.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// inserts a new team into the database
func (r *TeamRepository) CreateTeam(team *models.Team) error {
	return r.db.Create(team).Error
}

// inserts a roster entry for a team member
func (r *TeamRepository) AddMemberToTeam(roster *models.Roster) error {
	return r.db.Create(roster).Error
}
