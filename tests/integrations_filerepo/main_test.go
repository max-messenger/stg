package integrations_filerepo

import (
	"os"
	"testing"

	"stg/tests"

	steron "github.com/FluorescentTouch/testosteron/v2"
)

func TestMain(m *testing.M) {
	// init package with required options
	options := []steron.Option{
		steron.NewMinioService(tests.ImageMinio),
	}
	cfg, err := steron.Init(options...)
	if err != nil {
		panic(err)
	}

	_ = os.Setenv(tests.EnvIntegrationS3Endpoint, cfg.MinioConfig().Host)
	_ = os.Setenv(tests.EnvIntegrationS3AccessKey, cfg.MinioConfig().User)
	_ = os.Setenv(tests.EnvIntegrationS3SecretKey, cfg.MinioConfig().Password)

	// run the app
	code := m.Run()

	steron.Cleanup()
	os.Exit(code)
}
