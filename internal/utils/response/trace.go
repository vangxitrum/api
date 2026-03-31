package response

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var TraceHelperInstance *TraceHelper

const (
	DEFAULT_OUTPUT        = "./trace_data"
	MAX_TRACE_FOLDER_SIZE = 100 * 1024 * 1024
	MAX_TRACE_FILES       = 5
)

type TraceHelper struct {
	Output string
	Total  int64
	ticker *time.Ticker
}

func NewTraceHelper(output string) error {
	output = filepath.Clean(output)
	absPath, err := filepath.Abs(output)
	if err != nil {
		return err
	}

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if absPath == "" || absPath == workDir {
		absPath = DEFAULT_OUTPUT
	}

	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(absPath, 0755); err != nil {
				return err
			}
		}

		return err
	}

	var total int64
	if err := filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		total += info.Size()

		return nil
	}); err != nil {
		return err
	}

	TraceHelperInstance = &TraceHelper{
		Output: absPath,
		Total:  total,
		ticker: time.NewTicker(1 * time.Second),
	}

	go func() {
		for range TraceHelperInstance.ticker.C {
			if err := TraceHelperInstance.Clean(); err != nil {
				slog.Error("error when cleaning trace data", slog.Any("err", err))
			}
		}
	}()

	return nil
}

func (t *TraceHelper) Save(traceId string, data any) error {
	outputFile := fmt.Sprintf("%s/%s.json", t.Output, traceId)
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

func (t *TraceHelper) Load(traceId string) ([]byte, error) {
	output := fmt.Sprintf("%s/%s.json", t.Output, traceId)
	file, err := os.Open(output)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	data := make([]byte, stat.Size())
	if _, err := file.Read(data); err != nil {
		return nil, err
	}

	return data, nil
}

func (t *TraceHelper) Clean() error {
	files, err := os.ReadDir(t.Output)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		fiStat, _ := files[i].Info()
		fjStat, _ := files[j].Info()

		if fiStat.ModTime().Before(fjStat.ModTime()) {
			return true
		}

		return false
	})

	count := len(files)
	for _, file := range files {
		if t.Total < MAX_TRACE_FOLDER_SIZE || count <= MAX_TRACE_FILES {
			break
		}

		filePath := fmt.Sprintf("%s/%s", t.Output, file.Name())
		fileStat, _ := file.Info()
		if err := os.Remove(filePath); err != nil {
			return err
		}

		t.Total -= fileStat.Size()
		count--
	}

	return nil
}

func (t *TraceHelper) Close() {
	t.ticker.Stop()
}
