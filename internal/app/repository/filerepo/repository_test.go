package filerepo_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"stg/internal/app/domain"
	"stg/internal/app/repository/filerepo"
)

func TestFileRepository(t *testing.T) {
	suite.Run(t, new(testRepository))
}

type testRepository struct {
	suite.Suite

	store *MockStore
	repo  *filerepo.Repository
}

func (t *testRepository) SetupTest() {
	ctrl := gomock.NewController(t.T())
	t.store = NewMockStore(ctrl)
	t.repo = filerepo.NewRepository(t.store)
}

func (t *testRepository) TestPutSuccess() {
	ctx := context.Background()
	r := bytes.NewReader([]byte("hello"))

	t.store.EXPECT().Upload(ctx, "test-key", r, int64(5), "text/plain").Return(nil)

	t.NoError(t.repo.Put(ctx, "test-key", r, 5, "text/plain"))
}

func (t *testRepository) TestPutError() {
	ctx := context.Background()
	r := bytes.NewReader([]byte("hello"))

	t.store.EXPECT().Upload(ctx, "test-key", r, int64(5), "text/plain").Return(errors.New("wtf"))

	err := t.repo.Put(ctx, "test-key", r, 5, "text/plain")
	t.EqualError(err, "upload file: wtf")
}

func (t *testRepository) TestGetSuccess() {
	ctx := context.Background()

	stringReader := strings.NewReader("hello")
	stringReadCloser := io.NopCloser(stringReader)

	meta := domain.ObjectMeta{
		ContentType: "text/plain",
		Size:        53,
	}

	expect := domain.FileDownload{
		ContentType: "text/plain",
		Size:        53,
		Reader:      stringReadCloser,
	}

	t.store.EXPECT().Download(ctx, "test-key").Return(stringReadCloser, meta, nil)

	res, err := t.repo.Get(ctx, "test-key")
	t.NoError(err)
	t.Equal(expect, res)
}

func (t *testRepository) TestGetError() {
	ctx := context.Background()

	t.store.EXPECT().Download(ctx, "test-key").Return(nil, domain.ObjectMeta{}, errors.New("not found"))

	_, err := t.repo.Get(ctx, "test-key")
	t.EqualError(err, "download file: not found")
}

func (t *testRepository) TestRemoveSuccess() {
	ctx := context.Background()

	t.store.EXPECT().Delete(ctx, "test-key").Return(nil)
	t.NoError(t.repo.Remove(ctx, "test-key"))
}

func (t *testRepository) TestRemoveError() {
	ctx := context.Background()

	t.store.EXPECT().Delete(ctx, "test-key").Return(errors.New("not found"))

	err := t.repo.Remove(ctx, "test-key")
	t.EqualError(err, "delete file: not found")
}
