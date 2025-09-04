package repositories

import (
	"context"
	"go_service/internal/models"

	"gorm.io/gorm"
)

type TeamRepository struct {
	db *gorm.DB
}

func NewTeamRepository(db *gorm.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) GetTeamByID(ctx context.Context, teamID int) (*models.Team, error) {
	var team models.Team
	if err := r.db.WithContext(ctx).First(&team, teamID).Error; err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *TeamRepository) GetUserRoleInTeam(ctx context.Context, teamID string, userID string) (string, error) {
	var member models.Roster
	err := r.db.WithContext(ctx).
		Select("role").
		Where("teamId = ? AND userId = ?", teamID, userID).
		First(&member).Error

	if err != nil {
		return "", err
	}
	return member.Role, nil
}

// inserts a new team into the database
func (r *TeamRepository) CreateTeam(ctx context.Context, team *models.Team) error {
	return r.db.WithContext(ctx).Create(team).Error
}

// inserts a roster entry for a team member
func (r *TeamRepository) AddMemberToTeam(ctx context.Context, roster []models.Roster) error {
	tx := r.db.WithContext(ctx).Begin()
	if err := tx.Create(&roster).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
}
