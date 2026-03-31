package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/calculator"
	mails "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/mail"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/number"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payment"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/repositories"
)

type UsageService struct {
	db             *gorm.DB
	usageRepo      models.UsageRepository
	userRepo       models.UserRepository
	cdnFileRepo    models.CdnFileRepository
	paymentRepo    models.PaymentRepository
	mediaUsageRepo models.MediaUsageRepository

	CostPerStorage  int64
	CostPerDelivery int64

	userStatusMapping map[uuid.UUID]string
	paymentClient     *payment.PaymentClient
	storageHelper     storage.StorageHelper
	cdnUsageRepo      models.CdnUsageStatisticRepository
	mailHelper        mails.MailHelper
	mailRepo          models.MailRepository
}

func NewUsageService(
	db *gorm.DB,
	usageRepo models.UsageRepository,
	userRepo models.UserRepository,
	cdnFileRepo models.CdnFileRepository,
	paymentRepo models.PaymentRepository,
	mediaUsageRepo models.MediaUsageRepository,

	costPerStorage int64,
	costPerDelivery int64,

	userStatusMapping map[uuid.UUID]string,
	paymentClient *payment.PaymentClient,
	storageHelper storage.StorageHelper,
	cdnUsageRepo models.CdnUsageStatisticRepository,
	mailHelper mails.MailHelper,
	mailRepo models.MailRepository,
) *UsageService {
	return &UsageService{
		db:             db,
		usageRepo:      usageRepo,
		userRepo:       userRepo,
		cdnFileRepo:    cdnFileRepo,
		paymentRepo:    paymentRepo,
		mediaUsageRepo: mediaUsageRepo,
		cdnUsageRepo:   cdnUsageRepo,
		mailRepo:       mailRepo,

		CostPerStorage:  costPerStorage,
		CostPerDelivery: costPerDelivery,

		userStatusMapping: userStatusMapping,
		paymentClient:     paymentClient,
		storageHelper:     storageHelper,
		mailHelper:        mailHelper,
	}
}

func (u *UsageService) newUsageServiceWithTx(tx *gorm.DB) *UsageService {
	return &UsageService{
		db:             tx,
		usageRepo:      repositories.MustNewUsageRepository(tx, false),
		userRepo:       repositories.MustNewUserRepository(tx, false),
		cdnFileRepo:    repositories.MustNewCdnFileRepository(tx, false),
		paymentRepo:    repositories.MustNewPaymentRepository(tx, false),
		mediaUsageRepo: repositories.MustNewMediaUsageRepository(tx, false),
		cdnUsageRepo:   repositories.MustNewCdnUsageRepository(tx, false),
		mailRepo:       repositories.MustNewMailRepository(tx, false),

		CostPerStorage:  u.CostPerStorage,
		CostPerDelivery: u.CostPerDelivery,

		userStatusMapping: u.userStatusMapping,
		paymentClient:     u.paymentClient.NewPaymentClientWithTx(tx),
		storageHelper:     u.storageHelper,
		mailHelper:        u.mailHelper,
	}
}

func (s *UsageService) GetUserTopUps(
	ctx context.Context,
	input models.GetPaymentInput,
	authInfo models.AuthenticationInfo,
) ([]*models.TopUp, int64, error) {
	return s.paymentRepo.GetUserTopUps(ctx, authInfo.User.Id, input)
}

func (s *UsageService) GetUserBillings(
	ctx context.Context,
	input models.GetPaymentInput,
	authInfo models.AuthenticationInfo,
) ([]*models.Billing, int64, error) {
	billings, total, err := s.paymentRepo.GetUserBillings(ctx, authInfo.User.Id, input)
	if err != nil {
		return []*models.Billing{}, 0, response.NewInternalServerError(err)
	}

	mediaUsages, err := s.mediaUsageRepo.GetUserMediaUsages(ctx, authInfo.User.Id)
	if err != nil {
		return []*models.Billing{}, 0, response.NewInternalServerError(err)
	}

	for _, billing := range billings {
		for _, mediaUsage := range mediaUsages {
			if billing.CreatedAt.Day() == mediaUsage.CreatedAt.Day() {
				billing.Transcode += mediaUsage.Duration
				billing.TranscodeCost = billing.TranscodeCost.Add(mediaUsage.Cost)
			}
		}
	}

	return billings, total, nil
}

func (s *UsageService) GetUserUsageByInterval(
	ctx context.Context,
	from, to time.Time,
	authInfo models.AuthenticationInfo,
) (*models.Billing, error) {
	usage, err := s.paymentRepo.GetTotalUsageByInterval(ctx, authInfo.User.Id, from, to)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	mediaUsages, err := s.mediaUsageRepo.GetUserMediaUsages(ctx, authInfo.User.Id)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	for _, mediaUsage := range mediaUsages {
		if mediaUsage.CreatedAt.Before(from) || mediaUsage.CreatedAt.After(to) {
			continue
		}

		usage.Transcode += mediaUsage.Duration
		usage.TranscodeCost = usage.TranscodeCost.Add(mediaUsage.Cost)
	}

	return usage, nil
}

func (s *UsageService) CreateUsersUsage(ctx context.Context) error {
	users, err := s.userRepo.GetActiveAllUsers(ctx)
	if err != nil {
		return err
	}

	hour := time.Now().UTC().Truncate(time.Hour)
	for _, user := range users {
		if err := s.db.Transaction(func(tx *gorm.DB) error {
			usageService := s.newUsageServiceWithTx(tx)
			if _, err := usageService.usageRepo.GetUserLatestUsage(ctx, user.Id, hour); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					if _, err := usageService.createUserUsage(ctx, user.Id); err != nil {
						return err
					}
				} else {
					return err
				}
			}

			return nil
		}); err != nil {
			return nil
		}
	}

	if _, err := s.cdnUsageRepo.GetCdnUsageStatistic(ctx, hour); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, err := s.createCdnUsage(ctx, hour); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (s *UsageService) createUserUsage(
	ctx context.Context,
	userId uuid.UUID,
) (*models.Usage, error) {
	userStorage, err := s.cdnFileRepo.GetUserTotalStorage(ctx, userId)
	if err != nil {
		return nil, err
	}

	usage := models.NewUsage(userId, userStorage, 0, 0)
	if err := s.usageRepo.Create(ctx, usage); err != nil {
		return nil, err
	}

	return usage, nil
}

func (s *UsageService) CreateUsageLog(ctx context.Context, log *models.UsageLog) error {
	if err := s.usageRepo.CreateLog(ctx, log); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *UsageService) CalculateUsage(ctx context.Context) error {
	cursor := time.Now().Add(-time.Second * 15)
	usageLogs, err := s.usageRepo.GetUsageLogs(ctx, cursor)
	if err != nil {
		return err
	}

	var totalDelivery int64
	var totalTranscodeCost decimal.Decimal
	var transcodeDuration float64
	for _, log := range usageLogs {
		hour := log.CreatedAt.Truncate(time.Hour)
		usage, err := s.usageRepo.GetUserLatestUsage(ctx, log.UserId, hour)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if usage == nil {
			usage, err = s.createUserUsage(ctx, log.UserId)
			if err != nil {
				return err
			}
		}

		usage.Storage += log.Storage
		usage.Transcode += log.Transcode
		usage.Delivery += log.Delivery
		usage.TranscodeCost = usage.TranscodeCost.Add(log.TranscodeCost)
		usage.TotalCost = usage.StorageCost.Add(usage.DeliveryCost).Add(usage.TranscodeCost)
		usage.LivestreamDuration += log.LivestreamDuration
		totalDelivery += log.Delivery + log.Storage
		transcodeDuration += log.Transcode
		totalTranscodeCost = totalTranscodeCost.Add(log.TranscodeCost)
		if err := s.usageRepo.UpdateUsage(ctx, usage); err != nil {
			return err
		}

		if err := func() error {
			cdnStatic, err := s.cdnUsageRepo.GetCdnUsageStatistic(ctx, hour)
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}

			if cdnStatic == nil {
				cdnStatic, err = s.createCdnUsage(ctx, hour)
				if err != nil {
					return err
				}
			}

			cdnStatic.TotalStorage = cdnStatic.TotalStorage + log.Storage
			cdnStatic.CdnStorageCredit = decimal.NewFromInt(cdnStatic.TotalStorage).Mul(decimal.NewFromInt(models.HubCostPerStorage))
			cdnStatic.TotalDelivery = cdnStatic.TotalDelivery + log.Delivery
			cdnStatic.CdnDeliveryCredit = decimal.NewFromInt(cdnStatic.TotalDelivery).Mul(decimal.NewFromInt(models.HubCostPerDelivery))
			cdnStatic.TotalLiveStreamDuration = cdnStatic.TotalLiveStreamDuration + log.LivestreamDuration
			cdnStatic.Transcode = cdnStatic.Transcode + log.Transcode
			if err := s.cdnUsageRepo.UpdateCdnUsage(
				ctx,
				cdnStatic,
			); err != nil {
				return err
			}

			return nil
		}(); err != nil {
			slog.Error(
				"Error while updating cdn usage",
			)
		}
	}

	if err := s.usageRepo.DeleteUsageLogsByCursor(ctx, cursor); err != nil {
		return err
	}

	return nil
}

func (c *UsageService) createCdnUsage(
	ctx context.Context,
	hour time.Time,
) (*models.CdnUsageStatistic, error) {
	var credit decimal.Decimal

	balanceDetail, err := c.storageHelper.GetDetailBalance(ctx)
	if err != nil {
		previousHour := hour.Add(-1 * time.Hour)
		previousHourData, err := c.cdnUsageRepo.GetCdnUsageStatistic(ctx, previousHour)
		if err != nil {
			return nil, err
		}

		credit = previousHourData.RemainCredit

		slog.Warn("Using previous hour remain credit due to get balance fetch from cdn error",
			slog.Time("hour", hour),
			slog.Time("previous_hour", previousHour),
		)
	} else {
		credit, err = decimal.NewFromString(balanceDetail.Credit)
		if err != nil {
			return nil, fmt.Errorf("failed to parse credit: %w", err)
		}
	}

	totalStorage, err := c.cdnFileRepo.GetTotalSizeStorage(ctx)
	if err != nil {
		return nil, err
	}

	createdCdnUsage, err := c.cdnUsageRepo.CreateCdnUsageStatistic(ctx, &models.CdnUsageStatistic{
		Id:           uuid.New(),
		RemainCredit: credit,
		TotalStorage: totalStorage,
		CdnStorageCredit: decimal.NewFromInt(models.HubCostPerStorage).
			Mul(decimal.NewFromInt(totalStorage)),
		CreatedAt: hour,
	})
	if err != nil {
		return nil, err
	}

	return createdCdnUsage, nil
}

func (c *UsageService) ConvertUsdToAioz(
	ctx context.Context,
	usdAmount float64,
	authInfo models.AuthenticationInfo,
) (float64, error) {
	if authInfo.User.AiozPrice != 0 &&
		time.Now().UTC().Sub(authInfo.User.LastPriceUpdatedAt.Add(10*time.Minute)) < 0 {
		aiozAmount, _, err := c.paymentClient.ConvertUsdToAioz(
			ctx,
			decimal.NewFromFloat(usdAmount).Mul(decimal.NewFromFloat(math.Pow10(18))),
		)
		if err != nil {
			return 0, response.NewInternalServerError(err)
		}

		return number.Round(
			aiozAmount.Div(decimal.NewFromFloat(math.Pow10(18))).InexactFloat64(),
			6,
		), nil
	}

	aiozAmount, aiozPrice, err := c.paymentClient.ConvertUsdToAioz(
		ctx,
		decimal.NewFromFloat(usdAmount).Mul(decimal.NewFromFloat(math.Pow10(18))),
	)
	if err != nil {
		return 0, response.NewInternalServerError(err)
	}

	authInfo.User.AiozPrice = aiozPrice
	authInfo.User.LastPriceUpdatedAt = time.Now().UTC()
	if err := c.userRepo.UpdateUserPriceInfo(ctx, authInfo.User.Id, aiozPrice); err != nil {
		return 0, response.NewInternalServerError(err)
	}

	return number.Round(
		aiozAmount.Div(decimal.NewFromFloat(math.Pow10(18))).InexactFloat64(),
		6,
	), nil
}

func (s *UsageService) HandleUsage(ctx context.Context) error {
	lastHourUsages, err := s.usageRepo.GetLastHourUsagesByStatus(ctx, models.PendingStatus)
	if err != nil {
		return err
	}

	for _, usage := range lastHourUsages {
		if err := s.db.Transaction(func(tx *gorm.DB) error {
			usageService := s.newUsageServiceWithTx(tx)
			usage.StorageCost = decimal.NewFromInt(usage.Storage).
				Mul(decimal.NewFromInt(usageService.CostPerStorage))
			usage.DeliveryCost = decimal.NewFromInt(usage.Delivery).
				Mul(decimal.NewFromInt(usageService.CostPerDelivery))
			user, err := usageService.userRepo.GetUserById(ctx, usage.UserId)
			if err != nil {
				return err
			}

			usage.TotalCost = usage.TranscodeCost.Add(usage.StorageCost).Add(usage.DeliveryCost)
			usage.Status = models.SuccessStatus
			total := usage.StorageCost.Add(usage.DeliveryCost)
			if err := usageService.paymentClient.CreateTransaction(ctx, usage.UserId, total, models.PaymentTypeBill, usage.Id); err != nil {
				if err == payment.InsufficientBalanceError {
					usage.Status = models.FailStatus
				} else {
					return err
				}
			}

			if err := usageService.usageRepo.UpdateUsage(ctx, usage); err != nil {
				return err
			}

			if usage.Status == models.FailStatus {
				if err := usageService.userRepo.UpdateUserStatus(ctx, user.Id, models.BlockedStatus); err != nil {
					return err
				}

				usageService.userStatusMapping[user.Id] = models.BlockedStatus
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *UsageService) HandleFailUsage(ctx context.Context) error {
	blockedUsers, err := s.userRepo.GetUsersByStatus(ctx, models.BlockedStatus)
	if err != nil {
		return err
	}

	for _, user := range blockedUsers {
		if err := s.db.Transaction(func(tx *gorm.DB) error {
			usageService := s.newUsageServiceWithTx(tx)
			failUsages, err := usageService.usageRepo.GetUserUsagesByStatus(ctx, user.Id, models.FailStatus)
			if err != nil {
				return err
			}

			if len(failUsages) == 0 {
				return fmt.Errorf("no fail usage found for user %s", user.Id)
			}

			flag := false
			for i, usage := range failUsages {
				usage.Status = models.SuccessStatus
				if err := usageService.paymentClient.CreateTransaction(ctx, user.Id, usage.StorageCost.Add(usage.DeliveryCost), models.PaymentTypeBill, usage.Id); err != nil {
					if err == payment.InsufficientBalanceError {
						usage.Status = models.FailStatus
					} else {
						return err
					}
				}

				if err := usageService.usageRepo.UpdateUsage(ctx, usage); err != nil {
					return err
				}

				if i == len(failUsages)-1 && usage.Status == models.SuccessStatus {
					flag = true
				}

				if usage.Status == models.FailStatus {
					break
				}
			}

			if flag {
				if err := usageService.mailRepo.DeleteMailByUserMail(ctx, user.Email, mails.LowBalanceMailType); err != nil {
					return err
				}

				if err := usageService.mailRepo.DeleteMailByUserMail(ctx, user.Email, mails.OutOfBalanceMailType); err != nil {
					return err
				}

				if err := usageService.userRepo.UpdateUserStatus(ctx, user.Id, models.ActiveStatus); err != nil {
					return err
				}

				usageService.userStatusMapping[user.Id] = models.ActiveStatus
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *UsageService) HandleLowBalanceUsers(ctx context.Context) error {
	users, err := s.GetLowBalanceUsers(ctx)
	if err != nil {
		slog.Error("Error while getting users with balance issues", "err", err)
		return err
	}

	for _, user := range users {
		switch user.Status {
		case models.OutOfBalanceStatus:
			if err = s.sendMail(ctx, user, mails.OutOfBalanceMailType, models.ExpiredEmailOutOfBalanceTime, nil); err != nil {
				slog.Error("Error while sending out of balance email", "err", err)
			}
		case models.LowBalanceStatus:
			if err = s.sendMail(ctx, user, mails.LowBalanceMailType, models.ExpiredEmailLowBalanceTime, nil); err != nil {
				slog.Error("Error while sending low wallet balance email", "err", err)
			}
		}
	}

	return nil
}

func (s *UsageService) GetLowBalanceUsers(
	ctx context.Context,
) ([]*models.User, error) {
	lastDate := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	users, err := s.userRepo.GetActiveAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		wallet, err := s.paymentClient.GetWalletByUserId(ctx, user.Id)
		if err != nil {
			return nil, err
		}

		userHourlyUsage, err := s.usageRepo.GetUserLastHourUsage(ctx, user.Id)
		if err != nil {
			return nil, err
		}

		if wallet.Balance.LessThan(userHourlyUsage) {
			user.Status = models.OutOfBalanceStatus
			continue
		}

		userDailyUsage, err := s.usageRepo.GetUserTocalCostByDate(ctx, user.Id, lastDate)
		if err != nil {
			return nil, err
		}

		if wallet.Balance.LessThan(userDailyUsage) {
			user.Status = models.LowBalanceStatus
		}
	}

	return users, nil
}

func (s *UsageService) HandleContentOutOfBalanceUser(ctx context.Context) ([]*models.User, error) {
	var outOfBalanceUsers []*models.User
	outOfBalanceMails, err := s.mailRepo.GetMailsByType(ctx, mails.OutOfBalanceMailType)
	if err != nil {
		return nil, err
	}

	for _, mail := range outOfBalanceMails {
		if time.Now().UTC().After(mail.ExpiredAt) {
			user, err := s.userRepo.GetActiveUserByEmail(ctx, mail.Mail)
			if err != nil {
				return nil, err
			}

			lastHourUsage, err := s.usageRepo.GetUserLastHourUsage(ctx, user.Id)
			if err != nil {
				return nil, err
			}

			wallet, err := s.paymentClient.GetWalletByUserId(ctx, user.Id)
			if err != nil {
				return nil, err
			}

			if wallet.Balance.LessThan(lastHourUsage) {
				outOfBalanceUsers = append(outOfBalanceUsers, user)
			}

			mail.Status = models.DoneStatus
			if err := s.mailRepo.UpdateMail(ctx, mail); err != nil {
				return nil, err
			}
		}
	}

	return outOfBalanceUsers, nil
}

func (s *UsageService) HandleMonthlyReceipt(ctx context.Context) error {
	now := time.Now().UTC()
	var monthlyReceiptUser []models.User
	expiredTime := calculator.CalculateExpiredTimeToEndOfMonth()
	firstDayOfPreviousMonth := time.Date(now.Year(), now.Month()-1, 1, 7, 0, 0, 0, time.UTC)
	monthTime := firstDayOfPreviousMonth.Format("January 2006")
	firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 6, 0, 0, 0, time.UTC)

	timeRange := models.TimeRange{
		Start: firstDayOfPreviousMonth,
		End:   firstDayOfCurrentMonth,
	}
	userIds, err := s.usageRepo.GetUserIdsByTimeRange(ctx, timeRange)
	if err != nil {
		return err
	}
	for _, userId := range userIds {
		user, err := s.userRepo.GetUserById(ctx, userId)
		if err != nil {
			return err
		}
		existMail, err := s.mailRepo.GetMailByUserMail(ctx, user.Email, mails.MonthlyReceiptType)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if existMail == nil {
			monthlyReceiptUser = append(monthlyReceiptUser, *user)
		}
	}

	for _, user := range monthlyReceiptUser {
		totalCost, err := s.usageRepo.GetUserTotalCostByTimeRange(ctx, user.Id, timeRange)
		if err != nil {
			return err
		}
		if totalCost.TotalCost.Equal(decimal.Zero) &&
			totalCost.DeliveryCost.Equal(decimal.Zero) &&
			totalCost.TranscodeCost.Equal(decimal.Zero) &&
			totalCost.StorageCost.Equal(decimal.Zero) {
			continue
		}
		totalCostUSD := number.Round(calculator.ConvertPriceToUSD(totalCost.TotalCost), 4)
		totalDeliveryCostUSD := number.Round(
			calculator.ConvertPriceToUSD(totalCost.DeliveryCost),
			4,
		)
		totalTranscodeCostUSD := number.Round(
			calculator.ConvertPriceToUSD(totalCost.TranscodeCost),
			4,
		)
		totalStorageCostUSD := number.Round(calculator.ConvertPriceToUSD(totalCost.StorageCost), 4)
		if totalCostUSD < 1 {
			continue
		}

		if err := s.sendMail(ctx, &user, mails.MonthlyReceiptType, expiredTime, map[string]any{
			"Total":     totalCostUSD,
			"Delivery":  totalDeliveryCostUSD,
			"Transcode": totalTranscodeCostUSD,
			"Storage":   totalStorageCostUSD,
			"Month":     monthTime,
		}); err != nil {
			log.Err(err).Msg("Error while sending monthly receipt email")
		}
	}

	return nil
}

func (s *UsageService) sendMail(
	ctx context.Context,
	user *models.User,
	mailType string,
	expiredTime time.Duration,
	additionalData map[string]any,
) error {
	exist, err := s.mailRepo.GetMailByUserMail(ctx, user.Email, mailType)
	if err != nil && err != gorm.ErrRecordNotFound {
		return response.NewInternalServerError(err)
	}

	if exist == nil {
		if user.FirstName == "" {
			user.FirstName = user.Email
		}

		emailData := map[string]any{
			"FirstName": user.FirstName,
		}
		for key, value := range additionalData {
			emailData[key] = value
		}

		if err := s.mailHelper.SendEmail(
			ctx,
			[]string{user.Email},
			mailType,
			emailData,
		); err != nil {
			return response.NewInternalServerError(err)
		}

		expiredTime := time.Now().UTC().Add(expiredTime)
		mail := models.NewMail(
			user.Email,
			mailType,
			time.Now().UTC(),
			expiredTime,
		)

		if mail.Type == mails.OutOfBalanceMailType {
			mail.Status = models.PendingStatus
		}

		if err := s.mailRepo.CreateMail(
			ctx,
			mail,
		); err != nil {
			return response.NewInternalServerError(err)
		}
		return nil
	}
	return nil
}
