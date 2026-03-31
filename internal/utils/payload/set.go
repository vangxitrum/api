package utils

import (
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

type FilterPayload struct {
	Limit  uint64
	SortBy string
	Order  string
}

func SetDefaultsFilter(payload *FilterPayload, defaultLimit uint64, defaultSortBy string, defaultOrder string) {
	if payload.Limit == 0 || payload.Limit > models.MaxPageLimit {
		payload.Limit = defaultLimit
	}

	if payload.SortBy == "" {
		payload.SortBy = defaultSortBy
	}

	if payload.Order == "" {
		payload.Order = defaultOrder
	}
}
