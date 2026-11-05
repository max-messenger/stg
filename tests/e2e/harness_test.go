package e2e_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"stg/internal/app/domain"
	"stg/internal/app/router/todolistctrl"
)

var stgHTTPAddr = envOrDefault("INTEGRATION_STG_HTTP_ADDR", "localhost:8080")

const botToken = "test-token"

// cookieClient использует CookieJar — куки от /auth/token автоматически
// отправляются на последующие запросы.
var cookieClient *http.Client

func init() {
	jar, _ := cookiejar.New(nil)
	cookieClient = &http.Client{Jar: jar}
}

func nextUserID() int64 {
	var buf [8]byte
	_, _ = rand.Read(buf[:])

	return int64(binary.BigEndian.Uint64(buf[:])%math.MaxInt32) + 1
}

// --- Auth helpers ---

func authenticateUser(t *testing.T, userID int64) {
	t.Helper()

	initData := buildInitData(t, userID)

	resp := postJSON(t, "/v1/auth/login", map[string]string{"init_data": initData})
	requireStatus(t, http.StatusOK, resp)
	resp.Body.Close()

	// Куки (access_token, refresh_token) сохранены в cookieClient.Jar
}

func buildInitData(t *testing.T, userID int64) string {
	t.Helper()

	authDate := time.Now().Unix()

	userJSON := fmt.Sprintf(`{"id":%d,"first_name":"Test","last_name":"User","username":"testuser","language_code":"ru"}`, userID)
	chatJSON := fmt.Sprintf(`{"id":%d,"type":"DIALOG"}`, userID)

	values := url.Values{
		"query_id":    {"test-query-id"},
		"user":        {userJSON},
		"chat":        {chatJSON},
		"auth_date":   {strconv.FormatInt(authDate, 10)},
		"start_param": {"promo"},
	}

	hash := computeHash(t, values)
	values.Set("hash", hash)

	return values.Encode()
}

func computeHash(t *testing.T, values url.Values) string {
	t.Helper()

	copied := url.Values{}
	for k, v := range values {
		copied[k] = v
	}

	copied.Del("hash")

	pairs := make([]string, 0, len(copied))
	for key := range copied {
		pairs = append(pairs, key+"="+copied.Get(key))
	}

	sort.Strings(pairs)

	launchParams := strings.Join(pairs, "\n")

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	sig := hmac.New(sha256.New, secretKey.Sum(nil))
	sig.Write([]byte(launchParams))

	return hex.EncodeToString(sig.Sum(nil))
}

// --- HTTP helpers ---

func stgURL(path string) string {
	return fmt.Sprintf("http://%s%s", stgHTTPAddr, path)
}

func postJSON(t *testing.T, path string, body any) *http.Response {
	t.Helper()

	data, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, stgURL(path), bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cookieClient.Do(req)
	require.NoError(t, err)

	return resp
}

func getJSON(t *testing.T, path string) *http.Response {
	t.Helper()

	resp, err := cookieClient.Get(stgURL(path))
	require.NoError(t, err)

	return resp
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var out struct {
		Data T `json:"data"`
	}

	err = json.Unmarshal(raw, &out)
	require.NoError(t, err)

	return out.Data
}

func requireStatus(t *testing.T, expected int, resp *http.Response) {
	t.Helper()

	if resp.StatusCode != expected {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d, body: %s", expected, resp.StatusCode, string(raw))
	}
}

// --- Infrastructure ---

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

// --- Domain helpers ---

func todoItemToDomain(t *testing.T, item todolistctrl.TodoItem, userID int64) domain.Todo {
	t.Helper()

	created, err := time.Parse(time.RFC3339, item.Created)
	require.NoError(t, err)

	return domain.Todo{
		ID:      item.ID,
		UserID:  userID,
		Message: item.Message,
		Done:    item.Done,
		Created: created,
	}
}
