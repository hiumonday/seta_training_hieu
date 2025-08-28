package services

import (
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
func (s *TeamService) CreateTeam(teamName string, userIDs []uuid.UUID, creatorID uuid.UUID) (*models.Team, []uuid.UUID, error) {
	team := &models.Team{TeamName: teamName}
	if err := s.repo.CreateTeam(team); err != nil {
		return nil, nil, err
	}

	// Add creator as leader
	leaderRoster := &models.Roster{TeamID: team.ID, UserID: creatorID, IsLeader: true}
	if err := s.repo.AddMemberToTeam(leaderRoster); err != nil {
		return nil, nil, err
	}

	addedMembers := []uuid.UUID{creatorID}
	failedMembers := []uuid.UUID{}

	for _, userID := range userIDs {
		if userID == creatorID {
			continue
		}
		roster := &models.Roster{TeamID: team.ID, UserID: userID, IsLeader: false}
		if err := s.repo.AddMemberToTeam(roster); err != nil {
			failedMembers = append(failedMembers, userID)
		} else {
			addedMembers = append(addedMembers, userID)
		}
	}

	return team, failedMembers, nil
}

// //  adds members to a team with validation
// func (s *TeamService) AddMemberToTeam(teamID uint64, userIDs []uuid.UUID, currentUserID uuid.UUID) ([]map[string]interface{}, int, int, int, error) {
//     // Validate team and leadership (omitted for brevity; implement checks)
//     results := []map[string]interface{}{}
//     addedCount, existingCount, failedCount := 0, 0, 0

//     for _, userID := range userIDs {
//         // Check user exists, not already in team, etc.
//         // Add to team via repo
//         // Update counts and results
//     }

//     if addedCount > 0 && s.producer != nil {
//         // Emit Kafka event
//     }

//     return results, addedCount, existingCount, failedCount, nil
// }
