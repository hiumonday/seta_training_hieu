package services

import (
	"context"
	"errors"
	"go_service/internal/kafka"
	"go_service/internal/models"
	"go_service/internal/redisclient"
	"go_service/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamService struct {
	repo        *repositories.TeamRepository
	userService *UserService
	producer    *kafka.Producer
	redisClient *redisclient.TeamCache
}

func NewTeamService(db *gorm.DB, producer *kafka.Producer, redisClient *redisclient.TeamCache) *TeamService {
	return &TeamService{
		repo:        repositories.NewTeamRepository(db),
		userService: NewUserService(),
		producer:    producer,
		redisClient: redisClient,
	}
}

// Creates a new team and adds members
func (s *TeamService) CreateTeam(ctx context.Context, teamName string, userIDs []uuid.UUID, creatorID uuid.UUID) (*models.Team, []uuid.UUID, error) {
	team := &models.Team{TeamName: teamName}
	if err := s.repo.CreateTeam(ctx, team); err != nil {
		return nil, nil, err
	}

	// Add creator as leader
	leaderRoster := models.Roster{TeamID: team.ID, UserID: creatorID, Role: "MANAGER"}
	if err := s.repo.AddMemberToTeam(ctx, []models.Roster{leaderRoster}); err != nil {
		return nil, nil, err
	}

	failedMembers := []uuid.UUID{}
	if len(userIDs) > 0 {
		rosters := make([]models.Roster, 0, len(userIDs))
		for _, userID := range userIDs {
			rosters = append(rosters, models.Roster{TeamID: team.ID, UserID: userID, Role: "MEMBER"})
		}
		if err := s.repo.AddMemberToTeam(ctx, rosters); err != nil {
			// Nếu lỗi, giả sử tất cả đều fail
			failedMembers = userIDs
		}
	}

	return team, failedMembers, nil
}

// adds members to a team with validation
func (s *TeamService) AddMemberToTeam(ctx context.Context, teamID uuid.UUID, userIDs []uuid.UUID, currentUserID uuid.UUID) (int, int, []uuid.UUID, error) {
	currentUserRole, err := s.repo.GetUserRoleInTeam(ctx, teamID.String(), currentUserID.String())
	if err != nil || (currentUserRole != "MANAGER" && currentUserRole != "MAIN_MANAGER") {
		return 0, len(userIDs), userIDs, errors.New("you are not a manager")
	}

	addedCount := 0
	failedMembers := []uuid.UUID{}
	rosters := make([]models.Roster, 0, len(userIDs))
	for _, userID := range userIDs {
		rosters = append(rosters, models.Roster{TeamID: teamID, UserID: userID, Role: "MEMBER"})
	}
	if err := s.repo.AddMemberToTeam(ctx, rosters); err != nil {
		// Nếu lỗi, giả sử tất cả đều fail
		failedMembers = userIDs
		return 0, len(userIDs), failedMembers, nil
	}
	addedCount = len(userIDs)
	return addedCount, 0, failedMembers, nil
}
