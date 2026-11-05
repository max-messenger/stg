package auth

import "time"

type Config struct {
	JwtSecret  string        `yaml:"jwt_secret"`
	AccessTTL  time.Duration `yaml:"access_ttl"`
	RefreshTTL time.Duration `yaml:"refresh_ttl"`
	MaxToken   string        `yaml:"max_token"`
}
