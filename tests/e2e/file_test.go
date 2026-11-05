package e2e_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2EFile_UploadAndDownload(t *testing.T) {
	key := uploadFile(t, "hello.txt", []byte("hello from e2e"))

	resp, err := http.Get(stgURL("/v1/files/" + key))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("hello from e2e"), got)
}

func TestE2EFile_Delete(t *testing.T) {
	key := uploadFile(t, "delete-me.txt", []byte("temporary"))

	req, _ := http.NewRequest(http.MethodDelete, stgURL("/v1/files/"+key), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	getResp, err := http.Get(stgURL("/v1/files/" + key))
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestE2EFile_DownloadNotFound(t *testing.T) {
	resp, err := http.Get(stgURL("/v1/files/nonexistent-key"))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2EWebhook_SecretCheck(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, stgURL("/webhook"), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Max-Bot-Api-Secret", "wrong-secret")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2EWebhook_ValidUpdate(t *testing.T) {
	body := `{"update_type":"message_created","timestamp":1234567890,"message":{"recipient":{"user_id":1},"body":{"mid":"1","seq":1,"text":"test"}}}`
	req, _ := http.NewRequest(http.MethodPost, stgURL("/webhook"), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Max-Bot-Api-Secret", "test-secret")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- Helpers ---

func uploadFile(t *testing.T, filename string, content []byte) string {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, stgURL("/v1/files/"), &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	requireStatus(t, http.StatusOK, resp)

	type uploadResp struct {
		Key string `json:"key"`
	}

	result := decodeJSON[uploadResp](t, resp)
	require.NotEmpty(t, result.Key)

	return result.Key
}
