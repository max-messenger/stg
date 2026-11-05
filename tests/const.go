//nolint:all
package tests

const (
	ImagePostgres = "postgres:16.6-alpine3.20"
	ImageRedis    = "redis:7.4-alpine"
	ImageRedPanda = "redpandadata/redpanda"
	ImageMinio    = "minio/minio"

	PathMigrationDir = "../../db/migrations"
	PathConfigFile   = "../../config/stg/config.runner.yaml"

	EnvIntegrationStubAddr    = "INTEGRATION_STUB_ADDR"
	EnvMaxBotToken            = "MAXBOT_TOKEN"
	EnvAppHttpPort            = "INTEGRATION_HTTP_ADDR"
	EnvIntegrationS3Endpoint  = "INTEGRATION_S3_ENDPOINT"
	EnvIntegrationS3AccessKey = "INTEGRATION_S3_ACCESS_KEY"
	EnvIntegrationS3SecretKey = "INTEGRATION_S3_SECRET_KEY"
	EnvIntegrationRedisAddr   = "INTEGRATION_REDIS_ADDR"
	EnvIntegrationPgxDns      = "INTEGRATION_POSTGRES_DSN"
	EnvIntegrationKafkaAddr   = "INTEGRATION_KAFKA_ADDR"
)
