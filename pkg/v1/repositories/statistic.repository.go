package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type StatisticRepository struct {
	db *gorm.DB
}

func MustNewStatisticRepository(db *gorm.DB, init bool) models.StatisticRepository {
	if init {
		if err := db.AutoMigrate(&models.Session{}, &models.SessionMedia{}, &models.Action{}, &models.WatchInfo{}); err != nil {
			panic("failed to auto migrate statistic models")
		}
	}
	return &StatisticRepository{db: db}
}

func (r *StatisticRepository) CreateSession(ctx context.Context, session *models.Session) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateSession").Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(session).Error
}

func (r *StatisticRepository) CreateSessionMedia(
	ctx context.Context,
	sessionMedia *models.SessionMedia,
) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateSessionMedia").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(sessionMedia).Error
}

func (r *StatisticRepository) CreateActions(ctx context.Context, actions []*models.Action) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateActions").Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Create(actions).Error
}

func (r *StatisticRepository) CreateWatchInfos(
	ctx context.Context,
	watchInfos []*models.WatchInfo,
) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateWatchInfo").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Create(watchInfos).Error
}

func (r *StatisticRepository) GetSessionById(
	ctx context.Context,
	sessionId uuid.UUID,
) (*models.Session, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetSessionById").
			Observe(time.Since(t).Seconds())
	}()

	var rs models.Session
	if err := r.db.WithContext(ctx).
		Model(models.Session{}).
		Where("id = ?", sessionId).
		First(&rs).Error; err != nil {
		return nil, err
	}

	return &rs, nil
}

func (r *StatisticRepository) GetUncalculatedSessionMedia(
	ctx context.Context,
	cursor time.Time,
) ([]*models.SessionMedia, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUncalculatedSessionMedia").
			Observe(time.Since(t).Seconds())
	}()

	cursorCond := ""
	if cursor != (time.Time{}) {
		cursorCond = fmt.Sprintf("and created_at > '%s'", cursor.Format(time.RFC3339))
	}

	var rs []*models.SessionMedia
	query := fmt.Sprintf(`
		select *
		from session_media sm
		where status = 'new' %s
		order by created_at
		limit 100
	`, cursorCond)
	if err := r.db.WithContext(ctx).
		Raw(query).
		Scan(&rs).Error; err != nil {
		return nil, err
	}

	return rs, nil
}

func (r *StatisticRepository) GetUserAggreagatedMetricsInAction(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetAggregatedMetricsInput,
) (float64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetAggreagatedMetricsInAction").
			Observe(time.Since(t).Seconds())
	}()

	var rowNumConn string
	if input.Metric == models.StartMetric {
		rowNumConn = "and row_num = 1"
	}

	var query string
	rawQuery := fmt.Sprintf(`
		with actions as (
			select
				session_id,
				media_id,
				%%s as media_type,
				row_number() over (partition by session_id order by a.id desc) as row_num
			from actions a
				join session_media ma on a.session_media_id = ma.id
				%%s
			where a.user_id = '%s'
				and a.emitted_at >= '%s'
				and a.emitted_at <= '%s' and a.type = '%s'
		)
		select count(*) as result
		from sessions s
		join actions a on s.id = a.session_id %%s
		%%s
	`,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
		input.Metric,
	)

	if input.Filter != nil {
		whereConn := input.Filter.BuildQuery()
		switch input.Filter.MediaType {
		case models.VideoMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				rowNumConn,
				whereConn,
			)
		case models.AudioMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				rowNumConn,
				whereConn,
			)

		case models.StreamMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				rowNumConn,
				whereConn,
			)
		default:
			videoQuery := fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.VideoMediaType,
				),
				rowNumConn,
				whereConn,
			)
			audioQuery := fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.AudioMediaType,
				),
				rowNumConn,
				whereConn,
			)
			streamQuery := fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				rowNumConn,
				whereConn,
			)

			query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
		}
	} else {
		videoQuery := fmt.Sprintf(
			rawQuery,
			"'video'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.VideoMediaType,
			),
			rowNumConn,
			"",
		)
		audioQuery := fmt.Sprintf(
			rawQuery,
			"'audio'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.AudioMediaType,
			),
			rowNumConn,
			"",
		)
		streamQuery := fmt.Sprintf(
			rawQuery,
			"'stream'",
			"join live_stream_keys l on l.id = ma.media_id",
			rowNumConn,
			"",
		)

		query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
	}

	var result float64
	if err := r.db.
		Select("coalesce(sum(metric_value),0) as result").
		Table(fmt.Sprintf("(%s) as sub", query)).
		Scan(&result).Error; err != nil {
		return 0, err
	}

	return result, nil
}

func (r *StatisticRepository) GetUserAggreagatedMetricsInWatchInfo(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetAggregatedMetricsInput,
) (float64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetAggreagatedMetricsInAction").
			Observe(time.Since(t).Seconds())
	}()

	if input.Aggregation == models.AvarageAggregation {
		input.Aggregation = "avg"
	}

	var value, aggregationValue, watchCond string
	aggregationValue = "sum(metric_value)"
	if input.Metric == models.WatchTimeMetric {
		value = "wi.watch_time"
	} else if input.Metric == models.RetentionMetric {
		aggregationValue = "avg(metric_value)"
		value = "wi.retention"
	} else {
		value = "1"
		if input.Metric == models.ViewMetric {
			watchCond = "and wi.watch_time > 5"
		}
	}

	aggregationValue = fmt.Sprintf("coalesce(%s,0) as metric_value", aggregationValue)
	var query string
	rawQuery := fmt.Sprintf(`
		with watch_infos as (
			select
				session_id,
				media_id,
				%s as metric_value,
				%%s as media_type,
				row_number() over (partition  by session_media_id order by wi.created_at desc,wi.id desc) as row_num
			from watch_infos wi
				join session_media ma on wi.session_media_id = ma.id
				%%s
			where wi.user_id = '%s'
				and wi.created_at >= '%s'
				and wi.created_at <= '%s'
				%s
		)
		select %s
		from watch_infos wi
		join sessions s on s.id = wi.session_id
		%%s
	`,
		value,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
		watchCond,
		aggregationValue,
	)

	if input.Filter != nil {
		whereConn := input.Filter.BuildQuery()
		if whereConn != "" {
			whereConn += " and row_num = 1"
		} else {
			whereConn = "where row_num = 1"
		}

		switch input.Filter.MediaType {
		case models.VideoMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				whereConn,
			)
		case models.AudioMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				whereConn,
			)

		case models.StreamMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				whereConn,
			)
		default:
			videoQuery := fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.VideoMediaType,
				),
				whereConn,
			)
			audioQuery := fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.AudioMediaType,
				),
				whereConn,
			)
			streamQuery := fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				whereConn,
			)

			query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
		}
	} else {
		whereConn := "where row_num = 1"
		videoQuery := fmt.Sprintf(
			rawQuery,
			"'video'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.VideoMediaType,
			),
			whereConn,
		)
		audioQuery := fmt.Sprintf(
			rawQuery,
			"'audio'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.AudioMediaType,
			),
			whereConn,
		)
		streamQuery := fmt.Sprintf(
			rawQuery,
			"'stream'",
			"join live_stream_keys l on l.id = ma.media_id",
			whereConn,
		)

		query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
	}

	var result float64
	if err := r.db.
		Select(aggregationValue).
		Table(fmt.Sprintf("(%s) as sub", query)).
		Scan(&result).Error; err != nil {
		return 0, err
	}

	return result, nil
}

func (r *StatisticRepository) GetUserBreakdownMetricsInAction(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetBreakdownMetricsInput,
) ([]*models.MetricItem, int64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserBreakdownMetricsInAction").
			Observe(time.Since(t).Seconds())
	}()

	var rs []*models.MetricItem
	var total int64

	if input.SortBy == "" {
		input.SortBy = "metric_value"
	}

	var rowNumConn string
	if input.Metric == models.StartMetric {
		rowNumConn = "and row_num = 1"
	}

	var query string
	rawQuery := fmt.Sprintf(`
		with actions as (
			select
				session_id,
				media_id,
				%%s as media_type,
				row_number() over (partition by session_id order by a.id desc) as row_num
			from actions a
				join session_media ma on a.session_media_id = ma.id
				%%s
			where a.user_id = '%s'
				and a.emitted_at >= '%s'
				and a.emitted_at <= '%s' and a.type = '%s'
		)
		select 1 as metric_value,%s as dimension_value
		from sessions s
		join actions a on s.id = a.session_id %%s
		%%s
	`,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
		input.Metric,
		input.Breakdown,
	)

	if input.Filter != nil {
		whereConn := input.Filter.BuildQuery()
		switch input.Filter.MediaType {
		case models.VideoMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				rowNumConn,
				whereConn,
			)
		case models.AudioMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				rowNumConn,
				whereConn,
			)

		case models.StreamMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				rowNumConn,
				whereConn,
			)
		default:
			videoQuery := fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.VideoMediaType,
				),
				rowNumConn,
				whereConn,
			)
			audioQuery := fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.AudioMediaType,
				),
				rowNumConn,
				whereConn,
			)
			streamQuery := fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				rowNumConn,
				whereConn,
			)

			query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
		}

	} else {
		videoQuery := fmt.Sprintf(
			rawQuery,
			"'video'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.VideoMediaType,
			),
			rowNumConn,
			"",
		)
		audioQuery := fmt.Sprintf(
			rawQuery,
			"'audio'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.AudioMediaType,
			),
			rowNumConn,
			"",
		)
		streamQuery := fmt.Sprintf(
			rawQuery,
			"'stream'",
			"join live_stream_keys l on l.id = ma.media_id",
			rowNumConn,
			"",
		)

		query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
	}

	selectQuery := r.db.
		Select(
			"dimension_value, count(*) as metric_value, row_number() over (order by count(*) desc) as rn",
		).
		Table(fmt.Sprintf("(%s) as sub", query)).
		Group("dimension_value")
	if input.SortBy != "" {
		selectQuery = selectQuery.Order(fmt.Sprintf("%s %s", input.SortBy, input.OrderBy))
	}

	if !input.SumOthers || input.Limit == 1 {
		// if err := r.db.Raw(`select count(*) from (?)`, selectQuery).Scan(&total).Error; err != nil {
		// 	return nil, 0, err
		// }

		if err := selectQuery.
			Scan(&rs).Error; err != nil {
			return nil, 0, err
		}

		total = int64(len(rs))
		if len(rs) > input.Offset+input.Limit {
			rs = rs[input.Offset : input.Offset+input.Limit]
		} else {
			if len(rs) < input.Offset {
				rs = nil
			} else {
				rs = rs[input.Offset:]
			}
		}

	} else {
		subQuery := r.db.WithContext(ctx).
			Select(`
				case when rn < ? then dimension_value else 'Others' end as dimension_value,metric_value
		`, input.Limit).Table("(?) as sub", selectQuery)
		if err := r.db.Select(
			"dimension_value, sum(metric_value) as metric_value",
		).
			Table("(?) as sub1", subQuery).
			Limit(input.Limit).
			Group("dimension_value").
			Order(fmt.Sprintf("%s %s", input.SortBy, input.OrderBy)).
			Scan(&rs).Error; err != nil {
			return nil, 0, err
		}

		total = int64(len(rs))
	}

	return rs, total, nil
}

func (r *StatisticRepository) GetUserBreakdownMetricsInWatchInfo(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetBreakdownMetricsInput,
) ([]*models.MetricItem, int64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserBreakdownMetricsInWatchInfo").
			Observe(time.Since(t).Seconds())
	}()

	var rs []*models.MetricItem
	var total int64

	var selectBreakdown string
	if input.Breakdown == models.MediaTypeBreakdown {
		selectBreakdown = "case when type == 'video' then 'video' else 'stream' end "
	} else {
		selectBreakdown = input.Breakdown
	}

	var value, aggregationValue, watchCond string
	if input.Metric == models.WatchTimeMetric {
		aggregationValue = "sum(metric_value)"
		value = "wi.watch_time"
	} else if input.Metric == models.RetentionMetric {
		aggregationValue = "avg(metric_value)"
		value = "wi.retention"
	} else {
		aggregationValue = "count(*)"
		value = "1"
		if input.Metric == models.ViewMetric {
			watchCond = "and wi.watch_time > 5"
		}
	}

	var query string
	rawQuery := fmt.Sprintf(`
		with watch_infos as (
			select
				session_id,
				media_id,
				%s as metric_value,
				%%s as media_type,
				row_number() over (partition  by session_media_id order by wi.created_at desc,wi.id desc) as row_num
			from watch_infos wi
				join session_media ma on wi.session_media_id = ma.id
				%%s
			where wi.user_id = '%s'
				and wi.created_at >= '%s'
				and wi.created_at <= '%s'
				%s
		)
		select metric_value, %s as dimension_value
		from watch_infos wi
		join sessions s on s.id = wi.session_id
		%%s
	`,
		value,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
		watchCond,
		selectBreakdown,
	)

	if input.Filter != nil {
		whereConn := input.Filter.BuildQuery()
		if whereConn != "" {
			whereConn += " and row_num = 1"
		} else {
			whereConn = "where row_num = 1"
		}

		switch input.Filter.MediaType {
		case models.VideoMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				whereConn,
			)
		case models.AudioMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				whereConn,
			)

		case models.StreamMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				whereConn,
			)
		default:
			videoQuery := fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.VideoMediaType,
				),
				whereConn,
			)
			audioQuery := fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.AudioMediaType,
				),
				whereConn,
			)
			streamQuery := fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				whereConn,
			)

			query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
		}
	} else {
		whereConn := "where row_num = 1"
		videoQuery := fmt.Sprintf(
			rawQuery,
			"'video'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.VideoMediaType,
			),
			whereConn,
		)
		audioQuery := fmt.Sprintf(
			rawQuery,
			"'audio'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.AudioMediaType,
			),
			whereConn,
		)
		streamQuery := fmt.Sprintf(
			rawQuery,
			"'stream'",
			"join live_stream_keys l on l.id = ma.media_id",
			whereConn,
		)

		query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
	}

	selectQuery := r.db.
		Select(
			fmt.Sprintf(
				"dimension_value, %s as metric_value, row_number() over (order by count(*) desc) as rn",
				aggregationValue,
			),
		).
		Table(fmt.Sprintf("(%s) as sub", query)).
		Group("dimension_value")
	if input.SortBy != "" {
		selectQuery = selectQuery.Order(fmt.Sprintf("%s %s", input.SortBy, input.OrderBy))
	}

	if !input.SumOthers || input.Limit == 1 {
		// if err := r.db.Raw(`select count(*) from (?)`, selectQuery).Scan(&total).Error; err != nil {
		// 	return nil, 0, err
		// }

		if err := selectQuery.
			Scan(&rs).Error; err != nil {
			return nil, 0, err
		}

		total = int64(len(rs))
		if len(rs) > input.Offset+input.Limit {
			rs = rs[input.Offset : input.Offset+input.Limit]
		} else {
			if len(rs) < input.Offset {
				rs = nil
			} else {
				rs = rs[input.Offset:]
			}
		}

	} else {
		subQuery := r.db.WithContext(ctx).
			Select(`
				case when rn < ? then dimension_value else 'Others' end as dimension_value,metric_value
		`, input.Limit).Table("(?) as sub", selectQuery)
		if err := r.db.Select(
			"dimension_value, sum(metric_value) as metric_value",
		).
			Table("(?) as sub1", subQuery).
			Limit(input.Limit).
			Group("dimension_value").
			Order(fmt.Sprintf("%s %s", input.SortBy, input.OrderBy)).
			Scan(&rs).Error; err != nil {
			return nil, 0, err
		}

		total = int64(len(rs))
	}

	return rs, total, nil
}

func (r *StatisticRepository) GetUserOvertimeMetricsInAction(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetOvertimeMetricsInput,
) ([]*models.MetricItem, int64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserBreakMetricsInAction").
			Observe(time.Since(t).Seconds())
	}()

	var rs []*models.MetricItem
	var total int64
	var sessionJoinCond string
	if input.Metric == models.StartMetric {
		sessionJoinCond = "and row_num = 1"
	}

	interval := fmt.Sprintf("date_trunc('%s', wi.created_at at time zone 'utc')", input.Interval)
	var query string
	rawQuery := fmt.Sprintf(`
		with actions as (
			select
				session_id,
				media_id,
				1 as metric_value,
				%s as emitted_at,
				row_number() over (partition by session_id order by a.id desc) as row_num
			from actions a
				join session_media ma on a.session_media_id = ma.id
				%%s
			where a.user_id = '%s'
				and a.emitted_at >= '%s'
				and a.emitted_at <= '%s'
		)
		select metric_value, emitted_at
		from actions a
		%%s
		%%s
	`,
		interval,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
	)
	if input.Filter != nil {
		whereConn := input.Filter.BuildQuery()
		sessionJoin := fmt.Sprintf(
			"join sessions s ma on wi.session_id = s.id %s",
			sessionJoinCond,
		)

		switch input.Filter.MediaType {
		case models.VideoMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				sessionJoin,
				whereConn,
			)
		case models.AudioMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				sessionJoin,
				whereConn,
			)

		case models.StreamMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				sessionJoin,
				whereConn,
			)
		default:
			videoQuery := fmt.Sprintf(
				rawQuery,
				"'video'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.VideoMediaType,
				),
				sessionJoin,
				whereConn,
			)
			audioQuery := fmt.Sprintf(
				rawQuery,
				"'audio'",
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.AudioMediaType,
				),
				sessionJoin,
				whereConn,
			)
			streamQuery := fmt.Sprintf(
				rawQuery,
				"'stream'",
				"join live_stream_keys l on l.id = ma.media_id",
				sessionJoin,
				whereConn,
			)

			query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
		}
	} else {
		var whereConn string
		if input.Metric == models.StartMetric {
			whereConn = "where row_num = 1"
		}

		videoQuery := fmt.Sprintf(
			rawQuery,
			"'video'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.VideoMediaType,
			),
			"",
			whereConn,
		)
		audioQuery := fmt.Sprintf(
			rawQuery,
			"'audio'",
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.AudioMediaType,
			),
			"",
			whereConn,
		)
		streamQuery := fmt.Sprintf(
			rawQuery,
			"'stream'",
			"join live_stream_keys l on l.id = ma.media_id",
			"",
			whereConn,
		)

		query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
	}

	selectQuery := r.db.
		Select("count(*) as metric_value, emitted_at").
		Table(fmt.Sprintf("(%s) as sub", query)).
		Group("emitted_at")
	if input.SortBy != "" {
		selectQuery = selectQuery.Order(fmt.Sprintf("%s %s", input.SortBy, input.OrderBy))
	}

	if err := r.db.Select("count(*) as total").
		Table("(?) as sub", selectQuery).
		Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := selectQuery.
		Limit(input.Limit).
		Offset(input.Offset).
		Scan(&rs).Error; err != nil {
		return nil, 0, err
	}

	return rs, total, nil
}

func (r *StatisticRepository) GetUserOvertimeMetricsInWatchInfo(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetOvertimeMetricsInput,
) ([]*models.MetricItem, int64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserBreakMetricsInAction").
			Observe(time.Since(t).Seconds())
	}()

	var rs []*models.MetricItem
	var total int64

	interval := fmt.Sprintf("date_trunc('%s', wi.created_at at time zone 'utc')", input.Interval)
	var value, aggregationValue, watchCond string
	if input.Metric == models.WatchTimeMetric {
		aggregationValue = "sum(metric_value)"
		value = "wi.watch_time"
	} else if input.Metric == models.RetentionMetric {
		aggregationValue = "avg(metric_value)"
		value = "wi.retention"
	} else {
		aggregationValue = "count(*)"
		value = "1"
		if input.Metric == models.ViewMetric {
			watchCond = "and wi.watch_time > 5"
		}
	}

	var query string
	rawQuery := fmt.Sprintf(`
		with watch_infos as (
			select
				session_id,
				media_id,
				%s as metric_value,
				%s as emitted_at,
				row_number() over (partition  by session_media_id order by wi.created_at desc,wi.id desc) as row_num
			from watch_infos wi
				join session_media ma on wi.session_media_id = ma.id
				%%s
			where wi.user_id = '%s'
				and wi.created_at >= '%s'
				and wi.created_at <= '%s'
				%s
		)
		select metric_value, emitted_at
		from watch_infos wi
		%%s
		%%s
	`,
		value,
		interval,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
		watchCond,
	)

	if input.Filter != nil {
		whereConn := input.Filter.BuildQuery()
		if whereConn != "" {
			whereConn += " and row_num = 1"
		} else {
			whereConn = "where row_num = 1"
		}

		sessionJoin := "join sessions s on wi.session_id = s.id"
		switch input.Filter.MediaType {
		case models.VideoMediaType:
			query = fmt.Sprintf(
				rawQuery,
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				sessionJoin,
				whereConn,
			)
		case models.AudioMediaType:
			query = fmt.Sprintf(
				rawQuery,
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					input.Filter.MediaType,
				),
				sessionJoin,
				whereConn,
			)

		case models.StreamMediaType:
			query = fmt.Sprintf(
				rawQuery,
				"join live_stream_keys l on l.id = ma.media_id",
				sessionJoin,
				whereConn,
			)
		default:
			videoQuery := fmt.Sprintf(
				rawQuery,
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.VideoMediaType,
				),
				sessionJoin,
				whereConn,
			)
			audioQuery := fmt.Sprintf(
				rawQuery,
				fmt.Sprintf(
					"join media v on v.id = ma.media_id and v.type = '%s'",
					models.AudioMediaType,
				),
				sessionJoin,
				whereConn,
			)
			streamQuery := fmt.Sprintf(
				rawQuery,
				"join live_stream_keys l on l.id = ma.media_id",
				sessionJoin,
				whereConn,
			)

			query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)

		}
	} else {
		whereConn := "where row_num = 1"
		videoQuery := fmt.Sprintf(
			rawQuery,
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.VideoMediaType,
			),
			"",
			whereConn,
		)
		audioQuery := fmt.Sprintf(
			rawQuery,
			fmt.Sprintf(
				"join media v on v.id = ma.media_id and v.type = '%s'",
				models.AudioMediaType,
			),
			"",
			whereConn,
		)
		streamQuery := fmt.Sprintf(
			rawQuery,
			"join live_stream_keys l on l.id = ma.media_id",
			"",
			whereConn,
		)

		query = fmt.Sprintf(`
					(%s)
					union all
					(%s)
					union all
				    (%s)
			`, videoQuery, audioQuery, streamQuery)
	}

	selectQuery := r.db.
		Select(
			fmt.Sprintf(
				"%s as metric_value, emitted_at",
				aggregationValue,
			),
		).
		Table(fmt.Sprintf("(%s) as sub", query)).
		Group("emitted_at")
	if input.SortBy != "" {
		selectQuery = selectQuery.Order(fmt.Sprintf("%s %s", input.SortBy, input.OrderBy))
	}

	if err := r.db.Select("count(*) as total").
		Table("(?) as sub", selectQuery).
		Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := selectQuery.
		Limit(input.Limit).
		Offset(input.Offset).
		Scan(&rs).Error; err != nil {
		return nil, 0, err
	}

	return rs, total, nil
}

func (r *StatisticRepository) GetStatisticMedias(
	ctx context.Context,
	input models.GetStatisticMediasInput,
	userId uuid.UUID,
) ([]*models.Media, int64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetStatisticMedias").
			Observe(time.Since(t).Seconds())
	}()

	selectBreakdown := "media_id as dimension_value"
	var rs []*models.Media
	query := r.db.WithContext(ctx).
		Table("watch_infos wi").
		Select(fmt.Sprintf("1 as metric_value, %s,row_number() over (partition by session_media_id order by wi.watch_time desc,wi.id) as rn", selectBreakdown)).
		Joins("join session_media ma on wi.session_media_id = ma.id").
		Joins(`join media v on v.id = ma.media_id and v.status !='deleted' and v.type = ?`, input.Type).
		Where("wi.user_id = ? and wi.created_at >= ? and wi.created_at <= ? and (wi.watch_time > ?)",
			userId,
			input.From,
			input.To,
			models.MinWatchTime,
		)

	filterquery := r.db.WithContext(ctx).
		Select("dimension_value,count(*) as metric_value").
		Table("(?) as sub", query).
		Where("rn = 1").
		Group("dimension_value")

	var total int64
	if err := filterquery.
		Count(&total).
		Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Table("media v").
		Select(`
		id, user_id, title, description, qualities, "source", "type", public, status, "size", secret, is_mp4, mimetype, metadata, tags, avg_frame_rate, filter.metric_value as view, transcode_time, created_at, updated_at, player_theme_id, watch_time
		`).
		Joins(`
		join (
			?
		) as filter on v.id = filter.dimension_value
		`, filterquery).
		Preload("MediaQualities").
		Preload("Format").
		Preload("MediaThumbnail").
		Order("filter.metric_value desc").
		Limit(input.Limit).
		Offset(input.Offset).
		Find(&rs).Error; err != nil {
		return nil, 0, err
	}

	return rs, total, nil
}

func (r *StatisticRepository) GetSessionMediaBySessionIdAndMediaId(
	ctx context.Context,
	sessionId, mediaId uuid.UUID,
) (*models.SessionMedia, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetSessionMediaBySessionIdAndMediaId").
			Observe(time.Since(t).Seconds())
	}()

	var rs models.SessionMedia
	if err := r.db.WithContext(ctx).
		Model(models.SessionMedia{}).
		Where("session_id = ? and media_id = ?", sessionId, mediaId).
		First(&rs).Error; err != nil {
		return nil, err
	}

	return &rs, nil
}

func (r *StatisticRepository) GetSessionMediaLastWatchInfo(
	ctx context.Context,
	sessionMediaId uuid.UUID,
) (*models.WatchInfo, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetSessionMediaLastPausedWatchInfo").
			Observe(time.Since(t).Seconds())
	}()

	var rs models.WatchInfo
	if err := r.db.WithContext(ctx).
		Model(models.WatchInfo{}).
		Where("session_media_id = ?", sessionMediaId).
		Order("created_at desc").
		First(&rs).Error; err != nil {
		return nil, err
	}

	return &rs, nil
}

func (r *StatisticRepository) GetDataUsage(
	ctx context.Context,
	from, to time.Time,
	limit, offset uint64,
	interval string,
	userId uuid.UUID,
) ([]*models.DataUsage, int64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetDataUsage").
			Observe(time.Since(t).Seconds())
	}()

	var rs []*models.DataUsage
	var total int64

	countQuery := fmt.Sprintf(`
	select count(*) as total
	from (
		SELECT user_id
		FROM usages
		WHERE user_id = ? and created_at >= ? and created_at <= ?
		GROUP BY DATE_TRUNC('%s',created_at at Time zone 'utc'),user_id
	)`, interval)

	query := fmt.Sprintf(`
		SELECT user_id,
			coalesce(sum(delivery),0) as delivery,
			DATE_TRUNC('%s',created_at at Time zone 'utc') as created_at
		FROM usages
		WHERE user_id = ? and created_at >= ? and created_at <= ?
		GROUP BY DATE_TRUNC('%s',created_at at Time zone 'utc'),user_id
		ORDER BY DATE_TRUNC('%s',created_at at Time zone 'utc')
		OFFSET %d
		LIMIT %d
	`, interval, interval, interval, offset, limit)
	if err := r.db.Raw(countQuery, userId, from, to).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.DataUsage{}, 0, nil
	}

	if err := r.db.Raw(query, userId, from, to).Scan(&rs).Error; err != nil {
		return nil, 0, err
	}

	return rs, total, nil
}

func (r *StatisticRepository) GetNewUserCount(
	ctx context.Context,
	timeRange models.TimeRange,
) (int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetNewUserCount").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("created_at >= ? and created_at < ?", timeRange.Start, timeRange.End).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticRepository) GetTotalUserTopUps(
	ctx context.Context,
	timeRange models.TimeRange,
) (decimal.Decimal, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalUserTopUps").
			Observe(time.Since(t).Seconds())
	}()

	var totalTopUps decimal.Decimal
	if err := r.db.WithContext(ctx).
		Table("tx_ins").
		Select("COALESCE(SUM(credit),0)").
		Where("credit != 0 and status = true and created_at >= ? and created_at < ?", timeRange.Start, timeRange.End).
		Scan(&totalTopUps).Error; err != nil {
		return decimal.Zero, err
	}

	return totalTopUps, nil
}

func (r *StatisticRepository) GetMediaCount(
	ctx context.Context,
) (int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetVideoCount").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Media{}).
		Where("status = ?", models.DoneStatus).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticRepository) GetTotalUserCharge(
	ctx context.Context,
	timeRange models.TimeRange,
) (*models.Billing, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalUserChargeByType").
			Observe(time.Since(t).Seconds())
	}()

	var usageCost models.Billing
	err := r.db.WithContext(ctx).
		Model(&models.Usage{}).
		Select(`
			COALESCE(SUM(CAST(delivery AS numeric)), 0) as delivery,
			COALESCE(SUM(CAST(transcode AS numeric)), 0) as transcode,
			COALESCE(SUM(CAST(storage AS numeric)), 0) as storage,
			COALESCE(SUM(CAST(delivery_cost AS numeric)), 0) as delivery_cost,
			COALESCE(SUM(CAST(transcode_cost AS numeric)), 0) as transcode_cost,
			COALESCE(SUM(CAST(storage_cost AS numeric)), 0) as storage_cost,
			COALESCE(SUM(CAST(total_cost AS numeric)), 0) as total_cost
		`).
		Where("created_at >= ? AND created_at < ?", timeRange.Start, timeRange.End).
		Scan(&usageCost).Error

	return &usageCost, err
}

func (r *StatisticRepository) GetTotalVideoWatchTime(
	ctx context.Context,
	timeRange models.TimeRange,
) (float64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalVideoMediaWatchTime").
			Observe(time.Since(t).Seconds())
	}()

	var totalWatchTime float64
	query := `
	with watch_infos as (
		select session_media_id,wi.watch_time, row_number() over (partition by session_media_id order by wi.watch_time desc) as rn
		from watch_infos wi
		join session_media sm on wi.session_media_id = sm.id
	 	join media m on sm.media_id = m.id
		where sm.created_at >= ? and sm.created_at < ?
	 )
	 select COALESCE(sum(watch_time),0) as total_watch_time
	 from watch_infos
	 where rn = 1
	`
	if err := r.db.WithContext(ctx).Raw(query, timeRange.Start, timeRange.End).
		Scan(&totalWatchTime).Error; err != nil {
		return 0, err
	}

	return totalWatchTime, nil
}

func (r *StatisticRepository) GetTotalFailQualityCount(
	ctx context.Context,
	timeRange models.TimeRange,
) (int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalFailQualityCount").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.MediaQuality{}).
		Where("status = ? AND transcoded_at >= ? AND transcoded_at < ?", models.FailStatus, timeRange.Start, timeRange.End).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticRepository) GetTotalActiveUsers(
	ctx context.Context,
	timeRange models.TimeRange,
) (int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalActiveUsers").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("last_requested_at >= ?", timeRange.Start).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticRepository) GetTotalInactiveUsers(
	ctx context.Context,
	timeRange models.TimeRange,
) (int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalInactiveUsers").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("last_requested_at < ?", timeRange.Start).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticRepository) GetUserCount(
	ctx context.Context,
) (int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserCountByStatus").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticRepository) GetLiveStreamCount(
	ctx context.Context,
	timeRange models.TimeRange,
) (int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalLiveStream").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.LiveStreamMedia{}).
		Where("streamed_at >= ? AND streamed_at < ?", timeRange.Start, timeRange.End).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticRepository) UpdateSessionMediasStatus(
	ctx context.Context,
	sessionMediaIds []uuid.UUID,
	status string,
) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateSessionMediasCalculated").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(models.SessionMedia{}).
		Where("id in ?", sessionMediaIds).
		Update("status", status).Error
}
