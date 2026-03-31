package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	mails "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/mail"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payment"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/random"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/token"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/wallet"
)

type UserService struct {
	userRepo             models.UserRepository
	emailConnectionRepo  models.EmailConnectionRepository
	walletConnectionRepo models.WalletConnectionRepository
	mailRepo             models.MailRepository
	usageRepo            models.UsageRepository
	exclusiveCodeRepo    models.ExclusiveCodeRepository

	mailHelper    mails.MailHelper
	tokenIssuer   *token.TokenIssuer
	paymentClient *payment.PaymentClient
	storageHelper storage.StorageHelper
}

func NewUserService(
	userRepo models.UserRepository,
	emailConnectionRepo models.EmailConnectionRepository,
	walletConnectionRepo models.WalletConnectionRepository,
	mailRepo models.MailRepository,
	usageRepo models.UsageRepository,
	exclusiveCodeRepo models.ExclusiveCodeRepository,

	mailHelper mails.MailHelper,
	tokenIssuer *token.TokenIssuer,
	paymentClient *payment.PaymentClient,
	storageHelper storage.StorageHelper,
) *UserService {
	if err := exclusiveCodeRepo.GenerateExclusiveCodes(context.Background()); err != nil {
		slog.Error("Failed to get exclusive codes", slog.Any("error", err))
	}

	return &UserService{
		userRepo:             userRepo,
		emailConnectionRepo:  emailConnectionRepo,
		walletConnectionRepo: walletConnectionRepo,
		mailRepo:             mailRepo,
		usageRepo:            usageRepo,
		exclusiveCodeRepo:    exclusiveCodeRepo,

		mailHelper:    mailHelper,
		tokenIssuer:   tokenIssuer,
		paymentClient: paymentClient,
		storageHelper: storageHelper,
	}
}

func (s *UserService) SendLoginCode(
	ctx context.Context, email string,
) (int64, error) {
	user, err := s.userRepo.GetActiveUserByEmail(
		ctx,
		email,
	)
	if err != nil && !errors.Is(
		err,
		gorm.ErrRecordNotFound,
	) {
		return 0, response.NewInternalServerError(err)
	}
	if user == nil {
		return 0, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("User not found. Please sign up first."),
			"User not found. Please sign up first.",
		)
	} else {
		if user.FirstName == "" {
			user.FirstName = user.Email
		}
		cn, err := s.emailConnectionRepo.GetEmailConnectionByEmail(
			ctx,
			email,
		)
		if err != nil && !errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return 0, response.NewInternalServerError(err)
		}

		if cn != nil {
			if cn.ExpiredAt.After(time.Now().UTC()) {
				if cn.UpdatedAt.Add(1 * time.Minute).Before(time.Now()) {
					cn.UpdatedAt = time.Now()
					return s.handleEmail(ctx, email, mails.LoginMailType, cn, user.FirstName)
				}

				return cn.CreatedAt.Unix(), nil
			}

			cn.Code = random.GenerateRandomCode(6)
			cn.ExpiredAt = time.Now().UTC().Add(models.ExpiredEmailTime)
			cn.MaxRetries = models.MaxRetriesLogin
		} else {
			cn = models.NewEmailConnection(email, models.MaxRetriesLogin)
			if err := s.emailConnectionRepo.Create(
				ctx,
				cn,
			); err != nil {
				return 0, response.NewInternalServerError(err)
			}
		}
		return s.handleEmail(ctx, email, mails.LoginMailType, cn, user.FirstName)
	}
}

func (s *UserService) UpdateUsersFreeBalance(
	ctx context.Context,
) error {
	return s.userRepo.UpdateUsersFreeBalance(ctx)
}

func (s *UserService) RequestJoinExclusiveProgram(
	ctx context.Context,
	req *models.JoinExclusiveProgramRequest,
) error {
	existed, err := s.exclusiveCodeRepo.GetJoinRequestByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NewInternalServerError(err)
	}

	if existed != nil {
		existed.UpdatedAt = time.Now()
		existed.Retry++
		if existed.Retry > models.MaxJoinExclusiveRequestRetry {
			return nil
		}
	}

	if err := s.mailHelper.SendEmail(ctx, models.AdminMailList, mails.RequestJoinExclusiveProgramType, map[string]any{
		"OrgName":              req.OrgName,
		"Email":                req.Email,
		"Role":                 req.Role,
		"Content":              req.Content,
		"StorageUsage":         req.StorageUsage,
		"DeliveryUsage":        req.DeliveryUsage,
		"UsedStreamPlatforms":  req.UsedStreamPlatforms,
		"HeardAboutAIOZStream": req.HeardAboutAIOZStream,
	}); err != nil {
		return response.NewInternalServerError(err)
	}

	if existed != nil {
		if err := s.exclusiveCodeRepo.UpdateJoinRequest(ctx, existed); err != nil {
			return response.NewInternalServerError(err)
		}
	} else {
		if err := s.exclusiveCodeRepo.CreateRequestJoinExclusiveProgram(ctx, req); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	return nil
}

func (s *UserService) UseExclusiveCode(
	ctx context.Context,
	code string,
	authInfo models.AuthenticationInfo,
) error {
	if authInfo.User.ExclusiveCode != "" {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("You have already used an exclusive code."),
			"You have already used an exclusive code.",
		)
	}

	exclusiveCode, err := s.exclusiveCodeRepo.GetExclusiveCodeByCode(
		ctx,
		code,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewHttpError(http.StatusBadRequest, err, "Invalid exclusive code.")
		}
		return response.NewInternalServerError(err)
	}

	if exclusiveCode.Status != models.ActiveStatus {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Invalid exclusive code."),
			"Used exclusive code.",
		)
	}

	if err := s.paymentClient.UpdateWalletFreeBalance(ctx, authInfo.User.WalletAddress, exclusiveCode.Amount); err != nil {
		return response.NewInternalServerError(err)
	}

	authInfo.User.ExclusiveCode = code
	authInfo.User.UpdatedAt = time.Now().UTC()
	if err := s.userRepo.UpdateUser(ctx, authInfo.User); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.exclusiveCodeRepo.UpdateExclusiveCodeStatus(
		ctx,
		code,
		models.DeletedStatus,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *UserService) SendSignUpCode(
	ctx context.Context, email string,
) (int64, error) {
	user, err := s.userRepo.GetActiveUserByEmail(
		ctx,
		email,
	)
	if err != nil && !errors.Is(
		err,
		gorm.ErrRecordNotFound,
	) {
		return 0, response.NewInternalServerError(err)
	}

	if user != nil && user.Status != models.DeletedStatus {
		return 0, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("User already existed."),
			"User already existed.",
		)
	} else {
		cn, err := s.emailConnectionRepo.GetEmailConnectionByEmail(
			ctx,
			email,
		)
		if err != nil && !errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return 0, response.NewInternalServerError(err)
		}

		if cn != nil {
			if cn.ExpiredAt.After(time.Now().UTC()) {
				if cn.UpdatedAt.Add(1 * time.Minute).Before(time.Now()) {
					cn.UpdatedAt = time.Now()
					return s.handleEmail(ctx, email, mails.LoginMailType, cn, email)
				}

				return cn.CreatedAt.Unix(), nil
			}

			cn.Code = random.GenerateRandomCode(6)
			cn.ExpiredAt = time.Now().UTC().Add(models.ExpiredEmailTime)
			cn.MaxRetries = models.MaxRetriesSignUp
		} else {
			cn = models.NewEmailConnection(email, models.MaxRetriesSignUp)

			if err := s.emailConnectionRepo.Create(
				ctx,
				cn,
			); err != nil {
				return 0, response.NewInternalServerError(err)
			}
		}
		return s.handleEmail(ctx, email, mails.SignUpMailType, cn, email)
	}
}

func (s *UserService) handleEmail(
	ctx context.Context,
	email string,
	mailType string,
	cn *models.EmailConnection,
	firstName string,
) (int64, error) {
	if err := s.mailHelper.SendEmail(
		ctx,
		[]string{email},
		mailType,
		map[string]any{
			"Code":      cn.Code,
			"FirstName": firstName,
		},
	); err != nil {
		return 0, response.NewInternalServerError(err)
	}

	if err := s.emailConnectionRepo.UpdateEmailConnection(
		ctx,
		cn,
	); err != nil {
		return 0, response.NewInternalServerError(err)
	}

	mail := models.NewMail(
		cn.Email,
		mailType,
		cn.CreatedAt,
		cn.ExpiredAt,
	)
	if err := s.mailRepo.CreateMail(
		ctx,
		mail,
	); err != nil {
		return 0, response.NewInternalServerError(err)
	}

	return cn.ExpiredAt.Unix(), nil
}

func (s *UserService) GetUserById(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetUserById(ctx, id)
}

func (s *UserService) VerifyCode(
	ctx context.Context,
	email, code string,
) (string, string, error) {
	isTestAccount := models.TestAccounts[email]
	var (
		cn  *models.EmailConnection
		err error
	)

	if isTestAccount {
		cn = &models.EmailConnection{
			Email:      email,
			Code:       models.TestAccountCode,
			MaxRetries: models.MaxRetriesLogin,
			ExpiredAt:  time.Now().UTC().Add(models.ExpiredEmailTime),
		}
	} else {
		cn, err = s.emailConnectionRepo.GetEmailConnectionByEmail(
			ctx,
			email,
		)
		if err != nil {
			if errors.Is(
				err,
				gorm.ErrRecordNotFound,
			) {
				return "", "", response.NewNotFoundError(err)
			}

			return "", "", response.NewInternalServerError(err)
		}
	}

	if cn.ExpiredAt.Before(time.Now().UTC()) {
		return "", "", response.CodeExpiredError
	}

	if cn.MaxRetries == 0 {
		return "", "", response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Too many retries. Please try again later."),
			"Too many retries. Please try again later.",
		)
	}

	if cn.Code != code {
		if err := s.emailConnectionRepo.UpdateRetries(ctx, email); err != nil {
			return "", "", response.NewInternalServerError(err)
		}

		return "", "", response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Invalid code."),
			"Invalid code.",
		)
	}

	user, err := s.userRepo.GetActiveUserByEmail(
		ctx,
		email,
	)
	if err != nil && !errors.Is(
		err,
		gorm.ErrRecordNotFound,
	) {
		return "", "", response.NewInternalServerError(err)
	}

	var accessToken, refreshToken string
	if user == nil {
		user = models.NewUser(email, "", "")
		user.WalletAddress, err = s.paymentClient.CreateWallet(ctx, user.Id, decimal.Zero)
		if err != nil {
			return "", "", response.NewInternalServerError(err)
		}

		if err := s.userRepo.Create(
			ctx,
			user,
		); err != nil {
			return "", "", response.NewInternalServerError(err)
		}

		if err := s.usageRepo.Create(ctx, models.NewUsage(user.Id, 0, 0, 0)); err != nil {
			return "", "", response.NewInternalServerError(err)
		}
	}

	accessToken, refreshToken, _, err = s.tokenIssuer.CreateCredential(user.Id)
	if err != nil {
		return "", "", response.NewInternalServerError(err)
	}

	cn.Code = ""
	cn.ExpiredAt = time.Now().UTC()
	if err := s.emailConnectionRepo.UpdateEmailConnection(
		ctx,
		cn,
	); err != nil {
		return "", "", response.NewInternalServerError(err)
	}

	return accessToken, refreshToken, nil
}

func (s *UserService) GetAdminStatisticData(ctx context.Context) error {
	return nil
}

func (s *UserService) GetChallenge(
	ctx context.Context,
	walletAddress common.Address,
) (string, error) {
	user, err := s.userRepo.GetActiveUserByWalletConnection(
		ctx,
		walletAddress.Hex(),
	)
	if err != nil && !errors.Is(
		err,
		gorm.ErrRecordNotFound,
	) {
		return "", response.NewInternalServerError(err)
	}

	if user == nil {
		return "", response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("User not found. Please sign up first."),
			"User not found. Please sign up first.",
		)
	}
	wc, err := s.walletConnectionRepo.GetWalletConnectionByWalletAddress(
		ctx,
		walletAddress.Hex(),
	)
	if err != nil && !errors.Is(
		err,
		gorm.ErrRecordNotFound,
	) {
		return "", response.NewInternalServerError(err)
	}

	challenge := wallet.GetChallenge()
	if wc == nil {
		return "", response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Wallet not found."),
			"Wallet not found.",
		)
	} else {
		wc.Challenge = challenge
		if err := s.walletConnectionRepo.UpdateWalletConnection(
			ctx,
			wc,
		); err != nil {
			return "", response.NewInternalServerError(err)
		}
	}

	return wallet.FormatMessageForSigning(
		walletAddress.Hex(),
		challenge,
	), nil
}

func (s *UserService) VerifyChallenge(
	ctx context.Context,
	walletAddress common.Address,
	signature string,
) (string, string, error) {
	wc, err := s.walletConnectionRepo.GetWalletConnectionByWalletAddress(
		ctx,
		walletAddress.Hex(),
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return "", "", response.NewNotFoundError(err)
		}

		return "", "", response.NewInternalServerError(err)
	}

	if err := wallet.VerifySignature(
		walletAddress,
		signature,
	); err != nil {
		return "", "", response.InvalidSignatureError
	}

	if ok := wallet.VerifySig(
		walletAddress.Hex(),
		signature,
		"",
		wc.Challenge,
		wallet.SigningSig,
	); !ok {
		return "", "", response.InvalidSignatureError
	}

	user, err := s.userRepo.GetActiveUserByWalletConnection(
		ctx,
		walletAddress.Hex(),
	)
	if err != nil && !errors.Is(
		err,
		gorm.ErrRecordNotFound,
	) {
		return "", "", response.NewInternalServerError(err)
	}

	if user == nil {
		return "", "", response.NewHttpError(
			400,
			fmt.Errorf("User not found."),
			"User not found.",
		)
	}

	accessToken, refreshToken, _, err := s.tokenIssuer.CreateCredential(user.Id)
	if err != nil {
		return "", "", response.NewInternalServerError(err)
	}

	return accessToken, refreshToken, nil
}

func (s *UserService) RefreshToken(
	ctx context.Context,
	refreshToken string,
) (string, error) {
	sub, err := s.tokenIssuer.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", response.NewHttpError(
			http.StatusUnauthorized,
			fmt.Errorf("Invalid token."),
		)
	}

	userId, err := uuid.Parse(fmt.Sprint(sub["user_id"]))
	if err != nil {
		return "", response.NewInternalServerError(err)
	}

	user, err := s.userRepo.GetUserById(
		ctx,
		userId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return "", response.NewNotFoundError(err)
		}
	}

	accessToken, _, _, err := s.tokenIssuer.CreateCredential(user.Id)
	if err != nil {
		return "", response.NewInternalServerError(err)
	}

	return accessToken, nil
}

func (s *UserService) ChangeUserName(
	ctx context.Context,
	userId uuid.UUID,
	firstName, lastName string,
) error {
	user, err := s.userRepo.GetUserById(
		ctx,
		userId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	user.FirstName = firstName
	user.LastName = lastName
	if err := s.userRepo.UpdateUserInfo(
		ctx,
		user.Id,
		user.FirstName,
		user.LastName,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *UserService) GetUserChallenge(
	ctx context.Context,
	userId uuid.UUID,
	walletAddress common.Address,
) (string, error) {
	wc, err := s.walletConnectionRepo.GetWalletConnectionByWalletAddress(
		ctx,
		walletAddress.Hex(),
	)
	if err != nil && !errors.Is(
		err,
		gorm.ErrRecordNotFound,
	) {
		return "", response.NewInternalServerError(err)
	}

	existUserWithWallet, err := s.userRepo.GetActiveUserByWalletConnection(
		ctx,
		walletAddress.Hex(),
	)
	if err != nil && !errors.Is(err,
		gorm.ErrRecordNotFound) {
		return "", response.NewInternalServerError(err)
	}
	if existUserWithWallet != nil && existUserWithWallet.Id != userId {
		return "", response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Wallet is already linked with another user."),
			"Wallet is already linked with another user.",
		)
	}
	if existUserWithWallet != nil && existUserWithWallet.Id == userId {
		return "", response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Wallet is already linked with this user."),
			"Wallet is already linked with this user.",
		)
	}

	challenge := wallet.GetChallenge()
	if wc == nil {
		wc = models.NewWalletConnection(
			walletAddress.Hex(),
			challenge,
		)
		if err := s.walletConnectionRepo.Create(
			ctx,
			wc,
		); err != nil {
			return "", response.NewInternalServerError(err)
		}
	} else {
		wc.Challenge = challenge
		if err := s.walletConnectionRepo.UpdateWalletConnection(
			ctx,
			wc,
		); err != nil {
			return "", response.NewInternalServerError(err)
		}
	}

	return wallet.FormatMessageForSigning(
		walletAddress.Hex(),
		challenge,
	), nil
}

func (s *UserService) GetDeleteingUsers(ctx context.Context) ([]*models.User, error) {
	users, err := s.userRepo.GetDeletingUsers(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *UserService) GetMe(
	ctx context.Context,
	authInfo models.AuthenticationInfo,
) (*models.User, error) {
	balance, err := s.paymentClient.GetWalletByUserId(ctx, authInfo.User.Id)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	authInfo.User.Balance = balance.Balance.Add(balance.FreeBalance)
	if authInfo.User.Status == models.BlockedStatus {
		authInfo.User.Debt, err = s.usageRepo.GetUserDebt(ctx, authInfo.User.Id)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}
	}

	return authInfo.User, nil
}

func (s *UserService) LinkWallet(
	ctx context.Context,
	user models.AuthenticationInfo,
	walletAddress common.Address,
	signature string,
) error {
	wc, err := s.walletConnectionRepo.GetWalletConnectionByWalletAddress(
		ctx,
		walletAddress.Hex(),
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewNotFoundError(err)
	}
	existUserWithWallet, err := s.userRepo.GetActiveUserByWalletConnection(
		ctx,
		walletAddress.Hex(),
	)
	if err != nil && !errors.Is(err,
		gorm.ErrRecordNotFound) {
		return response.NewInternalServerError(err)
	}
	if existUserWithWallet != nil && existUserWithWallet.Id != user.User.Id {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Wallet is already linked with another user."),
			"Wallet is already linked with another user.",
		)
	}
	if existUserWithWallet != nil && existUserWithWallet.Id == user.User.Id {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Wallet is already linked with this user."),
			"Wallet is already linked with this user.",
		)
	}

	if ok := wallet.VerifySig(
		walletAddress.Hex(),
		signature,
		"",
		wc.Challenge,
		wallet.SigningSig,
	); !ok {
		return response.InvalidSignatureError
	}

	if err := s.walletConnectionRepo.UpdateUserWalletConnection(
		ctx,
		user.User.Id,
		walletAddress.Hex(),
	); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *UserService) CreateSubscribeInfo(
	ctx context.Context,
	email string,
) {
	if err := s.userRepo.CreateSubscribeInfo(ctx, models.NewSubcribeInfo(email)); err != nil {
		if !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			slog.ErrorContext(ctx, "create subscribe info error", slog.Any("err", err))
		}
	}
}

func (s *UserService) UpdateUserStatus(
	ctx context.Context,
	userId uuid.UUID,
	status string,
) error {
	user, err := s.userRepo.GetUserById(
		ctx,
		userId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if user.Status == status {
		return nil
	}

	user.Status = status
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *UserService) DeleteUser(
	ctx context.Context,
	userId uuid.UUID,
) error {
	user, err := s.userRepo.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if user.Status == models.DeletedStatus || user.DeletedAt != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("User is already deleted."),
			"User is already deleted.",
		)
	}

	now := time.Now().UTC()
	user.DeletedAt = &now
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}
