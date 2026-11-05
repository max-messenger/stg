package auth

//go:generate go tool mockgen -source=auth.go -destination=auth_mock_test.go -package=auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"

	"stg/internal/app/domain"
	"stg/pkg/http/middlewares"
)

const (
	jwtKeyUserID   = "user_id"
	jwtKeyChatID   = "chat_id"
	jwtKeyUserName = "user_name"
	jwtKeyExpire   = "expire"
	jwtKeyAwait    = "await"
	jwtKeyType     = "token_type"

	jwtTypeAccess  = "access"
	jwtTypeRefresh = "refresh"
)

type claimsData struct {
	TokenType string `json:"token_type"`
	UserName  string `json:"user_name"`
	UserID    int64  `json:"user_id"`
	ChatID    int64  `json:"chat_id"`
}

// {"await":1784290509,"chat_id":0,"expire":1784294109,"type":"refresh","user_id":0,"username":""}

type TokenRepo interface {
	StoreRefresh(ctx context.Context, data domain.AuthData) error
	GetRefresh(ctx context.Context, userID int64) (domain.AuthData, error)
	DeleteRefresh(ctx context.Context, userID int64) error
}

// Service предоставляет авторизацию через валидацию MAX initData.
type Service struct {
	botToken   string
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	tokenRepo  TokenRepo
}

func New(tokenRepo TokenRepo, cfg Config) *Service {
	return &Service{
		botToken:   cfg.MaxToken,
		secret:     []byte(cfg.JwtSecret),
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		tokenRepo:  tokenRepo,
	}
}

func (s *Service) Authenticate(ctx context.Context, initData string) (domain.AuthData, error) {
	data, err := maxbot.ValidateInitData(initData, s.botToken)
	if err != nil {
		return domain.AuthData{}, err
	}

	existing, err := s.tokenRepo.GetRefresh(ctx, data.User.ID)
	if err != nil {
		return domain.AuthData{}, err
	}
	if existing.User.ID != 0 {
		err = s.tokenRepo.DeleteRefresh(ctx, existing.User.ID)
		if err != nil {
			return domain.AuthData{}, err
		}
	}

	access, refresh, err := s.generateTokens(s.initDataToDomain(data))
	if err != nil {
		return domain.AuthData{}, err
	}

	authData := domain.AuthData{
		TokenPair: domain.AuthTokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
		},
		User: domain.AuthUser{
			ID:       data.User.ID,
			ChatID:   data.Chat.Id,
			Username: data.User.FirstName,
		},
	}

	err = s.tokenRepo.StoreRefresh(ctx, authData)
	if err != nil {
		return domain.AuthData{}, err
	}

	return authData, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (domain.AuthData, error) {
	refreshToken = strings.TrimPrefix(refreshToken, "Bearer ")
	claims, err := s.verifyToken(refreshToken)
	if err != nil {
		return domain.AuthData{}, fmt.Errorf("verify refresh token: %w", err)
	}

	claimsRes, err := s.parseClaims(claims)
	if err != nil {
		return domain.AuthData{}, fmt.Errorf("parse claims: %w", err)
	}

	if claimsRes.TokenType != jwtTypeRefresh {
		return domain.AuthData{}, fmt.Errorf("invalid token type, expected refresh")
	}

	storedData, err := s.tokenRepo.GetRefresh(ctx, claimsRes.UserID)
	if err != nil {
		return domain.AuthData{}, fmt.Errorf("refresh token not found: %w", err)
	}

	if storedData.TokenPair.RefreshToken != refreshToken {
		return domain.AuthData{}, fmt.Errorf("refresh token mismatch")
	}

	err = s.tokenRepo.DeleteRefresh(ctx, claimsRes.UserID)
	if err != nil {
		return domain.AuthData{}, fmt.Errorf("delete old refresh: %w", err)
	}

	access, refresh, err := s.generateTokens(storedData)
	if err != nil {
		return domain.AuthData{}, err
	}

	newData := domain.AuthData{
		TokenPair: domain.AuthTokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
		},
		User: domain.AuthUser{
			ID:       claimsRes.UserID,
			ChatID:   claimsRes.ChatID,
			Username: claimsRes.UserName,
		},
	}

	err = s.tokenRepo.StoreRefresh(ctx, newData)
	if err != nil {
		return domain.AuthData{}, err
	}

	return newData, nil
}

func (s *Service) Logout(ctx context.Context, accessToken string) error {
	claims, err := s.verifyToken(accessToken)
	if err != nil {
		return fmt.Errorf("verify refresh token: %w", err)
	}

	claimsRes, err := s.parseClaims(claims)
	if err != nil {
		return fmt.Errorf("parse claims: %w", err)
	}

	if claimsRes.TokenType != jwtTypeRefresh {
		return fmt.Errorf("invalid token type, expected refresh")
	}

	err = s.tokenRepo.DeleteRefresh(ctx, claimsRes.UserID)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}

	return nil
}

func (s *Service) ParseAndValidateToken(ctx context.Context, token string, expectedType string) (domain.AuthData, error) {
	token = strings.TrimPrefix(token, "Bearer ")

	claims, err := s.verifyToken(token)
	if err != nil {
		return domain.AuthData{}, err
	}

	claimsRes, err := s.parseClaims(claims)
	if err != nil {
		return domain.AuthData{}, fmt.Errorf("parse claims: %w", err)
	}

	if claimsRes.TokenType != expectedType {
		return domain.AuthData{}, fmt.Errorf("invalid token type")
	}

	// Для access токенов можем не проверять БД
	// Для refresh - обязательно проверяем
	if expectedType == jwtTypeRefresh {
		data, err := s.tokenRepo.GetRefresh(ctx, claimsRes.UserID)
		if err != nil {
			return domain.AuthData{}, fmt.Errorf("refresh token not found: %w", err)
		}

		// Проверяем, что переданный токен совпадает с сохраненным
		if data.TokenPair.RefreshToken != token {
			return domain.AuthData{}, fmt.Errorf("token mismatch")
		}

		return data, nil
	}

	// Для access токенов возвращаем только user данные
	return domain.AuthData{
		User: domain.AuthUser{
			ID:       claimsRes.UserID,
			ChatID:   claimsRes.ChatID,
			Username: claimsRes.UserName,
		},
	}, nil
}

func (s *Service) ValidateUserID(ctx context.Context, userID int64) error {
	uid := middlewares.GetUserID(ctx)
	if uid != userID {
		return fmt.Errorf("invalid user id")
	}

	return nil
}

func (s *Service) generateTokens(data domain.AuthData) (string, string, error) {
	now := time.Now()

	accessClaims := jwt.MapClaims{
		jwtKeyUserID:   data.User.ID,
		jwtKeyChatID:   data.User.ChatID,
		jwtKeyUserName: data.User.Username,
		jwtKeyExpire:   now.Add(s.accessTTL).Unix(),
		jwtKeyAwait:    now.Unix(),
		jwtKeyType:     jwtTypeAccess,
	}

	access := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := access.SignedString(s.secret)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := jwt.MapClaims{
		jwtKeyUserID:   data.User.ID,
		jwtKeyChatID:   data.User.ChatID,
		jwtKeyUserName: data.User.Username,
		jwtKeyExpire:   now.Add(s.refreshTTL).Unix(),
		jwtKeyAwait:    now.Unix(),
		jwtKeyType:     jwtTypeRefresh,
	}

	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refresh.SignedString(s.secret)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return accessStr, refreshStr, nil
}

func (s *Service) verifyToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	exp, ok := claims[jwtKeyExpire].(float64)
	if ok {
		expTime := time.Unix(int64(exp), 0)
		if time.Now().After(expTime) {
			return nil, fmt.Errorf("token expired")
		}
	}

	return claims, nil
}

func (s *Service) parseClaims(claims jwt.MapClaims) (claimsData, error) {
	var res claimsData

	data, err := json.Marshal(claims)
	if err != nil {
		return res, fmt.Errorf("marshal claims: %w", err)
	}

	err = json.Unmarshal(data, &res)
	if err != nil {
		return res, fmt.Errorf("unmarshal claims: %w", err)
	}

	return res, nil
}

func (s *Service) initDataToDomain(initData model.InitData) domain.AuthData {
	return domain.AuthData{
		User: domain.AuthUser{
			ID:       initData.User.ID,
			ChatID:   initData.Chat.Id,
			Username: initData.User.FirstName,
		},
	}
}
