package s3_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"stg/internal/app/clients/s3"
)

func TestS3Client(t *testing.T) {
	suite.Run(t, new(testS3))
}

type testS3 struct {
	suite.Suite

	bucket string
	store  *MockS3
	client *s3.Client
}

func (t *testS3) SetupTest() {
	ctrl := gomock.NewController(t.T())

	t.bucket = "test-bucket"
	t.store = NewMockS3(ctrl)
	t.client = s3.NewClient(zap.NewNop(), t.store, t.bucket)
}

func (t *testS3) TestUploadSuccess() {
	ctx := context.Background()
	r := bytes.NewReader([]byte("hello"))
	opt := minio.PutObjectOptions{ContentType: "text/plain"}

	t.store.EXPECT().PutObject(ctx, t.bucket, "test-key", r, int64(5), opt).
		Return(minio.UploadInfo{}, nil)

	err := t.client.Upload(ctx, "test-key", r, 5, "text/plain")
	t.NoError(err)
}

func (t *testS3) TestUploadError() {
	ctx := context.Background()
	r := bytes.NewReader([]byte("hello"))
	opt := minio.PutObjectOptions{ContentType: "text/plain"}

	t.store.EXPECT().PutObject(ctx, t.bucket, "test-key", r, int64(5), opt).
		Return(minio.UploadInfo{}, errors.New("oops"))

	err := t.client.Upload(ctx, "test-key", r, 5, "text/plain")
	t.EqualError(err, "put object: oops")
}

func (t *testS3) TestDownloadError() {
	ctx := context.Background()

	t.store.EXPECT().GetObject(ctx, t.bucket, "test-key", minio.GetObjectOptions{}).Return(nil, nil)

	_, _, err := t.client.Download(ctx, "test-key")
	t.EqualError(err, "stat object: Object is nil")
}

func (t *testS3) TestDeleteSuccess() {
	ctx := context.Background()

	t.store.EXPECT().RemoveObject(ctx, t.bucket, "test-key", minio.RemoveObjectOptions{}).Return(nil)

	err := t.client.Delete(ctx, "test-key")
	t.NoError(err)
}

func (t *testS3) TestDeleteError() {
	ctx := context.Background()

	t.store.EXPECT().RemoveObject(ctx, t.bucket, "test-key", minio.RemoveObjectOptions{}).
		Return(errors.New("oops"))

	err := t.client.Delete(ctx, "test-key")
	t.EqualError(err, "remove object: oops")
}

func (t *testS3) TestEnsureBucketExistsSuccess() {
	ctx := context.Background()

	t.store.EXPECT().BucketExists(ctx, t.bucket).Return(true, nil)

	err := t.client.EnsureBucket(ctx)
	t.NoError(err)
}

func (t *testS3) TestEnsureBucketNotExistsSuccess() {
	ctx := context.Background()

	t.store.EXPECT().BucketExists(ctx, t.bucket).Return(false, nil)
	t.store.EXPECT().MakeBucket(ctx, t.bucket, minio.MakeBucketOptions{}).Return(nil)

	err := t.client.EnsureBucket(ctx)
	t.NoError(err)
}

func (t *testS3) TestEnsureBucketExistsError() {
	ctx := context.Background()

	t.store.EXPECT().BucketExists(ctx, t.bucket).Return(false, errors.New("oops"))

	err := t.client.EnsureBucket(ctx)
	t.EqualError(err, "check bucket: oops")
}

func (t *testS3) TestEnsureBucketNotNotExistsError() {
	ctx := context.Background()

	t.store.EXPECT().BucketExists(ctx, t.bucket).Return(false, nil)
	t.store.EXPECT().MakeBucket(ctx, t.bucket, minio.MakeBucketOptions{}).Return(errors.New("oops"))

	err := t.client.EnsureBucket(ctx)
	t.EqualError(err, "create bucket: oops")
}
