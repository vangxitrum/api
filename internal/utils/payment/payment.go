package payment

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	payment_gateway_model "github.com/AIOZNetwork/payment/models"
	payment_gateway "github.com/AIOZNetwork/payment/payment-gateway"
	"github.com/google/uuid"
	"github.com/mdobak/go-xerrors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/repositories"
)

var InsufficientBalanceError = fmt.Errorf("unsufficient balance")

type PaymentClient struct {
	gateway *payment_gateway.SubscribeService

	userRepo       models.UserRepository
	paymentLogRepo models.PaymentRepository

	quitCh chan struct{}
}

func NewPaymentClient(
	gateway *payment_gateway.SubscribeService,

	userRepo models.UserRepository,
	paymentLogRepo models.PaymentRepository,
) *PaymentClient {
	return &PaymentClient{
		gateway: gateway,

		userRepo:       userRepo,
		paymentLogRepo: paymentLogRepo,

		quitCh: make(chan struct{}),
	}
}

func (c *PaymentClient) NewPaymentClientWithTx(db *gorm.DB) *PaymentClient {
	return &PaymentClient{
		gateway: c.gateway,

		userRepo:       repositories.MustNewUserRepository(db, false),
		paymentLogRepo: repositories.MustNewPaymentRepository(db, false),

		quitCh: c.quitCh,
	}
}

func (c *PaymentClient) UpdateWalletFreeBalance(
	ctx context.Context,
	walletAddress string,
	freeBalance decimal.Decimal,
) error {
	if err := c.gateway.UpdateFreeBalanceByWalletAddress(ctx, walletAddress, freeBalance); err != nil {
		return err
	}

	return nil
}

func (c *PaymentClient) CreateTransaction(
	ctx context.Context,
	userId uuid.UUID,
	credit decimal.Decimal,
	transactionType string,
	belongsToId uuid.UUID,
) error {
	if credit.IsZero() {
		return nil
	}

	wallet, err := c.gateway.GetWalletByUserID(ctx, userId.String())
	if err != nil {
		return err
	}
	var pType uint8
	if transactionType == models.PaymentTypeBill {
		pType = models.BillingType
		if wallet.Balance.Add(wallet.FreeBalance).LessThan(credit) {
			return InsufficientBalanceError
		}
	} else if transactionType == models.PaymentTypeRefund {
		pType = models.DepositType
	}

	if !credit.IsZero() {
		if err := c.gateway.UpdateBalanceByWalletAddress(ctx, wallet.WalletAddress, credit, pType); err != nil {
			if strings.Contains(err.Error(), models.NotEnoughBalanceMessage) {
				return InsufficientBalanceError
			}

			return err
		}

		if err := c.paymentLogRepo.CreatePaymentLog(ctx, models.NewPaymentLog(credit, wallet.Balance, transactionType, userId, belongsToId)); err != nil {
			return err
		}
	}

	return nil
}

func (c *PaymentClient) StopWatching() {
	close(c.quitCh)
}

func (c *PaymentClient) CreateWallet(
	ctx context.Context,
	userId uuid.UUID,
	freeBalance decimal.Decimal,
) (string, error) {
	wallet, err := c.gateway.CreateWallet(ctx, userId, freeBalance)
	if err != nil {
		return "", err
	}

	return wallet, nil
}

func (c *PaymentClient) GetWalletByUserId(
	ctx context.Context,
	userId uuid.UUID,
) (*payment_gateway_model.Wallet, error) {
	return c.gateway.GetWalletByUserID(ctx, userId.String())
}

func (c *PaymentClient) GetAiozPrice(
	ctx context.Context,
) (float64, error) {
	return c.gateway.GetAIOZPrice()
}

func (c *PaymentClient) ConvertUsdToAioz(
	ctx context.Context,
	usd decimal.Decimal,
) (decimal.Decimal, float64, error) {
	return c.gateway.ConvertUSDToAIOZ(ctx, usd)
}

func (c *PaymentClient) StartWatchingPayment(ctx context.Context) {
	go func() {
		for {
			select {
			case <-c.quitCh:
				return
			default:
				func() {
					defer time.Sleep(5 * time.Second)
					if err := c.gateway.SubscribeDeposit(ctx); err != nil {
						if !strings.Contains(err.Error(), "context deadline exceeded") {
							slog.Error(
								"Subscribe deposit error",
								slog.Any("err", xerrors.New(err)),
							)
						}
					}

					if err := c.gateway.WithdrawAll(ctx, nil); err != nil {
						slog.Error("WithdrawAll error", slog.Any("err", xerrors.New(err)))
					}

					func() {
						unconfirmedTrans, err := c.gateway.GetUnconfirmed(ctx)
						if err != nil {
							slog.Error(
								"get unconfirmed transaction error",
								slog.Any("err", xerrors.New(err)),
							)
							return
						}

						for _, tran := range unconfirmedTrans {
							user, err := c.userRepo.GetUserByWalletAddress(
								ctx,
								tran.Sender,
							)
							if err != nil {
								slog.Error(
									"get unconfirmed transaction's user error",
									slog.Any("err", xerrors.New(err)),
								)
								return
							}

							if user.AiozPrice == 0 ||
								user.LastPriceUpdatedAt.Add(10*time.Minute).
									Before(time.Now().UTC()) {
								aiozPrice, err := c.GetAiozPrice(ctx)
								if err != nil {
									slog.Error(
										"get aioz price error",
										slog.Any("err", xerrors.New(err)),
									)
									return
								}

								user.AiozPrice = aiozPrice
								user.LastPriceUpdatedAt = time.Now().UTC()
								if err := c.userRepo.UpdateUserPriceInfo(
									ctx,
									user.Id,
									aiozPrice,
								); err != nil {
									slog.Error(
										"update user error",
										slog.Any("err", xerrors.New(err)),
									)
									return
								}
							}

							if _, err := c.gateway.ConfirmDepositHistoryCreditLater(ctx, tran, user.AiozPrice); err != nil {
								slog.Error(
									"confirm deposit history error",
									slog.Any("err", xerrors.New(err)),
								)
								return
							}

							txOut, err := c.paymentLogRepo.GetTxOutById(ctx, tran.Id)
							if err != nil {
								slog.Error(
									"get tx out by id error",
									slog.Any("err", xerrors.New(err)),
								)
								return
							}

							if !txOut.Credit.IsZero() && txOut.TypeTx == "normal" {
								wallet, err := c.gateway.GetWalletByUserID(ctx, user.Id.String())
								if err != nil {
									slog.Error(
										"get wallet by user id error",
										slog.Any("err", xerrors.New(err)),
									)
									return
								}

								if err := c.paymentLogRepo.CreatePaymentLog(
									ctx,
									models.NewPaymentLog(tran.Credit, wallet.Balance, models.PaymentTypeTopUp, user.Id, tran.Id),
								); err != nil {
									slog.Error(
										"create payment log error",
										slog.Any("err", xerrors.New(err)),
									)
									return
								}
							}

						}
					}()
				}()
			}
		}
	}()
}
