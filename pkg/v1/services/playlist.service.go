package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/image"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/slice"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/sorting"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/repositories"
)

type PlaylistService struct {
	db                 *gorm.DB
	playlistRepository models.PlaylistRepository
	cdnFileRepo        models.CdnFileRepository
	usageRepo          models.UsageRepository
	thumbnailRepo      models.ThumbnailRepository
	mediaRepo          models.MediaRepository
	storageHelper      storage.StorageHelper
	playerThemeRepo    models.PlayerThemeRepository

	thumbnailHelper *image.ThumbnailHelper
}

func NewPlaylistService(
	db *gorm.DB,
	playlistRepository models.PlaylistRepository,
	cdnFileRepo models.CdnFileRepository,
	usageRepo models.UsageRepository,
	storageHelper storage.StorageHelper,
	thumbnailRepo models.ThumbnailRepository,
	mediaRepo models.MediaRepository,
	playerThemeRepo models.PlayerThemeRepository,

	thumbnailHelper *image.ThumbnailHelper,
) *PlaylistService {
	return &PlaylistService{
		db:                 db,
		playlistRepository: playlistRepository,
		cdnFileRepo:        cdnFileRepo,
		usageRepo:          usageRepo,
		storageHelper:      storageHelper,
		thumbnailRepo:      thumbnailRepo,
		mediaRepo:          mediaRepo,
		playerThemeRepo:    playerThemeRepo,

		thumbnailHelper: thumbnailHelper,
	}
}

func (s *PlaylistService) newPlaylistServiceWithTx(tx *gorm.DB) *PlaylistService {
	return &PlaylistService{
		db:                 tx,
		playlistRepository: repositories.MustNewPlaylistRepository(tx, false),
		cdnFileRepo:        repositories.MustNewCdnFileRepository(tx, false),
		usageRepo:          repositories.MustNewUsageRepository(tx, false),
		thumbnailRepo:      repositories.MustNewThumbnailRepository(tx, false),
		mediaRepo:          repositories.MustNewMediaRepository(tx, false),
		playerThemeRepo:    repositories.MustNewPlayerThemeRepository(tx, false),

		storageHelper: s.storageHelper,

		thumbnailHelper: s.thumbnailHelper,
	}
}

func (s *PlaylistService) CreatePlaylist(
	ctx context.Context,
	userId uuid.UUID,
	name string,
	metadata []models.Metadata,
	tags []string,
	playlistStyle string,
) (*models.Playlist, error) {
	var thumbnailId *uuid.UUID

	newPlaylist, err := models.NewPlaylist(userId, name, metadata, tags, thumbnailId, playlistStyle)
	if err != nil {
		return nil, err
	}

	result, err := s.playlistRepository.CreatePlaylist(ctx, newPlaylist)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return result, nil
}

func (s *PlaylistService) UpdatePlaylistInfo(
	ctx context.Context,
	userId uuid.UUID,
	input models.UpdatePlaylistInput,
) error {
	playlist, err := s.playlistRepository.GetPlaylistById(ctx, input.PlaylistId)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if playlist.UserId != userId {
		return response.NewHttpError(
			http.StatusForbidden,
			errors.New("You are not allowed to update this playlist."),
		)
	}

	if input.Thumbnail != nil {
		if playlist.Thumbnail != nil {
			if err := s.storageHelper.Delete(
				ctx,
				&storage.Object{
					Id: playlist.Thumbnail.Thumbnail.File.FileId,
				},
			); err != nil {
				return response.NewInternalServerError(err)
			}

			if err := s.playlistRepository.DeletePlaylistThumbnail(ctx, playlist.Id); err != nil {
				return response.NewInternalServerError(err)
			}
		}

		src, err := input.Thumbnail.Open()
		if err != nil {
			return response.NewInternalServerError(err)
		}

		defer src.Close()
		newThumbnail, totalSize, err := s.thumbnailHelper.GenerateThumbnail(
			ctx,
			playlist.UserId,
			src,
		)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		if err := s.playlistRepository.CreatePlaylistThumbnail(ctx, &models.PlaylistThumbnail{
			PlaylistId:  playlist.Id,
			ThumbnailId: newThumbnail.Id,
			Thumbnail:   newThumbnail,
		}); err != nil {
			return response.NewInternalServerError(err)
		}

		if err := s.usageRepo.CreateLog(
			ctx,
			(&models.UsageLogBuilder{}).
				SetUserId(playlist.UserId).
				SetStorage(totalSize).
				SetIsUserCost(false).
				Build(),
		); err != nil {
			return err
		}
	}

	if input.Tags != nil {
		var tags string
		if len(input.Tags) > 0 {
			tags = strings.Join(input.Tags, ",")
		}

		playlist.Tags = tags
	}

	if input.Metadata != nil {
		playlist.Metadata = make(map[string]any)
		for _, data := range input.Metadata {
			playlist.Metadata[data.Key] = data.Value
		}
	}

	if input.Name != nil {
		playlist.Name = *input.Name
	}

	if err := s.playlistRepository.UpdatePlaylistById(ctx, userId, input.PlaylistId, playlist); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *PlaylistService) GetPlaylistPublicById(
	ctx context.Context,
	playlistId uuid.UUID,
) (*models.Playlist, *models.PlayerTheme, error) {
	playlist, err := s.playlistRepository.GetPlaylistById(ctx, playlistId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, response.NewNotFoundError(err)
		}
		return nil, nil, response.NewInternalServerError(err)
	}

	userDefaultTheme, err := s.playerThemeRepo.GetDefaultPlayerTheme(ctx, playlist.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, response.NewInternalServerError(err)
	}

	var processedItems []*models.PlaylistItem
	validItems := make([]*models.PlaylistItem, 0, len(processedItems))
	for _, item := range s.sortPlaylistItems(playlist.MediaItems) {
		if item.Media.Status == models.DeletedStatus {
			if err := s.RemoveMediaFromPlaylist(ctx, playlist.UserId, playlistId, item.Id, []uuid.UUID{}); err != nil {
				return nil, nil, err
			}

			continue
		}

		duration := item.Media.GetMediaDuration()
		item.MediaItem = &models.PlaylistItemMedia{
			Title:    item.Media.Title,
			Duration: duration,
		}
		hlsUrl := item.Media.GetHlsManifestUrl()
		thumbnailUrl := item.Media.GetThumbnailUrl()
		item.MediaItem.ThumbnailUrl = thumbnailUrl
		item.MediaItem.HlsUrl = hlsUrl
		for _, chapter := range item.Media.Chapters {
			chapter.Url = chapter.GetUrl(item.Media.Secret)
			item.MediaItem.Chapters = append(item.MediaItem.Chapters, chapter)
		}

		for _, caption := range item.Media.Captions {
			caption.Url = caption.GetUrl(models.CdnCaptionType, item.Media.Secret)
			item.MediaItem.Captions = append(item.MediaItem.Captions, caption)
		}

		validItems = append(validItems, item)
	}

	iFrame := playlist.GetIframe()
	playlist.MediaItems = validItems
	playlist.ItemCount = len(validItems)
	playlist.IFrame = iFrame
	playlist.ThumbnailUrl = playlist.GetThumbnailUrl()
	return playlist, userDefaultTheme, nil
}

func (s *PlaylistService) GetPlaylistById(
	ctx context.Context,
	userId, playlistId uuid.UUID,
	filterPayload *models.PlaylistItemFilter,
) (*models.Playlist, error) {
	playlist, err := s.playlistRepository.GetPlaylistById(ctx, playlistId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if playlist.UserId != userId {
		return nil, response.NewHttpError(
			http.StatusForbidden,
			errors.New("You are not allowed to access this playlist."),
		)
	}

	var items []*models.PlaylistItem
	var processedItems []*models.PlaylistItem
	if filterPayload != nil &&
		(filterPayload.SortBy != "" || filterPayload.OrderBy != "" || filterPayload.Search != "") {
		items, err = s.playlistRepository.GetPlaylistItemMediaByIdWithFilter(
			ctx,
			playlistId,
			filterPayload,
		)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewInternalServerError(err)
		}
		processedItems = items
	} else {
		items, err = s.playlistRepository.GetPlaylistItemMediaById(ctx, playlistId)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewInternalServerError(err)
		}
		processedItems = s.sortPlaylistItems(items)
	}

	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	var totalDuration float64
	var totalSize int64
	validItems := make([]*models.PlaylistItem, 0, len(processedItems))
	for _, item := range processedItems {
		if item.Media.Status == models.DeletedStatus {
			if err := s.RemoveMediaFromPlaylist(ctx, userId, playlistId, item.Id, []uuid.UUID{}); err != nil {
				return nil, err
			}
			continue
		}

		duration := item.Media.GetMediaDuration()
		totalDuration += duration
		totalSize += item.Media.Size

		item.MediaItem = &models.PlaylistItemMedia{
			Title:     item.Media.Title,
			Qualities: sorting.SortQualities(item.Media.MediaQualities),
			Duration:  duration,
			Size:      item.Media.Size,
		}
		thumbnailUrl := item.Media.GetThumbnailUrl()
		item.MediaItem.ThumbnailUrl = thumbnailUrl

		validItems = append(validItems, item)
	}

	iFrame := playlist.GetIframe()
	playlistUrl := playlist.GetPlaylistUrl()

	playlist.MediaItems = validItems
	playlist.ItemCount = len(validItems)
	playlist.Duration = totalDuration
	playlist.Size = totalSize
	playlist.IFrame = iFrame
	playlist.PlaylistUrl = playlistUrl
	playlist.ThumbnailUrl = playlist.GetThumbnailUrl()

	return playlist, nil
}

func (s *PlaylistService) sortPlaylistItems(
	mediaItems []*models.PlaylistItem,
) []*models.PlaylistItem {
	if len(mediaItems) == 0 {
		return []*models.PlaylistItem{}
	}

	itemMap := make(map[string]*models.PlaylistItem)
	for _, item := range mediaItems {
		itemMap[item.Id.String()] = item
	}

	var firstItem *models.PlaylistItem
	for _, item := range mediaItems {
		if item.PreviousId == nil {
			if firstItem != nil {
				return []*models.PlaylistItem{}
			}
			firstItem = item
		}
	}

	if firstItem == nil {
		return []*models.PlaylistItem{}
	}

	sortedItems := []*models.PlaylistItem{firstItem}
	currentID := firstItem.NextId
	seenIds := make(map[string]bool)
	seenIds[firstItem.Id.String()] = true

	for currentID != nil {
		currentItem, exists := itemMap[currentID.String()]
		if !exists {
			return []*models.PlaylistItem{}
		}
		if seenIds[currentID.String()] {
			return []*models.PlaylistItem{}
		}
		sortedItems = append(sortedItems, currentItem)
		seenIds[currentID.String()] = true
		currentID = currentItem.NextId
	}

	if len(sortedItems) != len(mediaItems) {
		return []*models.PlaylistItem{}
	}

	return sortedItems
}

func (s *PlaylistService) GetUserPlaylists(
	ctx context.Context,
	userId uuid.UUID,
	filter models.PlaylistFilter,
) ([]*models.Playlist, int64, error) {
	playlists, total, err := s.playlistRepository.GetUserPlaylists(ctx, userId, filter)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}

	for _, playlist := range playlists {
		items, err := s.playlistRepository.GetPlaylistItemMediaById(ctx, playlist.Id)
		if err != nil {
			return nil, 0, response.NewInternalServerError(err)
		}

		var totalDuration float64

		iFrame := playlist.GetIframe()
		playlistUrl := playlist.GetPlaylistUrl()
		validCount := 0
		for _, item := range items {
			if item.Media != nil && item.Media.Status != models.DeletedStatus {
				duration := item.Media.GetMediaDuration()
				totalDuration += duration
				validCount++
			}
		}

		playlist.Duration = totalDuration
		playlist.ItemCount = validCount
		playlist.IFrame = iFrame
		playlist.PlaylistUrl = playlistUrl
		playlist.ThumbnailUrl = playlist.GetThumbnailUrl()

	}

	return playlists, total, nil
}

func (s *PlaylistService) DeletePlaylistById(
	ctx context.Context,
	userId, playlistId uuid.UUID,
) error {
	playlist, err := s.playlistRepository.GetPlaylistById(ctx, playlistId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if playlist.UserId != userId {
		return response.NewHttpError(
			http.StatusForbidden,
			errors.New("You are not allowed to delete this playlist."),
		)
	}

	if err := s.playlistRepository.DeletePlaylistItemByPlaylistId(ctx, playlist.Id); err != nil {
		return response.NewInternalServerError(err)
	}

	if playlist.Thumbnail != nil {
		if err := s.playlistRepository.DeletePlaylistThumbnail(ctx, playlistId); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	if err := s.playlistRepository.DeletePlaylistById(ctx, userId, playlist.Id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *PlaylistService) AddMediaToPlaylist(
	ctx context.Context,
	userId uuid.UUID,
	uniquePlaylistIds []uuid.UUID,
	uniqueMediaIds []uuid.UUID,
) error {
	medias, err := s.mediaRepo.GetManyMediasByIds(ctx, uniqueMediaIds, userId)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	foundIds := make(map[uuid.UUID]struct{}, len(medias))
	notReady := make([]uuid.UUID, 0)
	for _, media := range medias {
		foundIds[media.Id] = struct{}{}

		if !media.IsDone() {
			notReady = append(notReady, media.Id)
		}
	}

	missingIds := make([]uuid.UUID, 0)
	for _, id := range uniqueMediaIds {
		if _, ok := foundIds[id]; !ok {
			missingIds = append(missingIds, id)
		}
	}

	if len(missingIds) > 0 {
		return response.NewNotFoundError(
			fmt.Errorf("Media not found: %v.", missingIds),
		)
	}

	if len(notReady) > 0 {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Media(s) not ready: %v.", notReady),
		)
	}

	playlists, err := s.playlistRepository.GetPlaylistByIds(ctx, uniquePlaylistIds)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if len(playlists) != len(uniquePlaylistIds) {
		return response.NewNotFoundError(errors.New("One or more playlists not found."))
	}

	for i := range playlists {
		if playlists[i].UserId != userId {
			return response.NewHttpError(http.StatusForbidden,
				fmt.Errorf("You are not allowed to update playlist: %s.", playlists[i].Id))
		}
	}

	for _, playlist := range playlists {
		for _, media := range medias {
			if playlist.PlaylistType != media.Type {
				return response.NewHttpError(
					http.StatusBadRequest,
					fmt.Errorf(
						"Invalid media type for this playlist. Media has type %s but playlist expects type %s.",
						media.Type,
						playlist.PlaylistType,
					),
				)
			}
		}
	}

	itemCounts, err := s.playlistRepository.GetPlaylistItemCounts(ctx, uniquePlaylistIds)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	for playlistId, count := range itemCounts {
		if count+int64(len(uniqueMediaIds)) > models.MaxPlaylistItemCount {
			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("Playlist %s has reached the maximum number of items.", playlistId),
			)
		}
	}

	exists, err := s.playlistRepository.GetExistsAnyMediaInPlaylists(
		ctx,
		uniquePlaylistIds,
		uniqueMediaIds,
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if exists {
		return response.NewHttpError(
			http.StatusBadRequest,
			errors.New("Media already in playlist."),
			"One or more selected media are already in the selected playlists.",
		)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		playlistService := s.newPlaylistServiceWithTx(tx)

		for _, playlistId := range uniquePlaylistIds {
			lastItem, err := playlistService.playlistRepository.GetLastPlaylistItemByPlaylistId(
				ctx,
				playlistId,
			)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return response.NewInternalServerError(err)
			}

			prevItem := lastItem

			for _, mediaId := range uniqueMediaIds {
				var previousId *uuid.UUID
				if prevItem != nil {
					previousId = &prevItem.Id
				}

				newItem, err := models.NewPlaylistItem(playlistId, mediaId, nil, previousId)
				if err != nil {
					return err
				}

				createdItem, err := playlistService.playlistRepository.CreatePlaylistItem(
					ctx,
					newItem,
				)
				if err != nil {
					return err
				}

				if prevItem != nil {
					prevItem.NextId = &createdItem.Id
					if err := playlistService.playlistRepository.UpdatePlaylistItem(ctx, prevItem); err != nil {
						return err
					}
				}

				prevItem = createdItem
			}
		}
		return nil
	})
}

func (s *PlaylistService) RemoveMediaFromPlaylist(
	ctx context.Context,
	userId, playlistId, itemId uuid.UUID,
	optionsPlaylistId []uuid.UUID,
) error {
	playlist, err := s.playlistRepository.GetPlaylistById(ctx, playlistId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if playlist.UserId != userId {
		return response.NewHttpError(
			http.StatusForbidden,
			errors.New("You are not allowed to update this playlist."),
		)
	}

	item, err := s.playlistRepository.GetPlaylistItemById(ctx, itemId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if len(optionsPlaylistId) > 0 {
		uniquePlaylistIds := slice.DeDupUUIDs(optionsPlaylistId)

		items, err := s.playlistRepository.CheckMediaExistsInPlaylists(
			ctx,
			uniquePlaylistIds,
			item.MediaId,
		)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		if err = s.removeMediaFromManyPlaylists(
			ctx,
			items,
		); err != nil {
			return response.NewInternalServerError(err)
		}
		return nil
	}

	if item.PreviousId == nil {
		if item.NextId != nil {
			nextItem, err := s.playlistRepository.GetPlaylistItemById(ctx, *item.NextId)
			if err == nil {
				nextItem.PreviousId = nil
				if err := s.playlistRepository.UpdatePlaylistItemPosition(ctx, nextItem); err != nil {
					return response.NewInternalServerError(err)
				}
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return response.NewInternalServerError(err)
			}
		}
	} else {
		prevItem, err := s.playlistRepository.GetPlaylistItemById(ctx, *item.PreviousId)
		if err == nil {
			prevItem.NextId = item.NextId
			if err := s.playlistRepository.UpdatePlaylistItemPosition(ctx, prevItem); err != nil {
				return response.NewInternalServerError(err)
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewInternalServerError(err)
		}
	}

	if item.NextId != nil {
		nextItem, err := s.playlistRepository.GetPlaylistItemById(ctx, *item.NextId)
		if err == nil {
			nextItem.PreviousId = item.PreviousId
			if err := s.playlistRepository.UpdatePlaylistItemPosition(ctx, nextItem); err != nil {
				return response.NewInternalServerError(err)
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewInternalServerError(err)
		}
	}

	if err := s.playlistRepository.DeletePlaylistItemById(ctx, playlistId, itemId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *PlaylistService) removeMediaFromManyPlaylists(
	ctx context.Context,
	items []*models.PlaylistItem,
) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		playlistService := s.newPlaylistServiceWithTx(tx)
		for _, item := range items {
			if item.PreviousId == nil {
				if item.NextId != nil {
					nextItem, err := playlistService.playlistRepository.GetPlaylistItemById(
						ctx,
						*item.NextId,
					)
					if err == nil {
						nextItem.PreviousId = nil
						if err := playlistService.playlistRepository.UpdatePlaylistItemPosition(ctx, nextItem); err != nil {
							return response.NewInternalServerError(err)
						}
					} else if !errors.Is(err, gorm.ErrRecordNotFound) {
						return response.NewInternalServerError(err)
					}
				}
			} else {
				prevItem, err := playlistService.playlistRepository.GetPlaylistItemById(ctx, *item.PreviousId)
				if err == nil {
					prevItem.NextId = item.NextId
					if err := playlistService.playlistRepository.UpdatePlaylistItemPosition(ctx, prevItem); err != nil {
						return response.NewInternalServerError(err)
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return response.NewInternalServerError(err)
				}
			}

			if item.NextId != nil {
				nextItem, err := playlistService.playlistRepository.GetPlaylistItemById(
					ctx,
					*item.NextId,
				)
				if err == nil {
					nextItem.PreviousId = item.PreviousId
					if err := playlistService.playlistRepository.UpdatePlaylistItemPosition(ctx, nextItem); err != nil {
						return response.NewInternalServerError(err)
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return response.NewInternalServerError(err)
				}
			}

			if err := playlistService.playlistRepository.DeletePlaylistItemById(
				ctx,
				item.PlaylistId,
				item.Id,
			); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return response.NewNotFoundError(err)
				}
				return response.NewInternalServerError(err)
			}
		}
		return nil
	})
}

func (s *PlaylistService) MoveItemInPlaylist(
	ctx context.Context,
	userId uuid.UUID,
	playlistId uuid.UUID,
	currentId uuid.UUID,
	nextId *uuid.UUID,
	previousId *uuid.UUID,
) error {
	playlist, err := s.playlistRepository.GetPlaylistById(ctx, playlistId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if playlist.UserId != userId {
		return response.NewHttpError(
			http.StatusForbidden,
			errors.New("You are not allowed to update this playlist."),
		)
	}

	allItems, err := s.playlistRepository.GetPlaylistItemsByPlaylistId(ctx, playlistId)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	itemMap := make(map[string]*models.PlaylistItem)
	for _, item := range allItems {
		itemMap[item.Id.String()] = item
	}

	currentItem, exists := itemMap[currentId.String()]
	if !exists {
		return response.NewNotFoundError(errors.New("Current item not found."))
	}

	var updates []*models.PlaylistItem

	if currentItem.PreviousId != nil {
		prevItem := itemMap[currentItem.PreviousId.String()]
		prevItem.NextId = currentItem.NextId
		updates = append(updates, prevItem)
	}
	if currentItem.NextId != nil {
		nextItem := itemMap[currentItem.NextId.String()]
		nextItem.PreviousId = currentItem.PreviousId
		updates = append(updates, nextItem)
	}

	if nextId != nil && previousId == nil {
		nextItem := itemMap[nextId.String()]
		if nextItem.PreviousId != nil {
			return response.NewHttpError(
				http.StatusBadRequest,
				errors.New("Next item must be the first item when moving to start."),
			)
		}
		currentItem.NextId = nextId
		currentItem.PreviousId = nil
		nextItem.PreviousId = &currentId
		updates = append(updates, nextItem)
	} else if previousId != nil && nextId == nil {
		prevItem := itemMap[previousId.String()]
		if prevItem.NextId != nil {
			return response.NewHttpError(
				http.StatusBadRequest,
				errors.New("Previous item must be the last item when moving to end."),
			)
		}
		currentItem.PreviousId = previousId
		currentItem.NextId = nil
		prevItem.NextId = &currentId
		updates = append(updates, prevItem)
	} else if previousId != nil && nextId != nil {
		prevItem := itemMap[previousId.String()]
		nextItem := itemMap[nextId.String()]

		if prevItem.NextId == nil || *prevItem.NextId != *nextId {
			return response.NewHttpError(
				http.StatusBadRequest,
				errors.New("Previous and next items must be adjacent."),
			)
		}

		currentItem.PreviousId = previousId
		currentItem.NextId = nextId
		prevItem.NextId = &currentId
		nextItem.PreviousId = &currentId

		updates = append(updates, prevItem, nextItem)
	} else {
		return response.NewHttpError(
			http.StatusBadRequest,
			errors.New("Invalid move operation: must specify either next_id for start, previous_id for end, or both for middle position."),
		)
	}

	updates = append(updates, currentItem)

	updatedItems := make([]*models.PlaylistItem, len(allItems))
	copy(updatedItems, allItems)
	for _, update := range updates {
		for i, item := range updatedItems {
			if item.Id == update.Id {
				updatedItems[i] = update
				break
			}
		}
	}

	if !s.isValidLinkedList(updatedItems) {
		return response.NewHttpError(
			http.StatusBadRequest,
			errors.New("Invalid playlist order after move operation."),
		)
	}

	if err := s.playlistRepository.UpdatePlaylistItemsPosition(ctx, updates); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *PlaylistService) GetPlaylistThumbnail(
	ctx context.Context,
	playlistId uuid.UUID,
	resolution string,
) (*models.FileInfo, error) {
	playlist, err := s.playlistRepository.GetPlaylistById(
		ctx,
		playlistId,
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

	if playlist.Thumbnail == nil {
		return nil, response.NewNotFoundError(errors.New("Thumbnail does not exist."))
	}

	var thumbnailResolution *models.ThumbnailResolution
	for _, res := range playlist.Thumbnail.Thumbnail.Resolutions {
		if res.Resolution == resolution {
			thumbnailResolution = res
			break
		}
	}

	if thumbnailResolution == nil {
		return nil, response.NewNotFoundError(errors.New("Thumbnail resolution does not exist."))
	}

	redirectUrl, expiredAt, err := s.storageHelper.GetLink(
		ctx,
		&storage.Object{
			Id:     playlist.Thumbnail.Thumbnail.File.FileId,
			Size:   thumbnailResolution.Size,
			Offset: thumbnailResolution.Offset,
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
				Id:     playlist.Thumbnail.Thumbnail.File.FileId,
				Size:   thumbnailResolution.Size,
				Offset: thumbnailResolution.Offset,
			},
		)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}
	}

	return (&models.FileInfoBuilder{}).
		SetMediaId(playlistId).
		SetUserId(playlist.UserId).
		SetRedirectUrl(redirectUrl).
		SetReader(reader).
		SetExpiredAt(expiredAt).
		SetSize(thumbnailResolution.Size).
		Build(ctx), nil
}

func (s *PlaylistService) isValidLinkedList(items []*models.PlaylistItem) bool {
	if len(items) == 0 {
		return true
	}

	itemMap := make(map[string]*models.PlaylistItem)
	for _, item := range items {
		itemMap[item.Id.String()] = item
	}

	var firstItem *models.PlaylistItem
	for _, item := range items {
		if item.PreviousId == nil {
			if firstItem != nil {
				return false
			}
			firstItem = item
		}
	}

	if firstItem == nil {
		return false
	}

	count := 1
	currentID := firstItem.NextId
	seenIds := make(map[string]bool)
	seenIds[firstItem.Id.String()] = true

	for currentID != nil {
		currentItem, exists := itemMap[currentID.String()]
		if !exists {
			return false
		}
		if seenIds[currentID.String()] {
			return false
		}
		seenIds[currentID.String()] = true
		currentID = currentItem.NextId
		count++
	}

	return count == len(items)
}

func (s *PlaylistService) DeletePlaylistThumbnail(
	ctx context.Context,
	userId, playlistId uuid.UUID,
) error {
	playlist, err := s.playlistRepository.GetPlaylistById(
		ctx,
		playlistId,
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

	if playlist.UserId != userId {
		return response.NewHttpError(
			http.StatusForbidden,
			fmt.Errorf("You are not allowed to update this playlist."),
		)
	}

	if playlist.Thumbnail == nil {
		return response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Thumbnail does not exist."),
		)
	}

	if err := s.storageHelper.Delete(
		ctx,
		&storage.Object{
			Id: playlist.Thumbnail.Thumbnail.File.FileId,
		},
	); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.playlistRepository.DeletePlaylistThumbnail(ctx, playlistId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *PlaylistService) DeleteUserPlaylists(
	ctx context.Context,
	userId uuid.UUID,
) error {
	var offset int
	for {
		playlists, _, err := s.playlistRepository.GetUserPlaylists(
			ctx,
			userId,
			models.PlaylistFilter{
				Offset: offset,
			},
		)
		if err != nil {
			return err
		}

		if len(playlists) == 0 {
			break
		}

		for _, playlist := range playlists {
			if err := s.DeletePlaylistById(ctx, userId, playlist.Id); err != nil {
				return err
			}
		}

		offset += len(playlists)
	}

	return nil
}
