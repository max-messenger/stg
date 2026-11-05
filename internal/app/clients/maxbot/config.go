//nolint:revive
package maxbot

type Config struct {
	TestApiUrl       string   `yaml:"test_api_url"`
	Token            string   `yaml:"token"`
	Url              string   `yaml:"url"`
	Path             string   `yaml:"path"`
	Secret           string   `yaml:"secret"` // nolint:gosec
	SubscriptionType string   `yaml:"subscription_type"`
	Types            []string `yaml:"types"`
}

func (c Config) isTest() bool {
	return c.TestApiUrl != ""
}
