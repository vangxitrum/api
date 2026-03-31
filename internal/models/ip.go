package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IpInfoRepository interface {
	Save(ctx context.Context, ip *IpInfo) error

	GetIpInfo(ctx context.Context, ip string) (*IpInfo, error)
}

type IpInfo struct {
	Id          uuid.UUID `json:"id"             gorm:"type:uuid;primaryKey"`
	Ip          string    `json:"ip"`
	CountryCode string    `json:"country_code"`
	Region      string    `json:"region_name"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Continent   string    `json:"continent_name"`
	City        string    `json:"city_name"`
	ExpiredAt   time.Time `json:"expired_at"`
}

func NewIpInfo(
	ip string,
	countryCode string,
	region string,
	latitude float64,
	longitude float64,
	continent string,
	city string,
) *IpInfo {
	return &IpInfo{
		Id:          uuid.New(),
		Ip:          ip,
		CountryCode: countryCode,
		Region:      region,
		Latitude:    latitude,
		Longitude:   longitude,
		Continent:   continent,
		City:        city,
		ExpiredAt:   time.Now().Add(15 * 24 * time.Hour), // 15 days
	}
}
