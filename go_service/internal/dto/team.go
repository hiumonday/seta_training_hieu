package dto

import (
	"github.com/google/uuid"
)

type CreateTeamReq struct {
	TeamName string      `json:"teamName" binding:"required"`
	UserIDs  []uuid.UUID `json:"userIds"`
}

type AddMemberReq struct {
	UserIDs []uuid.UUID `json:"userIds" binding:"required,min=1"`
}
