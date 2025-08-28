package repositories

import (
	"go_service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TeamRepository định nghĩa các phương thức truy cập dữ liệu cho Team
type TeamRepository interface {
	FindByID(id uint64) (*models.Team, error)
	IsTeamLeader(teamID uint64, userID uuid.UUID) (bool, error)
	FindMember(teamID uint64, userID uuid.UUID) (*models.Roster, error)
	CountLeaders(teamID uint64) (int64, error)
	DeleteMember(roster *models.Roster) error
	UpdateMember(roster *models.Roster) error
	GetMembers(teamID uint64) ([]models.Roster, error)
	// Transactional methods
	CreateTeamWithLeaderInTx(tx *gorm.DB, team *models.Team, leaderID uuid.UUID) error
	AddMembersInTx(tx *gorm.DB, rosters []models.Roster) error
}

type teamRepository struct {
	db *gorm.DB
}

// NewTeamRepository tạo một instance mới của teamRepository
func NewTeamRepository(db *gorm.DB) TeamRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) FindByID(id uint64) (*models.Team, error) {
	var team models.Team
	err := r.db.First(&team, id).Error
	return &team, err
}

func (r *teamRepository) IsTeamLeader(teamID uint64, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.Roster{}).Where("\"teamId\" = ? AND \"userId\" = ? AND \"isLeader\" = ?", teamID, userID, true).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teamRepository) FindMember(teamID uint64, userID uuid.UUID) (*models.Roster, error) {
	var roster models.Roster
	if err := r.db.Where("\"teamId\" = ? AND \"userId\" = ?", teamID, userID).First(&roster).Error; err != nil {
		return nil, err
	}
	return &roster, nil
}

func (r *teamRepository) CountLeaders(teamID uint64) (int64, error) {
	var leaderCount int64
	err := r.db.Model(&models.Roster{}).Where("\"teamId\" = ? AND \"isLeader\" = ?", teamID, true).Count(&leaderCount).Error
	return leaderCount, err
}

func (r *teamRepository) DeleteMember(roster *models.Roster) error {
	return r.db.Delete(roster).Error
}

func (r *teamRepository) UpdateMember(roster *models.Roster) error {
	return r.db.Save(roster).Error
}

func (r *teamRepository) GetMembers(teamID uint64) ([]models.Roster, error) {
	var members []models.Roster
	err := r.db.Where("\"teamId\" = ?", teamID).Find(&members).Error
	return members, err
}

// --- Các phương thức dùng Transaction ---

func (r *teamRepository) CreateTeamWithLeaderInTx(tx *gorm.DB, team *models.Team, leaderID uuid.UUID) error {
	if err := tx.Create(team).Error; err != nil {
		return err
	}

	leaderRoster := models.Roster{
		TeamID:   team.ID,
		UserID:   leaderID,
		IsLeader: true,
	}

	return tx.Create(&leaderRoster).Error
}

func (r *teamRepository) AddMembersInTx(tx *gorm.DB, rosters []models.Roster) error {
	if len(rosters) == 0 {
		return nil
	}
	return tx.Create(&rosters).Error
}
