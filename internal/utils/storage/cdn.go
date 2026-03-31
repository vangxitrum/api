package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	custom_log "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/log"
)

var logger *slog.Logger = slog.Default()

var UploadSpeed int64 = 10 * 100

type CdnHelper struct {
	cdnUrl          string
	hubUrl          string
	businessAddress string
	ticketMapping   *sync.Map
	retry           int
}

type Option func(*CdnHelper)

type fileRecord struct {
	ID       string `json:"ID"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	ReaderID int    `json:"readerId"`
}

type zipHeader struct {
	File []struct {
		Name               string `json:"Name"`
		CompressedSize     uint32 `json:"CompressedSize"`
		UncompressedSize   uint32 `json:"UncompressedSize"`
		CompressedSize64   uint64 `json:"CompressedSize64"`
		UncompressedSize64 uint64 `json:"UncompressedSize64"`
		Offset             uint64 `json:"Offset"`
	}
}

type UploadFileResponse struct {
	FileRecord fileRecord
	ZipHeader  zipHeader
}

func MustNewCdnHelper(
	cdnUrl, hubUrl, bussinessAddress string, options ...Option,
) StorageHelper {
	helper := &CdnHelper{
		cdnUrl:          cdnUrl,
		hubUrl:          hubUrl,
		businessAddress: bussinessAddress,
		ticketMapping:   &sync.Map{},
		retry:           5,
	}

	for _, option := range options {
		option(helper)
	}

	if logger == nil {
		logger = slog.New(custom_log.NewHandler(&slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	if err := helper.getBalance(); err != nil {
		panic(
			fmt.Sprintf(
				"can not init cdn with err %s",
				err.Error(),
			),
		)
	}

	if helper.retry < 1 {
		panic("retry must bigger than 1")
	}

	return helper
}

func WithRetry(retry int) Option {
	return func(h *CdnHelper) {
		h.retry = retry
	}
}

func WithTicketMapping(ticketMapping *sync.Map) Option {
	return func(h *CdnHelper) {
		h.ticketMapping = ticketMapping
	}
}

func WithDebugLog() Option {
	return func(h *CdnHelper) {
		logger = slog.New(custom_log.NewHandler(&slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
}

func (h *CdnHelper) handleRequest(
	ctx context.Context,
	req *http.Request,
	expectCode int,
	timeout time.Duration,
	needRetry bool,
	readers []io.Reader,
) (*http.Response, error) {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 3 * time.Second,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
	}

	client := http.Client{
		Transport: transport,
		Timeout:   max(10*time.Second, timeout),
	}

	var (
		count int
		resp  *http.Response
		err   error
	)

	start := time.Now().UTC()
	defer func() {
		diff := time.Since(start).Seconds()
		if diff >= timeout.Seconds()*0.5 {
			logger.WarnContext(
				ctx,
				"cdn response too long",
				slog.Any(
					"url",
					req.URL.String(),
				),
				slog.Any(
					"response time",
					diff,
				),
				slog.Any(
					"retry time",
					count,
				),
			)
		}
	}()
	for {
		resp, err = func() (
			*http.Response, error,
		) {
			now := time.Now().UTC()

			resp, err = client.Do(req)
			if err != nil {
				return nil, err
			}

			if resp.StatusCode != expectCode {
				defer resp.Body.Close()
				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.ErrorContext(
						ctx,
						"read response body error",
						slog.Any("err", err),
						slog.Any("url", req.URL.String()),
						slog.Any("responseTime", time.Since(now).Seconds()),
					)
				}

				return nil, fmt.Errorf(
					"Expect code is %d but cdn response: %d, body: %s, url:	%s",
					expectCode,
					resp.StatusCode,
					string(respBody),
					req.URL.String(),
				)
			}

			return resp, nil
		}()
		if err != nil {
			logger.ErrorContext(
				ctx,
				"cdn request error",
				slog.Any("err", err),
			)
			if count == h.retry || !needRetry {
				return nil, err
			}

			if len(readers) > 0 {
				body := new(bytes.Buffer)
				writer := multipart.NewWriter(body)
				for _, reader := range readers {
					if readSeeker, ok := reader.(io.ReadSeeker); ok {
						if _, err := readSeeker.Seek(0, io.SeekStart); err != nil {
							return nil, err
						}

						part, err := writer.CreateFormFile(
							"file",
							"file",
						)
						if err != nil {
							return nil, err
						}

						if n, err := io.Copy(part, readSeeker); err != nil || n == 0 {
							if n == 0 {
								return nil, errors.New("empty readers")
							}

							return nil, err
						}

					} else {
						return nil, err
					}
				}

				if err := writer.Close(); err != nil {
					return nil, err
				}
				newReq, err := http.NewRequest(
					req.Method,
					req.URL.String(),
					body,
				)
				if err != nil {
					return nil, err
				}

				newReq.Header.Add(
					"Content-Type",
					writer.FormDataContentType(),
				)

				req = newReq
			}

			count++
			continue
		}

		return resp, nil
	}
}

func (h *CdnHelper) Uploads(
	ctx context.Context,
	data string, files map[string]io.Reader,
) ([]*Object, int64, error) {
	now := time.Now().UTC()
	var timeout int64
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn upload files info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any(
				"file count",
				len(files),
			),
			slog.Any("timeout", timeout),
		)
	}()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	var totalByte int64
	readers := make([]io.Reader, 0, len(files))
	for name, reader := range files {
		part, err := writer.CreateFormFile(
			"file",
			name,
		)
		if err != nil {
			return nil, 0, err
		}

		n, err := io.Copy(
			part,
			reader,
		)
		if err != nil {
			return nil, 0, err
		}

		readers = append(readers, reader)

		totalByte += n
	}

	if err := writer.Close(); err != nil {
		return nil, 0, err
	}

	timeout = totalByte / UploadSpeed
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/packUpload", h.cdnUrl),
		body,
	)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Add(
		"Content-Type",
		writer.FormDataContentType(),
	)
	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		time.Duration(timeout)*time.Second,
		true,
		readers,
	)
	if err != nil {
		return nil, 0, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf(
			"uploader response with code: %d",
			resp.StatusCode,
		)
	}

	var rs UploadFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, 0, err
	}

	if len(rs.ZipHeader.File) == 0 {
		return nil, 0, fmt.Errorf("zip header's file is empty")
	}

	objects := make(
		[]*Object,
		0,
		len(rs.ZipHeader.File),
	)
	for _, file := range rs.ZipHeader.File {
		objects = append(
			objects, &Object{
				Name:   file.Name,
				Id:     rs.FileRecord.ID,
				Offset: int64(file.Offset),
				Size:   int64(file.UncompressedSize64),
			},
		)
	}

	return objects, rs.FileRecord.Size, nil
}

func (h *CdnHelper) Upload(
	ctx context.Context,
	data string, reader io.Reader,
) (*Object, error) {
	now := time.Now().UTC()
	var timeout int64
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn upload file info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("timeout", timeout),
		)
	}()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(
		"file",
		data,
	)
	if err != nil {
		return nil, err
	}

	n, err := io.Copy(
		part,
		reader,
	)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	timeout = n / UploadSpeed
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/packUpload", h.cdnUrl),
		body,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Add(
		"Content-Type",
		writer.FormDataContentType(),
	)
	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		time.Duration(timeout)*time.Second,
		true,
		[]io.Reader{reader},
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"uploader response with code: %d",
			resp.StatusCode,
		)
	}

	var rs UploadFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	if len(rs.ZipHeader.File) == 0 {
		return nil, fmt.Errorf("zip header's file is empty")
	}

	return &Object{
		Id:     rs.FileRecord.ID,
		Offset: int64(rs.ZipHeader.File[0].Offset),
		Size:   int64(rs.ZipHeader.File[0].UncompressedSize64),
	}, nil
}

func (h *CdnHelper) UploadZip(
	ctx context.Context,
	data string, reader io.Reader,
) (*Object, error) {
	now := time.Now().UTC()
	var timeout int64
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn upload zip file info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("timeout", timeout),
		)
	}()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(
		"file",
		data,
	)
	if err != nil {
		return nil, err
	}

	n, err := io.Copy(
		part,
		reader,
	)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	timeout = n / UploadSpeed
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf(
			"%s/packUpload?name=%s",
			h.cdnUrl,
			data,
		),
		body,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Add(
		"Content-Type",
		writer.FormDataContentType(),
	)
	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		time.Duration(timeout)*time.Second,
		true,
		[]io.Reader{reader},
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"uploader response with code: %d",
			resp.StatusCode,
		)
	}

	var rs UploadFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	if len(rs.ZipHeader.File) == 0 {
		return nil, fmt.Errorf("zip header's file is empty")
	}

	return &Object{
		Id:     rs.FileRecord.ID,
		Offset: int64(rs.ZipHeader.File[0].Offset),
		Size:   int64(rs.ZipHeader.File[0].UncompressedSize64),
	}, nil
}

type UploadRawResponse struct {
	FileId string `json:"fileId"`
	Url    string `json:"url"`
}

func (h *CdnHelper) UploadRaw(
	ctx context.Context,
	data string, size int64, reader io.Reader,
) (*Object, error) {
	now := time.Now().UTC()
	var timeout int64
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn upload raw file info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("timeout", timeout),
			slog.Any("size", size),
		)
	}()

	timeout = size / UploadSpeed
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf(
			"%s/uploadRaw?name=%s",
			h.cdnUrl,
			data,
		),
		reader,
	)
	if err != nil {
		return nil, err
	}

	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		time.Duration(timeout)*time.Second,
		false,
		[]io.Reader{reader},
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"uploader response with code: %d",
			resp.StatusCode,
		)
	}

	var rs UploadRawResponse
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	return &Object{
		Id:     rs.FileId,
		Offset: 0,
		Size:   size,
	}, nil
}

func (h *CdnHelper) Download(ctx context.Context, obj *Object) (
	io.Reader, error,
) {
	now := time.Now().UTC()
	timeout := obj.Size / 10 / 1024
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn download file info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("file id", obj.Id),
			slog.Any("offset", obj.Offset),
			slog.Any("size", obj.Size),
			slog.Any("timeout", timeout),
		)
	}()

	endPoint, err := url.Parse(
		fmt.Sprintf(
			"%s/cacheFile/%s",
			h.cdnUrl,
			obj.Id,
		),
	)
	if err != nil {
		return nil, err
	}

	rawQuery := endPoint.Query()
	rawQuery.Set(
		"range",
		fmt.Sprintf(
			"%d,%d",
			obj.Offset,
			obj.Size,
		),
	)

	endPoint.RawQuery = rawQuery.Encode()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endPoint.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "*/*")

	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		time.Duration(timeout)*time.Second,
		true,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (h *CdnHelper) Delete(ctx context.Context, obj *Object) error {
	now := time.Now().UTC()
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn delete file info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("file id", obj.Id),
		)
	}()

	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf(
			"%s/endFileRecord?file_record_id=%s",
			h.cdnUrl,
			obj.Id,
		),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		3*time.Second,
		true,
		nil,
	)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"cdn response with code: %d, fild id: %v",
			resp.StatusCode,
			obj.Id,
		)
	}

	return nil
}

func (h *CdnHelper) GetLink(ctx context.Context, obj *Object) (
	string, int64, error,
) {
	now := time.Now().UTC()
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn get link file info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("file id", obj.Id),
			slog.Any("offset", obj.Offset),
			slog.Any("size", obj.Size),
		)
	}()

	tickerKey := fmt.Sprintf("%s-%d-%d", obj.Id, obj.Offset, obj.Size)
	dt, ok := h.ticketMapping.Load(tickerKey)
	if ok {
		lastTicket, ok := dt.(GetTicketResponse)
		if ok && lastTicket.ExpiredAt-1000000 > time.Now().UTC().UnixNano() {
			return fmt.Sprintf(
				"%s/file/%s?expire=%d&signature=%s&range=%d,%d",
				h.hubUrl,
				obj.Id,
				lastTicket.ExpiredAt,
				lastTicket.DecodedSignature,
				obj.Offset,
				obj.Size,
			), lastTicket.ExpiredAt - time.Second.Nanoseconds(), nil
		}
	}

	canGenerate, err := h.canGeneratePresignedLink(ctx, obj.Id)
	if err != nil {
		return "", 0, err
	}

	if !canGenerate {
		return "", 0, nil
	}

	ticket, err := h.getFileRecordTicket(
		ctx,
		obj.Id,
		fmt.Sprintf(
			"%d,%d",
			obj.Offset,
			obj.Size,
		),
	)
	if err != nil {
		return "", 0, err
	}

	h.ticketMapping.Store(tickerKey, *ticket)
	return fmt.Sprintf(
		"%s/file/%s?expire=%d&signature=%s&range=%s",
		h.hubUrl,
		obj.Id,
		ticket.ExpiredAt,
		ticket.DecodedSignature,
		fmt.Sprintf(
			"%d,%d",
			obj.Offset,
			obj.Size,
		),
	), ticket.ExpiredAt - time.Second.Nanoseconds(), nil
}

func (c *CdnHelper) GetFileRecord(
	ctx context.Context,
	fileId string,
) (*GetFileRecordResponse, error) {
	now := time.Now().UTC()
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn get file record",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("file id", fileId),
		)
	}()

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/getFileRecord/%s",
			c.cdnUrl,
			fileId,
		),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.handleRequest(
		ctx,
		req,
		http.StatusOK,
		3*time.Second,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var rs GetFileRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	return &rs, nil
}

func (c *CdnHelper) GetZipHeader(ctx context.Context, fileId string) (*zipHeader, error) {
	now := time.Now().UTC()
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn get zip header",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("file id", fileId),
		)
	}()

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/getZipHeaders/%s",
			c.cdnUrl,
			fileId,
		),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.handleRequest(
		ctx,
		req,
		http.StatusOK,
		3*time.Second,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var rs zipHeader
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	return &rs, nil
}

var FileRecordReadyStatus int = 2

type GetFileRecordResponse struct {
	Id     string `json:"ID"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Status int    `json:"status"`
}

var (
	TranscodingStatus = "transcoding"
	TranscodedStatus  = "transcoded"
	FailStatus        = "fail"
)

func (h *CdnHelper) GetTranscodeStatus(ctx context.Context, fileId string) (string, error) {
	now := time.Now().UTC()
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn get transcode status info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("file id", fileId),
		)
	}()

	resp, err := h.GetFileRecord(ctx, fileId)
	if err != nil {
		return "", err
	}

	switch resp.Status {
	case 1:
		return TranscodingStatus, nil
	case 2:
		return TranscodedStatus, nil
	default:
		return FailStatus, nil
	}
}

func (h *CdnHelper) canGeneratePresignedLink(ctx context.Context, fileId string) (
	bool, error,
) {
	resp, err := h.GetFileRecord(ctx, fileId)
	if err != nil {
		return false, err
	}

	return resp.Status == FileRecordReadyStatus, nil
}

type GetTicketResponse struct {
	Version          int    `json:"version"`
	FileRecordId     string `json:"file_record_id"`
	Signature        string `json:"signature"`
	DecodedSignature string `json:"-"`
	ExpiredAt        int64  `json:"expire_at_ns"`
}

func (h *CdnHelper) getFileRecordTicket(
	ctx context.Context, fileId string,
	fileRange string,
) (*GetTicketResponse, error) {
	var data GetTicketResponse
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/getTicket?id=%s&range=%s",
			h.cdnUrl,
			fileId,
			fileRange,
		),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		3*time.Second,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, nil
	}

	decodeSignature, err := base64.StdEncoding.DecodeString(data.Signature)
	if err != nil {
		return nil, err
	}

	data.DecodedSignature = base64.RawURLEncoding.EncodeToString(decodeSignature)

	return &data, nil
}

type GetBalanceResponse struct {
	DepositAddress string `json:"deposit_address"`
	SetCreditLater bool   `json:"set_credit_later"`
}

func (h *CdnHelper) getBalance() error {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/getBalance", h.cdnUrl),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := h.handleRequest(
		context.Background(),
		req,
		http.StatusOK,
		3*time.Second,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	var data GetBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	if data.DepositAddress != h.businessAddress {
		return fmt.Errorf(
			"business address is not match, cdn address: %s, business address: %s",
			data.DepositAddress,
			h.businessAddress,
		)
	}

	if !data.SetCreditLater {
		return fmt.Errorf("cdn not set credit later")
	}

	return nil
}

type GetAiozPriceResponse struct {
	AiozPrice string `json:"aioz_price"`
}

func (h *CdnHelper) GetAIOZPrice(ctx context.Context) (
	float64, error,
) {
	now := time.Now().UTC()
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn get aioz price info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
		)
	}()

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/currentPrice", h.cdnUrl),
		nil,
	)
	if err != nil {
		return 0, err
	}

	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		3*time.Second,
		true,
		nil,
	)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()
	var getPriceResp GetAiozPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&getPriceResp); err != nil {
		return 0, err
	}

	rs, err := strconv.ParseFloat(
		getPriceResp.AiozPrice,
		64,
	)
	if err != nil {
		return 0, err
	}

	return rs, nil
}

var (
	H264Encoder = "h264_nvenc"
	HEVCEncoder = "hevc_nvenc"

	AccCodec = "aac"

	FileStructType    = "file"
	SegmentStructType = "segment"

	Mp4Container  = "mp4"
	TsContainer   = "ts"
	WebmContainer = "webm"
	M4sContainer  = "m4s"
)

type TranscodeProfile struct {
	Version              int    `json:"version"`
	Encoder              string `json:"encoder"`
	OutputVwidth         int    `json:"output_vwidth"`
	OutputVheight        int    `json:"output_vheight"`
	OutputVmaxRate       int    `json:"output_vmax_rate"`
	OutputTranscodeAudio bool   `json:"output_transcode_audio"`
	OutputAcodec         string `json:"output_acodec"`
	OutputAbitrate       int    `json:"output_abitrate"`
	OutputStructType     string `json:"output_struct_type"`
	OutputContainer      string `json:"output_container"`
	SegmentNameTemplate  string `json:"segment_name_template"`
	SegmentDuration      int    `json:"segment_duration"`
	PlaylistName         string `json:"playlist_name"`
}

type TranscodeResponse struct {
	FileRecordId string `json:"file_record_id"`
}

func (h *CdnHelper) Transcode(
	ctx context.Context,
	fileId string,
	profile *TranscodeProfile,
) (*TranscodeResponse, error) {
	now := time.Now().UTC()
	var rs TranscodeResponse
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn transcode file info",
			slog.Any(
				"run time",
				time.Since(now).Seconds(),
			),
			slog.Any("source id", fileId),
			slog.Any("profile", profile),
			slog.Any("response", rs),
		)
	}()

	data, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/transcode?file_record_id=%s", h.cdnUrl, fileId),
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}

	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		3*time.Second,
		true,
		nil,
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	return &rs, nil
}

func (h *CdnHelper) PackUploadByte(ctx context.Context, name string, data []byte) (*Object, error) {
	now := time.Now().UTC()
	var timeout int64
	defer func() {
		logger.DebugContext(
			ctx,
			"Cdn upload file info",
			slog.Any("run time", time.Since(now).Seconds()),
			slog.Any("timeout", timeout),
		)
	}()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, err
	}

	n, err := part.Write(data)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	timeout = int64(n) / 10 / 1024
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/packUpload", h.cdnUrl),
		body,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Add(
		"Content-Type",
		writer.FormDataContentType(),
	)
	resp, err := h.handleRequest(
		ctx,
		req,
		http.StatusOK,
		time.Duration(timeout)*time.Second,
		true,
		[]io.Reader{bytes.NewReader(data)},
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"uploader response with code: %d",
			resp.StatusCode,
		)
	}

	var rs UploadFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	if len(rs.ZipHeader.File) == 0 {
		return nil, fmt.Errorf("zip header's file is empty")
	}

	return &Object{
		Id:     rs.FileRecord.ID,
		Offset: int64(rs.ZipHeader.File[0].Offset),
		Size:   int64(rs.ZipHeader.File[0].UncompressedSize64),
		Name:   name,
	}, nil
}

type GetDetailBalanceResponse struct {
	Credit                 string `json:"credit"`
	DeliveryCreditExpense  string `json:"delivery_credit_expense"`
	DepositAddress         string `json:"deposit_address"`
	SetCreditLater         bool   `json:"set_credit_later"`
	StorageCreditExpense   string `json:"storage_credit_expense"`
	TranscodeCreditExpense string `json:"transcode_credit_expense"`
}

func (h *CdnHelper) GetDetailBalance(ctx context.Context) (*GetDetailBalanceResponse, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/getBalance", h.cdnUrl),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := h.handleRequest(
		context.Background(),
		req,
		http.StatusOK,
		3*time.Second,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var data GetDetailBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.DepositAddress != h.businessAddress {
		return nil, fmt.Errorf(
			"business address is not match, cdn address: %s, business address: %s",
			data.DepositAddress,
			h.businessAddress,
		)
	}

	if !data.SetCreditLater {
		return nil, fmt.Errorf("cdn not set credit later")
	}

	return &data, nil
}
