package models

import (
	"time"

	"github.com/google/uuid"
)

var MaxJoinExclusiveRequestRetry int = 2

type JoinExclusiveProgramRequest struct {
	Id                   uuid.UUID `gorm:"primaryKey;type:uuid"`
	Email                string
	OrgName              string
	Role                 string
	Content              string
	StorageUsage         float64
	DeliveryUsage        float64
	UsedStreamPlatforms  string
	HeardAboutAIOZStream string
	Retry                int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewJoinExclusiveProgramRequest(
	orgName, email, role, content string,
	storageUsase, deliveryUsase float64,
	usedStreamPlatforms string,
	heardAboutAIOZStream string,
) *JoinExclusiveProgramRequest {
	return &JoinExclusiveProgramRequest{
		Id:                   uuid.New(),
		OrgName:              orgName,
		Email:                email,
		Role:                 role,
		Content:              content,
		StorageUsage:         storageUsase,
		DeliveryUsage:        deliveryUsase,
		UsedStreamPlatforms:  usedStreamPlatforms,
		HeardAboutAIOZStream: heardAboutAIOZStream,
		Retry:                1,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}
