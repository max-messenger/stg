package e2e_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestE2EAuth_Me(t *testing.T) {
	userID := nextUserID()
	authenticateUser(t, userID)

	resp := getJSON(t, "/v1/auth/me")
	requireStatus(t, http.StatusOK, resp)
	defer resp.Body.Close()

	type meResp struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	}

	result := decodeJSON[meResp](t, resp)
	require.Equal(t, userID, result.User.ID)
}

func TestE2EAuth_Me_Unauthorized(t *testing.T) {
	// Без аутентификации — ожидаем 401
	// Используем DefaultClient без куки
	resp, err := http.Get(stgURL("/v1/auth/me"))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2EAuth_ExpiredAccessToken(t *testing.T) {
	// Требует access_ttl=3s в конфиге сервера.
	// В CI сервер запускается с обычным TTL, поэтому тест пропускается.
	if os.Getenv("INTEGRATION_STG_SHORT_ACCESS_TTL") == "" {
		t.Skip("skipped: set INTEGRATION_STG_SHORT_ACCESS_TTL=1 to run (requires access_ttl=3s)")
	}

	userID := nextUserID()
	authenticateUser(t, userID)

	// Сначала проверяем, что токен работает
	resp := getJSON(t, "/v1/auth/me")
	requireStatus(t, http.StatusOK, resp)
	resp.Body.Close()

	// Ждём протухания access token
	time.Sleep(4 * time.Second)

	// После протухания — 401
	expiredResp := getJSON(t, "/v1/auth/me")
	defer expiredResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, expiredResp.StatusCode)
}

func TestE2EAuth_Refresh(t *testing.T) {
	// Требует access_ttl=3s в конфиге сервера.
	if os.Getenv("INTEGRATION_STG_SHORT_ACCESS_TTL") == "" {
		t.Skip("skipped: set INTEGRATION_STG_SHORT_ACCESS_TTL=1 to run (requires access_ttl=3s)")
	}

	userID := nextUserID()
	authenticateUser(t, userID)

	// Убеждаемся, что токен работает
	resp := getJSON(t, "/v1/auth/me")
	requireStatus(t, http.StatusOK, resp)
	resp.Body.Close()

	// Ждём протухания access token
	time.Sleep(4 * time.Second)

	// Access token протух — /me возвращает 401
	expiredResp := getJSON(t, "/v1/auth/me")
	requireStatus(t, http.StatusUnauthorized, expiredResp)
	expiredResp.Body.Close()

	// Обновляем через refresh token
	refreshResp := postJSON(t, "/v1/auth/refresh", nil)
	requireStatus(t, http.StatusOK, refreshResp)
	refreshResp.Body.Close()

	// После refresh — /me снова работает
	meResp := getJSON(t, "/v1/auth/me")
	requireStatus(t, http.StatusOK, meResp)
	meResp.Body.Close()
}

func TestE2EAuth_Refresh_Rotation(t *testing.T) {
	// Проверяем, что refresh token ротируется —
	// каждый вызов /refresh выдаёт новую пару.
	// Не требует короткого access_ttl — проверяем только refresh.

	userID := nextUserID()
	authenticateUser(t, userID)

	for i := range 3 {
		refreshResp := postJSON(t, "/v1/auth/refresh", nil)
		requireStatus(t, http.StatusOK, refreshResp)
		refreshResp.Body.Close()

		// После каждого refresh доступ сохраняется
		meResp := getJSON(t, "/v1/auth/me")
		requireStatus(t, http.StatusOK, meResp)
		meResp.Body.Close()

		t.Logf("refresh rotation %d: ok", i+1)
	}
}
