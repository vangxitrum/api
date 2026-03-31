package transcribe_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type TranscribeClient struct {
	baseUrl string
	apiKey  string
	modelId string
}

func NewTranscribeClient(baseUrl, apiKey, modelId string) *TranscribeClient {
	return &TranscribeClient{
		baseUrl: baseUrl,
		apiKey:  apiKey,
		modelId: modelId,
	}
}

type CreateTaskRequest struct {
	Files []*struct {
		Key  string `json:"key"`
		Data string `json:"data"`
		Name string `json:"name"`
	} `json:"files"`
	ModelId     string `json:"model_id"`
	InputParams struct {
		Language  string `json:"language"`
		ModelName string `json:"model_name"`
	} `json:"input_params"`
}

type CreateTaskResponse struct {
	Status string `json:"status"`
	Data   struct {
		Data string `json:"data"`
	} `json:"data"`
}

func (c *TranscribeClient) CreateTask(
	ctx context.Context,
	videoId uuid.UUID,
	language, url string,
) (string, error) {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
	}

	payload := &CreateTaskRequest{
		Files: []*struct {
			Key  string `json:"key"`
			Data string `json:"data"`
			Name string `json:"name"`
		}{
			{
				Key:  "audio_file",
				Data: url,
				Name: "audio.mp3",
			},
		},
		ModelId: c.modelId,
		InputParams: struct {
			Language  string `json:"language"`
			ModelName string `json:"model_name"`
		}{
			Language:  language,
			ModelName: "Base",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/api-key/task", c.baseUrl),
		bytes.NewBuffer(data),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf(
			"failed to create task: response code %s, %s",
			resp.Status,
			string(body),
		)
	}

	defer resp.Body.Close()

	var createTaskResponse CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&createTaskResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return createTaskResponse.Data.Data, nil
}

var GenerateFailError = fmt.Errorf("generate fail")

type GetTaskResultResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result struct {
			OutputFile string `json:"output_file"`
			OutputText string `json:"output_text"`
		} `json:"result"`
		State   string `json:"state"`
		Success bool   `json:"success"`
	} `json:"data"`
}

func (c *TranscribeClient) GetTaskResult(
	ctx context.Context,
	taskId string,
) (io.ReadCloser, error) {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 3 * time.Second,
	}
	client := http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/api-key/task/%s/result", c.baseUrl, taskId),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get task result: response code %s", resp.Status)
	}

	var getTaskResultResponse GetTaskResultResponse
	if err := json.NewDecoder(resp.Body).Decode(&getTaskResultResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if getTaskResultResponse.Data.State != "SUCCESS" {
		if getTaskResultResponse.Data.State == "FAILURE" {
			return nil, GenerateFailError
		}

		return nil, fmt.Errorf("task failed: %s", getTaskResultResponse.Data.State)
	}

	resp, err = client.Get(getTaskResultResponse.Data.Result.OutputFile)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get task result file: response code %s", resp.Status)
	}

	return resp.Body, nil
}
