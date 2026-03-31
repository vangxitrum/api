package stream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

type StreamClient struct {
	streamApiUrl  string
	streamApiUser string
	streamApiPass string
}

func NewStreamClient(
	streamApiUrl string,
	streamApiUser string,
	streamApiPass string,
) *StreamClient {
	return &StreamClient{
		streamApiUrl:  streamApiUrl,
		streamApiUser: streamApiUser,
		streamApiPass: streamApiPass,
	}
}

func (c *StreamClient) addBasicAuthStreamRequest(req *http.Request) {
	auth := c.streamApiUser + ":" + c.streamApiPass
	basicAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+basicAuth)
}

func (c *StreamClient) CreateStreamId(
	ctx context.Context,
	streamKey string,
	streamMediaId string,
) (string, error) {
	formData := url.Values{}
	formData.Add("streamKey", streamKey)
	formData.Add("streamId", streamMediaId)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.streamApiUrl+models.StreamCreatePath,
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		slog.Error(
			"Failed to create request create stream",
			slog.Any("err", err),
		)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.addBasicAuthStreamRequest(req)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error(
			"Failed to make request create stream",
			slog.Any("err", err),
		)
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"Failed to create stream",
			slog.Any("status", resp.StatusCode),
			slog.Any("body", string(body)),
		)
		return "", fmt.Errorf(
			"failed to create stream, status: %d, body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	return streamMediaId, nil
}

func (c *StreamClient) GetStreamPathWithStreamType(
	ctx context.Context,
	streamId string,
	streamType string,
) (*models.LiveStreamWebhookResponse, error) {
	requestUrl := fmt.Sprintf("%s%s%s", c.streamApiUrl, models.StreamGetPath, streamId)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, nil)
	if err != nil {
		slog.Error(
			"Failed to create request get stream path",
			slog.Any("err", err),
		)
		return nil, fmt.Errorf("Failed to get stream path request: %w", err)
	}

	c.addBasicAuthStreamRequest(req)
	client := &http.Client{}
	var lastErr error

	if streamType == models.StreamConnectPath {
		for i := range models.MaxRetries {
			shouldRetry := false

			resp, err := client.Do(req)
			if err != nil {
				slog.Error(
					"Failed to make request get stream path",
					slog.Any("err", err),
				)
				lastErr = fmt.Errorf("Failed to make request: %w", err)
				shouldRetry = true
			} else {
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					slog.Error(
						"Failed to get stream path",
						slog.Any("status", resp.StatusCode),
						slog.Any("body", string(body)),
					)
					lastErr = fmt.Errorf("Failed to get stream path, status: %d, body: %s", resp.StatusCode, string(body))
					shouldRetry = true
				} else {
					var webhookResp models.LiveStreamWebhookResponse
					if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
						slog.Error(
							"Failed to decode webhook response",
							slog.Any("err", err),
						)
						lastErr = err
						shouldRetry = true
					} else {
						if webhookResp.Path == "" {
							slog.Error(
								"Stream path is empty",
							)
							lastErr = fmt.Errorf("Stream path is empty")
							shouldRetry = true
						} else {
							parsedUUID, err := uuid.Parse(webhookResp.Path)
							if err != nil {
								slog.Error(
									"Invalid stream path format",
									slog.Any("err", err),
								)
								lastErr = fmt.Errorf("Invalid stream path format: %w", err)
								shouldRetry = true
							} else if parsedUUID != uuid.Nil {
								return &webhookResp, nil
							}
						}
					}
				}
			}
			if shouldRetry {
				backoff := models.InitialBackoff * time.Duration(1<<uint(i))
				if backoff > models.MaxBackoff {
					backoff = models.MaxBackoff
				}

				jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
				backoff = backoff/2 + jitter

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}
		}
		slog.Error(
			"Failed to get stream path",
			slog.Any("lastErr", lastErr),
		)
		return nil, fmt.Errorf("after %d retries: %v", models.MaxRetries, lastErr)
	}

	if streamType == models.StreamDisconnectPath {
		resp, err := client.Do(req)
		if err != nil {
			slog.Error(
				"Failed to make request",
				slog.Any("err", err),
			)
			return nil, fmt.Errorf("Failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}

		if resp.StatusCode == http.StatusOK {
			var webhookResp models.LiveStreamWebhookResponse
			if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
				slog.Error(
					"Failed to decode webhook response",
					slog.Any("err", err),
				)
				return nil, fmt.Errorf("Failed to decode webhook response: %w", err)
			}

			return &webhookResp, nil
		}

	}

	return nil, fmt.Errorf("Invalid stream type: %s", streamType)
}

func (c *StreamClient) DeleteRTMPConnection(ctx context.Context, streamKey, streamId string) error {
	baseURL := c.streamApiUrl + models.StreamDeleteStreamIdPath
	queryParams := url.Values{}
	queryParams.Add("streamKey", streamKey)
	queryParams.Add("streamId", streamId)

	fullURL := baseURL + "?" + queryParams.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fullURL,
		nil,
	)
	if err != nil {
		slog.Error(
			"Failed to create request delete RTMP connection",
			slog.Any("err", err),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.addBasicAuthStreamRequest(req)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error(
			"Failed to make request delete RTMP connection",
			slog.Any("err", err),
		)
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"Failed to delete RTMP connection",
			slog.Any("status", resp.StatusCode),
			slog.Any("body", string(body)),
		)
		return fmt.Errorf(
			"failed to delete RTMP connection, status: %d, body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	return nil
}

func (c *StreamClient) DeleteRTMPConnectionByStreamKey(
	ctx context.Context,
	streamKey string,
) error {
	baseURL := c.streamApiUrl + models.StreamDeleteStreamKeyPath
	queryParams := url.Values{}
	queryParams.Add("streamKey", streamKey)

	fullURL := baseURL + "?" + queryParams.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fullURL,
		nil,
	)
	if err != nil {
		slog.Error(
			"Failed to create request delete RTMP connection",
			slog.Any("err", err),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.addBasicAuthStreamRequest(req)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error(
			"Failed to make request delete RTMP connection",
			slog.Any("err", err),
		)
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"Failed to delete RTMP connection",
			slog.Any("status", resp.StatusCode),
			slog.Any("body", string(body)),
		)
		return fmt.Errorf(
			"failed to delete RTMP connection, status: %d, body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	return nil
}

func (c *StreamClient) GetRTMPListConnection(
	ctx context.Context,
) (*models.RTMPListConnectionResponse, error) {
	var result models.RTMPListConnectionResponse
	result.PageCount = 1
	for page := 0; page <= result.PageCount-1; page++ {

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			fmt.Sprintf("%s%s?page=%d", c.streamApiUrl, models.StreamListPath, page),
			nil,
		)
		if err != nil {
			slog.Error(
				"Failed to create request get RTMP list connection",
				slog.Any("err", err),
			)
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		c.addBasicAuthStreamRequest(req)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			slog.Error(
				"Failed to make request get RTMP list connection",
				slog.Any("err", err),
			)
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			slog.Error(
				"Failed to get RTMP list connection",
				slog.Any("status", resp.StatusCode),
				slog.Any("body", string(body)),
			)
			return nil, fmt.Errorf(
				"failed to get RTMP list connection, status: %d, body: %s",
				resp.StatusCode,
				string(body),
			)
		}

		var response models.RTMPListConnectionResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			slog.Error(
				"Failed to decode response",
				slog.Any("err", err),
			)
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		result.ItemCount = response.ItemCount
		result.PageCount = response.PageCount
		result.Items = append(result.Items, response.Items...)
	}
	return &result, nil
}

func (c *StreamClient) GetHLSMuxersList(ctx context.Context) (*models.HLSMuxerResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.streamApiUrl+models.StreamGetHLSMuxersListPath,
		nil,
	)
	if err != nil {
		slog.Error(
			"Failed to create request get HLS muxers list",
			slog.Any("err", err),
		)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addBasicAuthStreamRequest(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error(
			"Failed to make request get HLS muxers list",
			slog.Any("err", err),
		)
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"Failed to get HLS muxers list",
			slog.Any("status", resp.StatusCode),
			slog.Any("body", string(body)),
		)
		return nil, fmt.Errorf(
			"failed to get HLS muxers list, status: %d, body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var response models.HLSMuxerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		slog.Error(
			"Failed to decode response",
			slog.Any("err", err),
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (c *StreamClient) GetHLSMuxerInfo(ctx context.Context, pathId string) (string, error) {
	requestUrl := fmt.Sprintf("%s%s%s", c.streamApiUrl, models.StreamGetHLSMuxerPath, pathId)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, nil)
	if err != nil {
		slog.Error(
			"Failed to create request get HLS muxer info",
			slog.Any("err", err),
		)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.addBasicAuthStreamRequest(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error(
			"Failed to make request get HLS muxer info",
			slog.Any("err", err),
		)
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		slog.Error(
			"HLS muxer not found",
			slog.Any("pathId", pathId),
		)
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"Failed to get HLS muxer info",
			slog.Any("status", resp.StatusCode),
			slog.Any("body", string(body)),
		)
		return "", fmt.Errorf(
			"failed to get HLS muxer info, status: %d, body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var muxerInfo models.HLSMuxerItem
	if err := json.NewDecoder(resp.Body).Decode(&muxerInfo); err != nil {
		slog.Error(
			"Failed to decode response get HLS muxer info",
			slog.Any("err", err),
		)
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	twelveHoursAgo := time.Now().Add(-12 * time.Hour)
	if muxerInfo.Created.Before(twelveHoursAgo) {
		return muxerInfo.Path, nil
	}

	return "", nil
}

func (c *StreamClient) KickRTMPConnection(ctx context.Context, connectionId string) error {
	requestUrl := fmt.Sprintf("%s%s%s", c.streamApiUrl, models.StreamKickPath, connectionId)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestUrl, nil)
	if err != nil {
		slog.Error(
			"Error kicking RTMP connection",
			slog.Any("err", err),
		)
		return fmt.Errorf("failed to create kick request: %w", err)
	}

	c.addBasicAuthStreamRequest(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error(
			"Error kicking RTMP connection",
			slog.Any("err", err),
		)
		return fmt.Errorf("failed to make kick request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		slog.Error(
			"RTMP connection not found",
			slog.Any("connectionId", connectionId),
		)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error(
			"Failed to kick RTMP connection",
			slog.Any("status", resp.StatusCode),
			slog.Any("body", string(body)),
		)
		return fmt.Errorf("failed to kick RTMP connection, status: %d, body: %s",
			resp.StatusCode, string(body))
	}

	return nil
}
