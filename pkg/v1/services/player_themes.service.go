package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/repositories"
)

type PlayerThemeService struct {
	db              *gorm.DB
	playerThemeRepo models.PlayerThemeRepository
	storageHelper   storage.StorageHelper
	mediaRepo       models.MediaRepository
	usageRepo       models.UsageRepository

	cdnFileRepo models.CdnFileRepository
}

func NewPlayerThemeService(
	db *gorm.DB,

	playerThemeRepo models.PlayerThemeRepository,
	cdnFileRepo models.CdnFileRepository,
	mediaRepo models.MediaRepository,
	usageRepo models.UsageRepository,

	storageHelper storage.StorageHelper,
) *PlayerThemeService {
	return &PlayerThemeService{
		db:              db,
		playerThemeRepo: playerThemeRepo,
		cdnFileRepo:     cdnFileRepo,
		mediaRepo:       mediaRepo,
		usageRepo:       usageRepo,

		storageHelper: storageHelper,
	}
}

func (s *PlayerThemeService) newPlayerThemeServiceWithTx(tx *gorm.DB) *PlayerThemeService {
	return &PlayerThemeService{
		db: tx,

		playerThemeRepo: repositories.MustNewPlayerThemeRepository(tx, false),
		cdnFileRepo:     repositories.MustNewCdnFileRepository(tx, false),
		mediaRepo:       repositories.MustNewMediaRepository(tx, false),
		usageRepo:       repositories.MustNewUsageRepository(tx, false),

		storageHelper: s.storageHelper,
	}
}

func (s *PlayerThemeService) CreatePlayerTheme(
	ctx context.Context, userId uuid.UUID,
	input models.PlayerThemeInput,
) (*models.PlayerTheme, error) {
	result, err := s.playerThemeRepo.CreatePlayerTheme(
		ctx, models.NewPlayerTheme(
			userId,
			input.Name,
			input.Theme,
		),
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return result, nil
}

func (s *PlayerThemeService) GetPlayerThemeById(
	ctx context.Context, userId, id uuid.UUID,
) (*models.PlayerTheme, error) {
	result, err := s.playerThemeRepo.GetUserPlayerThemeById(
		ctx,
		userId,
		id,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}
	if result.Asset.FileId != nil {
		baseUrl := fmt.Sprintf("%s/api/players/%s", models.BeUrl, result.Id)
		result.Asset.LogoImageLink = baseUrl + "/logo"
	}

	return result, nil
}

func (s *PlayerThemeService) DeleteUserPlayerThemeById(
	ctx context.Context, userId,
	themePlayerId uuid.UUID,
) error {
	existedPlayerTheme, err := s.playerThemeRepo.GetUserPlayerThemeById(
		ctx,
		userId,
		themePlayerId,
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

	existedActiveMediaWithTheme, err := s.playerThemeRepo.GetActiveMediaByPlayerThemeId(
		ctx, userId, themePlayerId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NewInternalServerError(err)
	}
	if existedActiveMediaWithTheme != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Theme is already used in media."),
		)
	}

	if existedPlayerTheme.Asset.FileId != nil {
		if err := s.storageHelper.Delete(
			ctx,
			&storage.Object{
				Id: *existedPlayerTheme.Asset.FileId,
			},
		); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	if err := s.playerThemeRepo.DeletePlayerThemeById(
		ctx,
		userId,
		themePlayerId,
	); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *PlayerThemeService) UpdatePlayerById(
	ctx context.Context,
	themeId, userId uuid.UUID,
	input models.PlayerThemeInput,
) (*models.PlayerTheme, error) {
	exist, err := s.playerThemeRepo.GetUserPlayerThemeById(
		ctx,
		userId,
		themeId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}
	if input.Name != "" {
		exist.Name = input.Name
	}

	if input.Theme.MainColor != "" {
		exist.Theme.MainColor = input.Theme.MainColor
	}

	if input.Theme.TextColor != "" {
		exist.Theme.TextColor = input.Theme.TextColor
	}

	if input.Theme.TextTrackColor != "" {
		exist.Theme.TextTrackColor = input.Theme.TextTrackColor
	}

	if input.Theme.TextTrackBackground != "" {
		exist.Theme.TextTrackBackground = input.Theme.TextTrackBackground
	}

	if input.Theme.ControlBarBackgroundColor != "" {
		exist.Theme.ControlBarBackgroundColor = input.Theme.ControlBarBackgroundColor
	}

	if input.Theme.MenuBackGroundColor != "" {
		exist.Theme.MenuBackGroundColor = input.Theme.MenuBackGroundColor
	}

	if input.Theme.MenuItemBackGroundHover != "" {
		exist.Theme.MenuItemBackGroundHover = input.Theme.MenuItemBackGroundHover
	}
	if input.Theme.ControlBarHeight != "" {
		exist.Theme.ControlBarHeight = input.Theme.ControlBarHeight
	}
	if input.Theme.ProgressBarHeight != "" {
		exist.Theme.ProgressBarHeight = input.Theme.ProgressBarHeight
	}
	if input.Theme.ProgressBarCircleSize != "" {
		exist.Theme.ProgressBarCircleSize = input.Theme.ProgressBarCircleSize
	}
	if input.IsDefault != nil {
		exist.IsDefault = input.IsDefault
		if *input.IsDefault {
			existDefault, err := s.playerThemeRepo.GetDefaultPlayerTheme(ctx, userId)
			if err != nil {
				return nil, response.NewInternalServerError(err)
			}
			if existDefault != nil && existDefault.Id != exist.Id {
				if err = s.playerThemeRepo.UpdateDefaultPlayerTheme(
					ctx,
					userId,
					existDefault.Id,
				); err != nil {
					return nil, response.NewInternalServerError(err)
				}
			}
		}
	}

	if input.Controls != nil {
		if input.Controls.EnableAPI != nil {
			exist.Controls.EnableAPI = input.Controls.EnableAPI
		}
		if input.Controls.EnableControls != nil {
			exist.Controls.EnableControls = input.Controls.EnableControls
		}
		if input.Controls.ForceAutoplay != nil {
			exist.Controls.ForceAutoplay = input.Controls.ForceAutoplay
		}
		if input.Controls.HideTitle != nil {
			exist.Controls.HideTitle = input.Controls.HideTitle
		}
		if input.Controls.ForceLoop != nil {
			exist.Controls.ForceLoop = input.Controls.ForceLoop
		}
	}

	result, err := s.playerThemeRepo.UpdatePlayerThemeById(
		ctx,
		exist.Id,
		exist,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return result, nil
}

func (s *PlayerThemeService) ListAllPlayersThemes(
	ctx context.Context,
	params models.GetThemePlayerList,
) ([]*models.PlayerTheme, int64, error) {
	result, total, err := s.playerThemeRepo.GetPlayerThemeList(
		ctx, models.GetThemePlayerList{
			UserId: params.UserId,
			SortBy: params.SortBy,
			Order:  params.Order,
			Search: params.Search,
			Offset: params.Offset,
			Limit:  params.Limit,
		},
	)
	if err != nil {
		return nil, 0, response.NewHttpError(
			http.StatusInternalServerError, err,
			"Failed to get player themes list.",
		)
	}
	for _, theme := range result {
		if theme.Asset.FileId != nil {
			baseUrl := fmt.Sprintf(models.PlayerLogoUlrFormat, models.BeUrl, theme.Id)
			theme.Asset.LogoImageLink = baseUrl + "/logo"
		}
	}

	return result, total, nil
}

func (s *PlayerThemeService) UploadPlayerThemeLogo(
	ctx context.Context,
	themeId uuid.UUID,
	userId uuid.UUID,
	link string,
	reader multipart.File,
) (*models.PlayerTheme, error) {
	defer func(reader multipart.File) {
		err := reader.Close()
		if err != nil {
			return
		}
	}(reader)
	result, err := s.playerThemeRepo.GetUserPlayerThemeById(
		ctx,
		userId,
		themeId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	var oldFileId *string
	if result.Asset.FileId != nil {
		if err := s.storageHelper.Delete(
			ctx,
			&storage.Object{
				Id: *result.Asset.FileId,
			},
		); err != nil {
			return nil, response.NewInternalServerError(err)
		}

		oldFileId = result.Asset.FileId
	}

	resp, err := s.storageHelper.Upload(
		ctx,
		result.Id.String(),
		reader,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if err := s.cdnFileRepo.Create(
		ctx, models.NewCdnFile(
			userId,
			resp.Id,
			resp.Size,
			resp.Offset,
			1,
			models.CdnLogoType,
		),
	); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	newLogoData := models.Asset{
		FileId:   &resp.Id,
		LogoLink: link,
	}
	playerInfo, err := s.playerThemeRepo.UpdatePlayerThemeAsset(
		ctx,
		themeId,
		newLogoData,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if err := s.usageRepo.CreateLog(
		ctx,
		(&models.UsageLogBuilder{}).
			SetStorage(resp.Size).
			SetUserId(userId).
			Build(),
	); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if oldFileId != nil {
		if err := s.cdnFileRepo.DeleteCdnFileByFileId(
			ctx,
			*oldFileId,
		); err != nil {
			return nil, response.NewInternalServerError(err)
		}
	}

	return playerInfo, nil
}

func (s *PlayerThemeService) DeletePlayerThemeLogo(
	ctx context.Context,
	userId, themePlayerId uuid.UUID,
) error {
	result, err := s.playerThemeRepo.GetUserPlayerThemeById(
		ctx,
		userId,
		themePlayerId,
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

	if result.Asset.FileId == nil {
		return response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Logo does not exist."),
		)
	}

	if err := s.storageHelper.Delete(ctx, &storage.Object{
		Id: *result.Asset.FileId,
	}); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.playerThemeRepo.DeletePlayerThemeAsset(
		ctx,
		userId,
		themePlayerId,
	); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if err := s.cdnFileRepo.DeleteCdnFileByFileId(
		ctx,
		*result.Asset.FileId,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *PlayerThemeService) AddPlayerThemeToMediaById(
	ctx context.Context, userId,
	mediaId, themePlayerId uuid.UUID,
) error {
	existed, err := s.mediaRepo.GetUserMediaById(
		ctx,
		userId,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewHttpError(
				http.StatusNotFound,
				fmt.Errorf("Media not found."),
			)
		}
		return response.NewInternalServerError(err)
	}

	_, err = s.playerThemeRepo.GetUserPlayerThemeById(
		ctx,
		userId,
		themePlayerId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewHttpError(
				http.StatusNotFound,
				fmt.Errorf("Player theme not found."),
			)
		}
		return response.NewInternalServerError(err)
	}

	if err := s.playerThemeRepo.AddPlayerThemeToMediaById(
		ctx,
		themePlayerId,
		existed.Id,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *PlayerThemeService) RemovePlayerThemeFromMediaById(
	ctx context.Context,
	userId,
	mediaId,
	themePlayerId uuid.UUID,
) error {
	existed, err := s.mediaRepo.GetUserMediaById(
		ctx,
		userId,
		mediaId,
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

	if existed.PlayerThemeId == nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Media does not have a player theme."),
		)
	}

	if err := s.playerThemeRepo.RemovePlayerThemeFromMedia(
		ctx,
		themePlayerId,
		existed.Id,
		userId,
	); err != nil {
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *PlayerThemeService) GetPlayerThemeLogo(
	ctx context.Context,
	themePlayerId uuid.UUID,
) (*models.FileInfo, error) {
	playerTheme, err := s.playerThemeRepo.GetPlayerThemeById(
		ctx,
		themePlayerId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if playerTheme.Asset.File == nil {
		return nil, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Logo does not exist."),
		)
	}

	redirectUrl, expiredAt, err := s.storageHelper.GetLink(
		ctx,
		&storage.Object{
			Id:     playerTheme.Asset.File.Id,
			Size:   playerTheme.Asset.File.Size,
			Offset: playerTheme.Asset.File.Offset,
		},
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	var reader io.Reader
	if redirectUrl == "" {
		reader, err = s.storageHelper.Download(
			ctx,
			&storage.Object{
				Id:     playerTheme.Asset.File.Id,
				Size:   playerTheme.Asset.File.Size,
				Offset: playerTheme.Asset.File.Offset,
			},
		)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}
	}

	return (&models.FileInfoBuilder{}).
		SetMediaId(playerTheme.Id).
		SetUserId(playerTheme.UserId).
		SetRedirectUrl(redirectUrl).
		SetReader(reader).
		SetExpiredAt(expiredAt).
		SetSize(playerTheme.Asset.File.Size).
		Build(ctx), nil
}

func (s *PlayerThemeService) DeleteUserPlayerThemes(
	ctx context.Context,
	userId uuid.UUID,
) error {
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		service := s.newPlayerThemeServiceWithTx(tx)

		var offset uint64
		for {
			playerThemes, _, err := service.playerThemeRepo.GetPlayerThemeList(
				ctx,
				models.GetThemePlayerList{
					UserId: userId,
					Offset: offset,
				},
			)
			if err != nil {
				return err
			}

			if len(playerThemes) == 0 {
				break
			}

			for _, playerTheme := range playerThemes {
				if err := service.DeleteUserPlayerThemeById(
					ctx,
					userId,
					playerTheme.Id,
				); err != nil {
					return err
				}
			}

			offset += uint64(len(playerThemes))
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
