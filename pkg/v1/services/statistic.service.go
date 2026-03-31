package services

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	ip_helper "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/ip"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/number"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
)

type StatisticService struct {
	statisticRepo       models.StatisticRepository
	mediaRepo           models.MediaRepository
	ipInfoRepo          models.IpInfoRepository
	liveStreamMediaRepo models.LiveStreamMediaRepository
	cdnUsageRepo        models.CdnUsageStatisticRepository

	ipHelper ip_helper.IpHelper

	dataCache *cache.Cache

	adminStatisticApiKey string
}

func NewStatisticService(
	statisticRepo models.StatisticRepository,
	mediaRepo models.MediaRepository,
	ipInfoRepo models.IpInfoRepository,
	liveStreamMediaRepo models.LiveStreamMediaRepository,
	cdnUsageRepo models.CdnUsageStatisticRepository,

	ipHelper ip_helper.IpHelper,

	adminStatisticApiKey string,
) *StatisticService {
	return &StatisticService{
		statisticRepo:       statisticRepo,
		mediaRepo:           mediaRepo,
		ipInfoRepo:          ipInfoRepo,
		liveStreamMediaRepo: liveStreamMediaRepo,
		cdnUsageRepo:        cdnUsageRepo,

		ipHelper:  ipHelper,
		dataCache: cache.New(5*time.Minute, 10*time.Minute),

		adminStatisticApiKey: adminStatisticApiKey,
	}
}

func (s *StatisticService) CreateWatchInfo(
	ctx context.Context,
	input models.CreateWatchInfoInput,
	sessionInfo models.SessionInfo,
) error {
	sessionMedia, userId, err := s.createSession(
		ctx,
		&sessionInfo,
		input.MediaId,
		input.MediaType,
	)
	if err != nil {
		return err
	}

	lastWatchInfo, err := s.statisticRepo.GetSessionMediaLastWatchInfo(
		ctx,
		sessionMedia.Id,
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NewInternalServerError(err)
	}

	watchInfos := make([]*models.WatchInfo, 0, len(input.Data))
	for _, item := range input.Data {
		var watchTime float64
		if lastWatchInfo == nil && item.MediaAt != 0 {
			continue
		}

		if lastWatchInfo != nil {
			if item.Paused && lastWatchInfo.Paused {
				watchTime = lastWatchInfo.WatchTime
			} else {
				if item.MediaAt-lastWatchInfo.MediaAt < 0.5 {
					continue
				}

				watchTime = lastWatchInfo.WatchTime + item.MediaAt - lastWatchInfo.MediaAt
				if watchTime < 0 {
					watchTime = lastWatchInfo.WatchTime
				}
			}
		}

		var duration float64
		if input.MediaType == models.VideoMediaType ||
			input.MediaType == models.AudioMediaType {
			media, err := s.mediaRepo.GetMediaById(ctx, input.MediaId)
			if err != nil {
				return response.NewInternalServerError(err)
			}

			duration = media.GetMediaDuration()
		}

		var retention float64
		if duration != 0 {
			retention = math.Min(math.Ceil(watchTime*100/duration), 100)
		}

		lastWatchInfo = models.NewWatchInfo(
			sessionMedia.Id,
			userId,
			item.Paused,
			item.MediaWidth,
			item.MediaHeight,
			watchTime,
			item.MediaAt,
			retention,
			input.Type,
		)

		watchInfos = append(watchInfos, lastWatchInfo)
	}

	if len(watchInfos) != 0 {
		if err := s.statisticRepo.CreateWatchInfos(ctx, watchInfos); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	return nil
}

func (s *StatisticService) CreateAction(
	ctx context.Context,
	input models.CreateActionInput,
	sessionInfo models.SessionInfo,
) error {
	sessionMedia, userId, err := s.createSession(
		ctx,
		&sessionInfo,
		input.MediaId,
		input.MediaType,
	)
	if err != nil {
		return err
	}

	actions := make([]*models.Action, 0, len(input.Data))
	for _, item := range input.Data {
		actions = append(
			actions,
			models.NewAction(sessionMedia.Id, userId, item.Type, item.MediaAt),
		)
	}

	if err := s.statisticRepo.CreateActions(ctx, actions); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *StatisticService) GetAggregatedMetrics(
	ctx context.Context,
	input models.GetAggregatedMetricsInput,
	authInfo models.AuthenticationInfo,
) (*models.MetricsContext, float64, error) {
	var rs float64
	switch input.Metric {
	case models.PlayMetric,
		models.StartMetric,
		models.EndMetric,
		models.ImpressionMetric:
		count, err := s.statisticRepo.GetUserAggreagatedMetricsInAction(
			ctx,
			authInfo.User.Id,
			input,
		)
		if err != nil {
			return nil, 0, response.NewInternalServerError(err)
		}

		if input.Aggregation == models.CountAggregation ||
			input.Aggregation == models.TotalAggregation {
			rs = count
			break
		} else if input.Aggregation == models.RateAggregation {
			newInput := input
			newInput.Aggregation = models.CountAggregation
			newInput.Metric = models.ImpressionMetric
			impressionCount, err := s.statisticRepo.GetUserAggreagatedMetricsInAction(
				ctx,
				authInfo.User.Id,
				newInput,
			)
			if err != nil {
				return nil, 0, response.NewInternalServerError(err)
			}

			if impressionCount == 0 {
				rs = 0
			} else {
				rs = count / impressionCount
			}
		}
	case models.WatchTimeMetric, models.ViewMetric:
		var err error
		rs, err = s.statisticRepo.GetUserAggreagatedMetricsInWatchInfo(
			ctx,
			authInfo.User.Id,
			input,
		)
		if err != nil {
			return nil, 0, response.NewInternalServerError(err)
		}
	}

	return &models.MetricsContext{
		Metric:      input.Metric,
		Aggregation: input.Aggregation,
		TimeFrame: &models.TimeFrame{
			From: input.From,
			To:   input.To,
		},
		Filter: input.Filter,
	}, rs, nil
}

func (s *StatisticService) GetBreakdownMetrics(
	ctx context.Context,
	input models.GetBreakdownMetricsInput,
	authInfo models.AuthenticationInfo,
) (*models.MetricsContext, []*models.MetricItem, int64, error) {
	start := time.Now()
	var rs []*models.MetricItem
	var err error
	var total int64
	key, err := input.Hash(authInfo.User.Id)
	if err == nil {
		if dt, found := s.dataCache.Get(key); found {
			data, ok := dt.(*models.MetricCacheItem)
			if ok {
				time.Sleep(1 * time.Second)
				return &models.MetricsContext{
					Metric:    input.Metric,
					Breakdown: input.Breakdown,
					Filter:    input.Filter,
					TimeFrame: &models.TimeFrame{
						From: input.From,
						To:   input.To,
					},
				}, data.Rs, data.Total, nil
			}
		}
	}

	switch input.Metric {
	case models.PlayMetric,
		models.PlayTotalMetric,
		models.StartMetric,
		models.EndMetric,
		models.ImpressionMetric:
		if input.Metric == models.PlayTotalMetric {
			input.Metric = models.PlayMetric
		}

		rs, total, err = s.statisticRepo.GetUserBreakdownMetricsInAction(
			ctx,
			authInfo.User.Id,
			input,
		)
		if err != nil {
			return nil, nil, 0, response.NewInternalServerError(err)
		}
	case models.PlayRateMetric:
		rs, total, err = s.statisticRepo.GetUserBreakdownMetricsInAction(
			ctx,
			authInfo.User.Id,
			input,
		)
		if err != nil {
			return nil, nil, 0, response.NewInternalServerError(err)
		}

		if len(rs) != 0 {
			newInput := input
			newInput.Metric = models.ImpressionMetric
			impressionRs, _, err := s.statisticRepo.GetUserBreakdownMetricsInAction(
				ctx,
				authInfo.User.Id,
				newInput,
			)
			if err != nil {
				return nil, nil, 0, response.NewInternalServerError(err)
			}

			for i, item := range rs {
				if impressionRs[i].MetricValue == 0 {
					item.MetricValue = 0
				} else {
					item.MetricValue = item.MetricValue / impressionRs[i].MetricValue
				}

				item.DimensionValue = models.PlayRateMetric
			}

		}
	case models.WatchTimeMetric, models.ViewMetric, models.RetentionMetric:
		rs, total, err = s.statisticRepo.GetUserBreakdownMetricsInWatchInfo(
			ctx,
			authInfo.User.Id,
			input,
		)
		if err != nil {
			return nil, nil, 0, response.NewInternalServerError(err)
		}
	}

	if time.Since(start) > 3*time.Second && key != "" {
		s.dataCache.Set(key, &models.MetricCacheItem{
			Rs:    rs,
			Total: total,
		}, cache.DefaultExpiration)
	}

	return &models.MetricsContext{
		Metric:    input.Metric,
		Breakdown: input.Breakdown,
		Filter:    input.Filter,
		TimeFrame: &models.TimeFrame{
			From: input.From,
			To:   input.To,
		},
	}, rs, total, nil
}

func (s *StatisticService) GetDataUsage(
	ctx context.Context,
	from, to time.Time,
	limit, offset uint64,
	interval string,
	authInfo models.AuthenticationInfo,
) ([]*models.DataUsage, int64, error) {
	dataUsages, total, err := s.statisticRepo.GetDataUsage(
		ctx,
		from,
		to,
		limit,
		offset,
		interval,
		authInfo.User.Id,
	)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}

	return dataUsages, total, nil
}

func (s *StatisticService) GetOvertimeMetrics(
	ctx context.Context,
	input models.GetOvertimeMetricsInput,
	authInfo models.AuthenticationInfo,
) (*models.MetricsContext, []*models.MetricItem, int64, error) {
	start := time.Now()
	var rs []*models.MetricItem
	var total int64
	var err error
	key, err := input.Hash(authInfo.User.Id)
	if err == nil {
		if dt, found := s.dataCache.Get(key); found {
			data, ok := dt.(*models.MetricCacheItem)
			if ok {
				time.Sleep(1 * time.Second)
				return &models.MetricsContext{
					Metric:   input.Metric,
					Interval: input.Interval,
					Filter:   input.Filter,
					TimeFrame: &models.TimeFrame{
						From: input.From,
						To:   input.To,
					},
				}, data.Rs, data.Total, nil
			}
		}
	}

	switch input.Metric {
	case models.PlayMetric,
		models.PlayTotalMetric,
		models.StartMetric,
		models.EndMetric,
		models.ImpressionMetric:
		rs, total, err = s.statisticRepo.GetUserOvertimeMetricsInAction(
			ctx,
			authInfo.User.Id,
			input,
		)
		if err != nil {
			return nil, nil, 0, response.NewInternalServerError(err)
		}
	case models.PlayRateMetric:
		rs, total, err = s.statisticRepo.GetUserOvertimeMetricsInAction(
			ctx,
			authInfo.User.Id,
			input,
		)
		if err != nil {
			return nil, nil, 0, response.NewInternalServerError(err)
		}

		newInput := input
		newInput.Metric = models.ImpressionMetric
		impressionRs, _, err := s.statisticRepo.GetUserOvertimeMetricsInAction(
			ctx,
			authInfo.User.Id,
			newInput,
		)
		if err != nil {
			return nil, nil, 0, response.NewInternalServerError(err)
		}

		for i, item := range rs {
			if impressionRs[i].MetricValue == 0 {
				item.MetricValue = 0
			} else {
				item.MetricValue = item.MetricValue / impressionRs[i].MetricValue
			}

			item.DimensionValue = models.PlayRateMetric
		}
	case models.WatchTimeMetric, models.ViewMetric, models.RetentionMetric:
		rs, total, err = s.statisticRepo.GetUserOvertimeMetricsInWatchInfo(
			ctx,
			authInfo.User.Id,
			input,
		)
		if err != nil {
			return nil, nil, 0, response.NewInternalServerError(err)
		}

	}

	if time.Since(start) > 3*time.Second && key != "" {
		s.dataCache.Set(key, &models.MetricCacheItem{
			Rs:    rs,
			Total: total,
		}, cache.DefaultExpiration)
	}

	return &models.MetricsContext{
		Metric:   input.Metric,
		Interval: input.Interval,
		Filter:   input.Filter,
		TimeFrame: &models.TimeFrame{
			From: input.From,
			To:   input.To,
		},
	}, rs, total, nil
}

func (s *StatisticService) createSession(
	ctx context.Context,
	sessionInfo *models.SessionInfo,
	mediaId uuid.UUID,
	mediaType string,
) (*models.SessionMedia, uuid.UUID, error) {
	var userId uuid.UUID
	var sessionMedia *models.SessionMedia
	session, err := s.statisticRepo.GetSessionById(ctx, sessionInfo.SessionId)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, uuid.Nil, response.NewInternalServerError(err)
	}

	if session == nil {
		ipInfo, err := s.ipInfoRepo.GetIpInfo(ctx, sessionInfo.IP)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, uuid.Nil, response.NewInternalServerError(err)
		}

		if ipInfo == nil || ipInfo.ExpiredAt.Before(time.Now().UTC()) {
			info, err := s.ipHelper.GetIpInfo(sessionInfo.IP)
			if err != nil {
				return nil, uuid.Nil, response.NewInternalServerError(err)
			}

			ipInfo = models.NewIpInfo(
				sessionInfo.IP,
				info.CountryCode,
				info.Region,
				info.Latitude,
				info.Latitude,
				info.Continent,
				info.City,
			)
			if err := s.ipInfoRepo.Save(ctx, ipInfo); err != nil {
				return nil, uuid.Nil, response.NewInternalServerError(err)
			}
		}

		session = models.NewSession(
			sessionInfo.SessionId,
			ipInfo.Continent,
			ipInfo.CountryCode,
			sessionInfo.IP,
			sessionInfo.DeviceType,
			sessionInfo.OperatorSystem,
			sessionInfo.Browser,
			sessionInfo.UserAgent,
			sessionInfo.SecUaBrowser,
		)
		if err := s.statisticRepo.CreateSession(ctx, session); err != nil {
			return nil, uuid.Nil, response.NewInternalServerError(err)
		}
	}

	sessionMedia, err = s.statisticRepo.GetSessionMediaBySessionIdAndMediaId(
		ctx,
		sessionInfo.SessionId,
		mediaId,
	)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, uuid.Nil, response.NewInternalServerError(err)
	}

	if sessionMedia == nil {
		sessionMedia = models.NewSessionMedia(sessionInfo.SessionId, mediaId)
		if err := s.statisticRepo.CreateSessionMedia(ctx, sessionMedia); err != nil {
			return nil, uuid.Nil, response.NewInternalServerError(err)
		}
	}

	sessionMedia, err = s.statisticRepo.GetSessionMediaBySessionIdAndMediaId(
		ctx,
		sessionInfo.SessionId,
		mediaId,
	)
	if err != nil {
		return nil, uuid.Nil, response.NewInternalServerError(err)
	}

	if mediaType == models.VideoMediaType ||
		mediaType == models.AudioMediaType {
		media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, uuid.Nil, response.NewNotFoundError(err)
			}

			return nil, uuid.Nil, response.NewInternalServerError(err)
		}

		userId = media.UserId
	} else {
		liveStreamMedia, err := s.liveStreamMediaRepo.GetById(ctx, mediaId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, uuid.Nil, response.NewNotFoundError(err)
			}

			return nil, uuid.Nil, response.NewInternalServerError(err)
		}

		userId = liveStreamMedia.UserId
	}

	return sessionMedia, userId, nil
}

func (r *StatisticService) GetStatisticMedias(
	ctx context.Context,
	input models.GetStatisticMediasInput,
	authInfo models.AuthenticationInfo,
) ([]*models.Media, int64, error) {
	media, total, err := r.statisticRepo.GetStatisticMedias(
		ctx,
		input,
		authInfo.User.Id,
	)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}

	return media, total, nil
}

func (r *StatisticService) CalculateMediaView(ctx context.Context) error {
	cursor := time.Time{}
	for {
		sessionMedias, err := r.statisticRepo.GetUncalculatedSessionMedia(
			ctx,
			cursor,
		)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		if len(sessionMedias) == 0 {
			break
		}

		mediaViewMap := make(map[uuid.UUID]int64)
		mediaWatchTimeMap := make(map[uuid.UUID]float64)
		countedList := make([]uuid.UUID, 0, len(sessionMedias))
		unCountedList := make([]uuid.UUID, 0, len(sessionMedias))
		for _, sessionMedia := range sessionMedias {
			watchInfo, err := r.statisticRepo.GetSessionMediaLastWatchInfo(
				ctx,
				sessionMedia.Id,
			)
			if err != nil && err != gorm.ErrRecordNotFound {
				return response.NewInternalServerError(err)
			}

			if watchInfo != nil {
				if watchInfo.WatchTime > models.MinWatchTime {
					if view, ok := mediaViewMap[sessionMedia.MediaId]; ok {
						mediaViewMap[sessionMedia.MediaId] = view + 1
					} else {
						mediaViewMap[sessionMedia.MediaId] = 1
					}
					countedList = append(countedList, sessionMedia.Id)
				} else {
					unCountedList = append(unCountedList, sessionMedia.Id)
				}

				if watchTime, ok := mediaWatchTimeMap[sessionMedia.MediaId]; ok {
					mediaWatchTimeMap[sessionMedia.MediaId] = watchTime + watchInfo.WatchTime
				} else {
					mediaWatchTimeMap[sessionMedia.MediaId] = watchInfo.WatchTime
				}
			} else {
				unCountedList = append(unCountedList, sessionMedia.Id)
			}
		}

		for mediaId, view := range mediaViewMap {
			watchTime := mediaWatchTimeMap[mediaId]
			if err := r.mediaRepo.UpdateMediaViewById(ctx, mediaId, view, watchTime); err != nil {
				return err
			}
		}

		if err := r.statisticRepo.UpdateSessionMediasStatus(ctx, unCountedList, models.UnCountedStatus); err != nil {
			return response.NewInternalServerError(err)
		}

		if err := r.statisticRepo.UpdateSessionMediasStatus(ctx, countedList, models.CountedStatus); err != nil {
			return response.NewInternalServerError(err)
		}

		cursor = sessionMedias[len(sessionMedias)-1].CreatedAt
	}

	return nil
}

func (s *StatisticService) GetMostViewedMedia(
	ctx context.Context,
	mediaType string,
	limit int,
	key string,
) ([]*models.MediaViewData, error) {
	if s.adminStatisticApiKey == "" || key != s.adminStatisticApiKey {
		return nil, response.NewHttpError(
			http.StatusForbidden,
			nil,
			"Forbidden access to this resource",
		)
	}

	viewData, err := s.mediaRepo.GetMostViewedMedia(ctx, mediaType, limit)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return viewData, nil
}

func (s *StatisticService) GetAdminStatisticData(
	ctx context.Context,
	timeRange models.TimeRange,
	key string,
) (*models.StatisticData, error) {
	if s.adminStatisticApiKey == "" || key != s.adminStatisticApiKey {
		return nil, response.NewHttpError(
			http.StatusForbidden,
			nil,
			"Forbidden access to this resource",
		)
	}

	lastHour := timeRange.End.Truncate(time.Hour)
	totalNewUser, err := s.statisticRepo.GetUserCount(ctx)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	totalTopUp, err := s.statisticRepo.GetTotalUserTopUps(ctx, timeRange)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	totalUserCharge, err := s.statisticRepo.GetTotalUserCharge(ctx, timeRange)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	totalMediaCount, err := s.statisticRepo.GetMediaCount(ctx)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	cdnUsage, err := s.cdnUsageRepo.GetCdnUsageStatistic(ctx, lastHour)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	totalVideoInteractions, err := s.statisticRepo.GetTotalVideoWatchTime(
		ctx,
		timeRange,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	totalTranscodeFail, err := s.statisticRepo.GetTotalFailQualityCount(
		ctx,
		timeRange,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	totalActiveUsers, err := s.statisticRepo.GetTotalActiveUsers(ctx, timeRange)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	totalLeaveUsers, err := s.statisticRepo.GetTotalInactiveUsers(
		ctx,
		timeRange,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	totalLiveStreamCount, err := s.statisticRepo.GetLiveStreamCount(
		ctx,
		timeRange,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return &models.StatisticData{
		Metrics: &models.Metrics{
			TotalUser: totalNewUser,
			TotalUsersTopUp: totalTopUp.Div(decimal.NewFromFloat(math.Pow10(18))).
				InexactFloat64(),
			TotalUsersCharge: number.Round(
				totalUserCharge.StorageCost.
					Add(totalUserCharge.TranscodeCost).
					Add(totalUserCharge.DeliveryCost).
					Div(decimal.NewFromFloat(math.Pow10(18))).
					InexactFloat64(), 6),
			TotalActiveUsers: totalActiveUsers,
			TotalLeaveUsers:  totalLeaveUsers,

			TotalStorageFee: number.Round(
				cdnUsage.CdnStorageCredit.
					Div(decimal.NewFromFloat(math.Pow10(18))).
					InexactFloat64(), 6),
			TotalStorageCapacity: cdnUsage.TotalStorage,
			TotalStorageChargedUser: number.Round(
				totalUserCharge.StorageCost.
					Div(decimal.NewFromFloat(math.Pow10(18))).
					InexactFloat64(), 6),

			TotalDeliveryFee: number.Round(
				cdnUsage.CdnDeliveryCredit.
					Div(decimal.NewFromFloat(math.Pow10(18))).
					InexactFloat64(), 6),
			TotalDeliveryCapacity: cdnUsage.TotalDelivery,
			TotalDeliveryChargedUser: number.Round(
				totalUserCharge.DeliveryCost.
					Div(decimal.NewFromFloat(math.Pow10(18))).
					InexactFloat64(), 6),
			TotalTranscodeFee:  decimal.Zero.InexactFloat64(),
			TotalTranscodeTime: totalUserCharge.Transcode,
			TotalTranscodeChargedUser: number.Round(
				totalUserCharge.TranscodeCost.
					Div(decimal.NewFromFloat(math.Pow10(18))).
					InexactFloat64(), 6),
			TotalVideosCount:       totalMediaCount,
			TotalVideoInteractions: totalVideoInteractions,
			TotalTranscodeFail:     totalTranscodeFail,
			TotalLivestreamHours: number.Round(
				cdnUsage.TotalLiveStreamDuration/60,
				2,
			),
			TotalLivestream: totalLiveStreamCount,
		},
		From: timeRange.Start.Unix(),
		To:   timeRange.End.Unix(),
	}, nil
}
