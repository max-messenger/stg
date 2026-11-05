package integrations_filerepo

import (
	"bytes"
	"context"
	"io"
	"testing"

	steron "github.com/FluorescentTouch/testosteron/v2"
	mn "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"stg/internal/app/clients/s3"
	"stg/internal/app/repository/filerepo"
	"stg/pkg/config"
	"stg/pkg/connections/minio"
	"stg/tests"
)

func TestFileRepo(t *testing.T) {
	suite.Run(t, new(testRepo))
}

type testRepo struct {
	suite.Suite

	bucket   string
	repo     *filerepo.Repository
	s3Client mn.Client
}

func (t *testRepo) SetupTest() {
	cfgFunc := config.NewConfig("test", tests.PathConfigFile)
	cfg, err := cfgFunc()
	t.NoError(err)

	minioCfg := minio.Config{}
	t.NoError(cfg.Provider.Get("minio").Populate(&minioCfg))

	mnClient, err := minio.NewClient(minioCfg)
	t.NoError(err)

	t.s3Client = steron.Minio().Client(t.T()).Client()
	t.bucket = "stg-integration-test"
	log := zap.NewNop()

	s3Client := s3.NewClient(log, mnClient, t.bucket)
	t.NoError(s3Client.EnsureBucket(context.Background()))

	t.repo = filerepo.NewRepository(s3Client)
}

func (t *testRepo) removeKey(key string) {
	t.NoError(t.s3Client.RemoveObject(context.Background(), t.bucket, key, mn.RemoveObjectOptions{}))
}

func (t *testRepo) TestPutAndGet() {
	key := "test-put-get.txt"
	content := []byte("hello from integration")
	defer t.removeKey(key)

	err := t.repo.Put(context.Background(), key, bytes.NewReader(content), int64(len(content)), "text/plain")
	t.NoError(err)

	download, err := t.repo.Get(context.Background(), key)
	t.NoError(err)
	defer func() { _ = download.Reader.Close() }()

	got, err := io.ReadAll(download.Reader)
	t.NoError(err)
	t.Equal(content, got)
	t.Equal("text/plain", download.ContentType)
}

func (t *testRepo) TestGetNotFound() {
	_, err := t.repo.Get(context.Background(), "nonexistent-key")
	t.Error(err)
}

func (t *testRepo) TestRemove() {
	key := "test-remove.txt"
	content := []byte("temporary")
	defer t.removeKey(key)

	err := t.repo.Put(context.Background(), key, bytes.NewReader(content), int64(len(content)), "text/plain")
	t.NoError(err)

	err = t.repo.Remove(context.Background(), key)
	t.NoError(err)

	_, err = t.repo.Get(context.Background(), key)
	t.Error(err)
}

func (t *testRepo) TestPutOverwrite() {
	key := "test-overwrite.txt"
	defer t.removeKey(key)

	err := t.repo.Put(context.Background(), key, bytes.NewReader([]byte("first")), 5, "text/plain")
	t.NoError(err)

	err = t.repo.Put(context.Background(), key, bytes.NewReader([]byte("second")), 6, "text/plain")
	t.NoError(err)

	download, err := t.repo.Get(context.Background(), key)
	t.NoError(err)
	defer func() { _ = download.Reader.Close() }()

	got, err := io.ReadAll(download.Reader)
	t.NoError(err)
	t.Equal([]byte("second"), got)
}
