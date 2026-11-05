package filestore_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"stg/internal/app/domain"
	"stg/internal/app/services/filestore"
)

func TestUnitFilestoreService(t *testing.T) {
	suite.Run(t, new(testService))
}

type testService struct {
	suite.Suite

	service    *filestore.Service
	repository *MockRepository
}

func (t *testService) SetupTest() {
	ctrl := gomock.NewController(t.T())

	t.repository = NewMockRepository(ctrl)
	t.service = filestore.NewService(t.repository)
}

func (t *testService) TestUploadSuccess() {
	ctx := context.Background()
	content := bytes.NewReader([]byte("hello"))

	t.repository.EXPECT().
		Put(ctx, gomock.Any(), gomock.Any(), int64(5), "text/plain").
		Return(nil)

	result, err := t.service.Upload(ctx, "test.txt", content, 5, "text/plain")
	t.NoError(err)
	assert.True(t.T(), strings.HasPrefix(result.Key, "test-"))
	assert.True(t.T(), strings.HasSuffix(result.Key, ".txt"))
}

func (t *testService) TestUploadEmptyFilename() {
	ctx := context.Background()
	content := bytes.NewReader([]byte("hello"))

	t.repository.EXPECT().
		Put(ctx, gomock.Any(), gomock.Any(), int64(5), "application/octet-stream").
		Return(nil)

	result, err := t.service.Upload(ctx, ".txt", content, 5, "application/octet-stream")
	t.NoError(err)
	assert.True(t.T(), strings.HasPrefix(result.Key, "file-"))
	assert.True(t.T(), strings.HasSuffix(result.Key, ".txt"))
}

func (t *testService) TestDownloadSuccess() {
	ctx := context.Background()
	expected := domain.FileDownload{
		ContentType: "text/plain",
		Size:        5,
	}

	t.repository.EXPECT().
		Get(ctx, "some-key").
		Return(expected, nil)

	result, err := t.service.Download(ctx, "some-key")
	t.NoError(err)
	t.Equal(expected, result)
}

func (t *testService) TestDeleteSuccess() {
	ctx := context.Background()

	t.repository.EXPECT().
		Remove(ctx, "some-key").
		Return(nil)

	err := t.service.Delete(ctx, "some-key")
	t.NoError(err)
}
