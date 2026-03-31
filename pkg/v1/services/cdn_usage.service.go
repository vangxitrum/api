package services

import (
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
)

type CdnUsageService struct {
	usageRepo     models.UsageRepository
	cdnFileRepo   models.CdnFileRepository
	cdnUsageRepo  models.CdnUsageStatisticRepository
	storageHelper storage.StorageHelper
	mediaRepo     models.MediaRepository
}

func NewCdnUsageService(
	usageRepo models.UsageRepository,
	cdnFileRepo models.CdnFileRepository,
	cdnUsageRepo models.CdnUsageStatisticRepository,
	storageHelper storage.StorageHelper,
	mediaRepo models.MediaRepository,
) *CdnUsageService {
	return &CdnUsageService{
		usageRepo:     usageRepo,
		cdnFileRepo:   cdnFileRepo,
		cdnUsageRepo:  cdnUsageRepo,
		storageHelper: storageHelper,
		mediaRepo:     mediaRepo,
	}
}
