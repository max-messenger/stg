package tokenrepo

import "time"

// Config конфигурация tokenrepo.
type Config struct {
	RefreshTTL time.Duration `yaml:"refresh_ttl"`
}
