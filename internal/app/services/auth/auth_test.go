package auth_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"stg/internal/app/domain"
	"stg/internal/app/services/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testBotToken = "test-bot-token"
	testUserId   = int64(12345)
)

func TestService(t *testing.T) {
	suite.Run(t, new(testService))
}

type testService struct {
	suite.Suite

	repo *MockTokenRepo
	srv  *auth.Service
}

func (t *testService) SetupTest() {
	ctrl := gomock.NewController(t.T())
	t.repo = NewMockTokenRepo(ctrl)

	config := auth.Config{
		JwtSecret:  testBotToken,
		AccessTTL:  time.Second,
		RefreshTTL: time.Second,
		MaxToken:   testBotToken,
	}
	t.srv = auth.New(t.repo, config)
}
func (t *testService) buildInitData(authDate int64) string {
	values := url.Values{
		"query_id":    {"test-query-id"},
		"user":        {`{"id":12345,"first_name":"Max","last_name":"User","username":"maxuser","language_code":"ru"}`},
		"chat":        {`{"id":67890,"type":"DIALOG"}`},
		"auth_date":   {strconv.FormatInt(authDate, 10)},
		"start_param": {"promo"},
	}

	hash := t.computeHash(values)
	values.Set("hash", hash)

	return values.Encode()
}

func (t *testService) computeHash(values url.Values) string {
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
	secretKey.Write([]byte(testBotToken))

	sig := hmac.New(sha256.New, secretKey.Sum(nil))
	sig.Write([]byte(launchParams))

	return hex.EncodeToString(sig.Sum(nil))
}

func (t *testService) TestUnitAuthService_ParseAccessToken_InvalidToken() {
	ctx := context.Background()
	_, err := t.srv.ParseAndValidateToken(ctx, "invalid-token", "access")
	t.Error(err)
}

func (t *testService) TestUnitAuthService_Authenticate_Success() {
	initData := t.buildInitData(time.Now().Unix())
	ctx := context.Background()

	t.repo.EXPECT().StoreRefresh(ctx, gomock.Any()).Return(nil)
	t.repo.EXPECT().GetRefresh(ctx, testUserId).Return(domain.AuthData{User: domain.AuthUser{ID: testUserId}}, nil)
	t.repo.EXPECT().DeleteRefresh(ctx, testUserId).Return(nil)

	data, err := t.srv.Authenticate(ctx, initData)
	t.NoError(err)

	t.Equal(testUserId, data.User.ID)

	t.NotEmpty(data.TokenPair.AccessToken)
	t.NotEmpty(data.TokenPair.RefreshToken)

	// Проверяем access token
	claims := jwt.MapClaims{}
	_, parseErr := jwt.ParseWithClaims(data.TokenPair.AccessToken, claims, func(_ *jwt.Token) (any, error) {
		return []byte(testBotToken), nil
	})
	t.NoError(parseErr)
	t.Equal(float64(testUserId), claims["user_id"])
}

func (t *testService) TestUnitAuthService_Authenticate_InvalidHash() {
	initData := "query_id=test&auth_date=" + strconv.FormatInt(time.Now().Unix(), 10) + "&hash=badhash"

	_, err := t.srv.Authenticate(context.Background(), initData)
	t.Require().Error(err)
}

func (t *testService) TestUnitAuthService_Authenticate_MissingHash() {
	initData := "query_id=test&auth_date=" + strconv.FormatInt(time.Now().Unix(), 10)

	_, err := t.srv.Authenticate(context.Background(), initData)
	t.EqualError(err, "hash parameter is missing")
}

func (t *testService) TestUnitAuthService_Authenticate_MultipleHashes() {
	initData := "query_id=test&auth_date=" + strconv.FormatInt(time.Now().Unix(), 10) + "&hash=abc&hash=def"

	_, err := t.srv.Authenticate(context.Background(), initData)
	t.Error(err)
}

func (t *testService) TestUnitAuthService_Authenticate_InvalidUserJSON() {
	values := url.Values{
		"query_id":  {"test-query-id"},
		"user":      {"not-json"},
		"auth_date": {strconv.FormatInt(time.Now().Unix(), 10)},
	}

	hash := t.computeHash(values)
	values.Set("hash", hash)

	_, err := t.srv.Authenticate(context.Background(), values.Encode())
	t.Error(err)
}

func (t *testService) TestUnitAuthService_Authenticate_URLEncodedInitData() {
	values := url.Values{
		"query_id":  {"8e35e478-6a54-4e97-b1ef-06aa07688ae1"},
		"user":      {`{"id":12345,"first_name":"Kirill Test","last_name":"","username":null,"language_code":"ru","photo_url":"https://example.com/i?r=abc123"}`},
		"chat":      {`{"id":7927362,"type":"DIALOG"}`},
		"auth_date": {strconv.FormatInt(time.Now().Unix(), 10)},
		"ip":        {"192.168.1.1"},
	}

	ctx := context.Background()
	hash := t.computeHash(values)
	values.Set("hash", hash)

	t.repo.EXPECT().StoreRefresh(ctx, gomock.Any()).Return(nil)
	t.repo.EXPECT().GetRefresh(ctx, testUserId).Return(domain.AuthData{User: domain.AuthUser{ID: testUserId}}, nil)
	t.repo.EXPECT().DeleteRefresh(ctx, testUserId).Return(nil)

	data, err := t.srv.Authenticate(ctx, values.Encode())
	t.NoError(err)

	t.Equal(testUserId, data.User.ID)
	t.Require().NotEmpty(t, data.TokenPair.AccessToken)
	t.Require().NotEmpty(t, data.TokenPair.RefreshToken)
}

func (t *testService) TestUnitAuthService_Refresh_Success() {
	initData := t.buildInitData(time.Now().Unix())
	ctx := context.Background()

	t.repo.EXPECT().StoreRefresh(ctx, gomock.Any()).Return(nil).Times(2)
	t.repo.EXPECT().GetRefresh(ctx, testUserId).Return(domain.AuthData{}, nil)

	data, err := t.srv.Authenticate(ctx, initData)
	t.NoError(err)

	t.repo.EXPECT().GetRefresh(ctx, testUserId).Return(data, nil)
	t.repo.EXPECT().DeleteRefresh(ctx, testUserId).Return(nil)

	// Refresh
	newPair, err := t.srv.Refresh(ctx, data.TokenPair.RefreshToken)
	t.NoError(err)

	t.NotEmpty(newPair.TokenPair.AccessToken)
	t.NotEmpty(newPair.TokenPair.RefreshToken)
}

func (t *testService) TestUnitAuthService_Refresh_InvalidToken() {
	_, err := t.srv.Refresh(context.Background(), "invalid-token")
	t.Error(err)
}

func (t *testService) TestUnitAuthService_Refresh_EmptyTokenParts() {
	_, err := t.srv.Refresh(context.Background(), "12345::")
	t.Error(err)
}

func (t *testService) TestUnitAuthService_Refresh_DeletedToken() {
	initData := t.buildInitData(time.Now().Unix())
	ctx := context.Background()

	t.repo.EXPECT().StoreRefresh(ctx, gomock.Any()).Return(nil)
	t.repo.EXPECT().DeleteRefresh(ctx, testUserId).Return(nil)
	t.repo.EXPECT().GetRefresh(ctx, testUserId).Return(domain.AuthData{}, nil)

	data, err := t.srv.Authenticate(ctx, initData)
	t.NoError(err)

	// Logout
	err = t.srv.Logout(ctx, data.TokenPair.RefreshToken)
	t.Require().NoError(err)

	t.repo.EXPECT().GetRefresh(ctx, testUserId).Return(domain.AuthData{}, errors.New("not found"))

	// Refresh после logout должен быть ошибкой
	_, err = t.srv.Refresh(ctx, data.TokenPair.RefreshToken)
	t.Error(err)
}

func (t *testService) TestUnitAuthService_Logout_Success() {
	initData := t.buildInitData(time.Now().Unix())
	ctx := context.Background()

	t.repo.EXPECT().StoreRefresh(ctx, gomock.Any()).Return(nil)
	t.repo.EXPECT().DeleteRefresh(ctx, testUserId).Return(nil)
	t.repo.EXPECT().GetRefresh(ctx, testUserId).Return(domain.AuthData{}, nil)

	data, err := t.srv.Authenticate(ctx, initData)
	t.NoError(err)

	err = t.srv.Logout(ctx, data.TokenPair.RefreshToken)
	t.Require().NoError(err)
}

func (t *testService) TestUnitAuthService_Logout_InvalidToken() {
	err := t.srv.Logout(context.Background(), "invalid-token")
	t.Error(err)
}

func (t *testService) TestUnitAuthService_ParseAccessToken_Success() {
	initData := t.buildInitData(time.Now().Unix())
	ctx := context.Background()

	t.repo.EXPECT().StoreRefresh(ctx, gomock.Any()).Return(nil)
	t.repo.EXPECT().GetRefresh(ctx, testUserId).Return(domain.AuthData{}, nil)

	data, err := t.srv.Authenticate(context.Background(), initData)
	t.NoError(err)

	claims, err := t.srv.ParseAndValidateToken(ctx, data.TokenPair.AccessToken, "access")
	t.Require().NoError(err)

	t.Equal(testUserId, claims.User.ID)

}
