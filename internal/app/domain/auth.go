package domain

type AuthData struct {
	TokenPair AuthTokenPair
	User      AuthUser
}

type AuthTokenPair struct {
	AccessToken  string
	RefreshToken string
}

// AuthUser содержит данные пользователя и чата для ответа.
type AuthUser struct {
	ID       int64
	ChatID   int64
	Username string
}
