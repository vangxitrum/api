package middlewares

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	custom_log "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/log"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/token"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/validate"
)

func DeserializeUser(
	tokenIssuer *token.TokenIssuer,
	userRepo models.UserRepository,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			userId, err := getUserIdFromToken(
				ctx,
				tokenIssuer,
			)
			if err != nil {
				return response.ResponseError(
					ctx,
					err,
				)
			}

			user, err := userRepo.GetUserById(
				ctx.Request().Context(),
				*userId,
			)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return response.ResponseError(
						ctx,
						response.NewHttpError(
							http.StatusUnauthorized,
							err,
							"User not found.",
						),
					)
				}

				return response.ResponseError(
					ctx,
					response.NewInternalServerError(err),
				)
			}

			if user.IsDeleted() {
				return response.ResponseError(
					ctx,
					response.NewHttpError(
						http.StatusBadRequest,
						fmt.Errorf("User is deleted."),
						"User is deleted.",
					),
				)
			}

			if err := userRepo.UpdateUserLastRequestedAt(ctx.Request().Context(), user.Id, time.Now().UTC()); err != nil {
				slog.Error(
					"Failed to update user last requested at",
					slog.Any("user_id", userId),
					slog.Any("error", err),
				)
			}

			ctx.Set(
				"authInfo",
				*models.NewAuthenticationInfo(user),
			)

			if attrs, ok := ctx.Request().Context().Value(custom_log.SlogFieldsKey).([]slog.Attr); ok {
				attrs = append(attrs, slog.Any("user_id", user.Id))
				ctx.SetRequest(ctx.Request().WithContext(context.WithValue(
					ctx.Request().Context(),
					custom_log.SlogFieldsKey,
					attrs,
				)))
			}

			return next(ctx)
		}
	}
}

func getUserIdFromToken(
	ctx echo.Context, t *token.TokenIssuer,
) (*uuid.UUID, error) {
	authorizationHeader := ctx.Request().Header.Get("Authorization")
	var accessToken string
	var err error
	fields := strings.Fields(authorizationHeader)
	if len(fields) == 2 && fields[0] == "Bearer" {
		accessToken = fields[1]
	}
	if accessToken == "" {
		return nil, response.UnauthorizedError
	}

	sub, err := t.ValidateAccessToken(accessToken)
	if err != nil {
		return nil, response.UnauthorizedError
	}

	userId, err := uuid.Parse(fmt.Sprint(sub["user_id"]))
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return &userId, nil
}

func AuthenticateAPIKey(
	tokenIssuer *token.TokenIssuer,
	us models.UseCase,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			var (
				token  string
				userId uuid.UUID
			)

			authorizationHeader := ctx.Request().Header.Get("Authorization")
			fields := strings.Fields(authorizationHeader)
			if len(fields) != 0 && fields[0] == "Bearer" {
				token = fields[1]
			}

			apiKey := ctx.Request().Header.Get(models.StreamPublicKeyHeader)
			apiSecret := ctx.Request().Header.Get(models.StreamSecretKeyHeader)
			if token != "" {
				sub, err := tokenIssuer.ValidateAccessToken(token)
				if err != nil {
					return response.NewUnauthorizedError(err)
				}

				userId, _ = uuid.Parse(
					fmt.Sprintf(
						"%v",
						sub["user_id"],
					),
				)

				if err := us.UserRepository().UpdateUserLastRequestedAt(ctx.Request().Context(), userId, time.Now().UTC()); err != nil {
					slog.Error(
						"Failed to update user last requested at",
						slog.Any("user_id", userId),
						slog.Any("error", err),
					)
				}
			} else {
				if len(fields) != 0 && fields[0] == "Basic" {
					decoded, err := base64.StdEncoding.DecodeString(fields[1])
					if err != nil {
						return response.ResponseFailMessage(
							ctx,
							http.StatusBadRequest,
							"Invalid basic auth format.",
						)
					}
					authParts := strings.SplitN(
						string(decoded),
						":",
						2,
					)
					if len(authParts) != 2 {
						return response.ResponseFailMessage(
							ctx,
							http.StatusBadRequest,
							"Invalid basic auth format.",
						)
					}
					apiKey = authParts[0]
					apiSecret = authParts[1]

					existed, err := us.ApiKeyRepository().GetApiKeyByKey(
						ctx.Request().Context(),
						apiKey,
					)
					if err != nil {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"API key not found.",
						)
					}

					if existed.IsInvalid() {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"API key is invalid.",
						)
					}

					numTtl, err := strconv.ParseInt(
						existed.Ttl,
						10,
						64,
					)
					if err != nil {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"API key not found.",
						)
					}
					err = bcrypt.CompareHashAndPassword(
						[]byte(existed.Secret),
						[]byte(apiSecret),
					)
					if err != nil {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"You are not logged in.",
						)
					}

					if time.Now().UTC().After(existed.ExpiredAt) && numTtl > 0 {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"The api key is expired.",
						)
					}

					if existed.Type == string(models.OnlyUpload) {
						path := ctx.Path()
						method := ctx.Request().Method

						isAllowed := validate.CheckAllowedPath(path, method)

						if !isAllowed {
							return response.ResponseFailMessage(
								ctx,
								http.StatusForbidden,
								"Api key are not allowed to access this resource.",
							)
						}
					}
					if err := us.ApiKeyRepository().UpdateApiKeyLastRequestedAt(
						ctx.Request().Context(), existed.Id,
						time.Now().UTC(),
					); err != nil {
						return response.NewInternalServerError(err)
					}
					userId = existed.UserId
				} else {
					if apiKey == "" || apiSecret == "" {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"You are not logged in.",
						)
					}

					existed, err := us.ApiKeyRepository().GetApiKeyByKey(
						ctx.Request().Context(),
						apiKey,
					)
					if err != nil {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"API key not found.",
						)
					}

					if existed.IsInvalid() {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"API key is invalid.",
						)
					}

					err = bcrypt.CompareHashAndPassword(
						[]byte(existed.Secret),
						[]byte(apiSecret),
					)
					if err != nil {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"You are not logged in.",
						)
					}

					if time.Now().UTC().After(existed.ExpiredAt) {
						return response.ResponseFailMessage(
							ctx,
							http.StatusUnauthorized,
							"The api key is expired.",
						)
					}

					if existed.Type == string(models.OnlyUpload) {
						path := ctx.Path()
						method := ctx.Request().Method

						isAllowed := validate.CheckAllowedPath(path, method)

						if !isAllowed {
							return response.ResponseFailMessage(
								ctx,
								http.StatusForbidden,
								"Api key are not allowed to access this resource.",
							)
						}
					}
					if err := us.ApiKeyRepository().UpdateApiKeyLastRequestedAt(
						ctx.Request().Context(), existed.Id,
						time.Now().UTC(),
					); err != nil {
						return response.NewInternalServerError(err)
					}
					userId = existed.UserId
				}
			}
			user, err := us.UserRepository().GetUserById(
				ctx.Request().Context(),
				userId,
			)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return response.ResponseFailMessage(
						ctx,
						http.StatusForbidden,
						"The user belonging to this token no longer exists.",
					)
				}

				return response.ResponseFailMessage(
					ctx,
					http.StatusInternalServerError,
					"Internal server error.",
				)
			}

			if user.IsDeleted() {
				return response.ResponseFailMessage(
					ctx,
					http.StatusForbidden,
					"The user belonging to this token no longer exists.",
				)
			}

			ctx.Set(
				"authInfo",
				*models.NewAuthenticationInfo(user),
			)

			if attrs, ok := ctx.Request().Context().Value(custom_log.SlogFieldsKey).([]slog.Attr); ok {
				attrs = append(attrs, slog.Any("user_id", user.Id))
				attrs = append(attrs, slog.Any("api_key", apiKey))
				ctx.SetRequest(ctx.Request().WithContext(context.WithValue(
					ctx.Request().Context(),
					custom_log.SlogFieldsKey,
					attrs,
				)))
			}

			return next(ctx)
		}
	}
}
