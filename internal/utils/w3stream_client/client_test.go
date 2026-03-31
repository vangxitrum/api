package w3stream_client_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/w3stream_client"
)

func TestUploadMedia(t *testing.T) {
	file, err := os.Open("client_test.go")
	assert.NoError(t, err)

	defer file.Close()

	stat, err := file.Stat()
	assert.NoError(t, err)

	client, err := w3stream_client.NewW3streamClient("10.0.0.48:8083")
	assert.NoError(t, err)

	err = client.UploadMedia(
		context.Background(),
		"39d29c6f-c08a-4dbe-a9ea-3de66b7be964",
		"output.mkv",
		stat.Size(),
		file,
	)

	assert.NoError(t, err)
}
