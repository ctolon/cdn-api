package service

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	//"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/notification"

	//"github.com/ctolon/cdn-api/internal/config"
	"github.com/ctolon/cdn-api/internal/logger"
)

type MinioService struct {
	Client *minio.Client
	Logger logger.LoggerAdapter
}

func NewMinioService(endpoint string, opts *minio.Options, logger logger.LoggerAdapter) (*MinioService, error) {

	minioClient, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, err
	}
	//logger = adapters.NewLoggerAdapter(adapters.ZerologLogger, nil)

	return &MinioService{Client: minioClient, Logger: logger}, nil
}

/* Bucket API operations */

// CreateBucket creates a new bucket in the Minio server
func (ms *MinioService) CreateBucket(ctx context.Context, bucketName string) error {
	exist, err := ms.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if exist {
		return ms.Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	}
	return nil
}

// RemoveBucket removes a bucket from the Minio server
func (ms *MinioService) RemoveBucket(ctx context.Context, bucketName string) error {
	return ms.Client.RemoveBucket(ctx, bucketName)
}

// ListBuckets lists all buckets in the Minio server
func (ms *MinioService) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return ms.Client.ListBuckets(ctx)
}

// BucketExists checks if a bucket exists in the Minio server
func (ms *MinioService) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	return ms.Client.BucketExists(ctx, bucketName)
}

// ListObjects lists all objects in a bucket
func (ms *MinioService) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	return ms.Client.ListObjects(ctx, bucketName, opts)
}

/* BucketPolicy API operations*/

// GetBucketPolicy gets the bucket policy of a bucket
func (ms *MinioService) GetBucketPolicy(ctx context.Context, bucketName string) (string, error) {
	return ms.Client.GetBucketPolicy(ctx, bucketName)
}

// SetBucketPolicy sets the bucket policy of a bucket
func (ms *MinioService) SetBucketPolicy(ctx context.Context, bucketName, policy string) error {
	return ms.Client.SetBucketPolicy(ctx, bucketName, policy)
}

/* Bucket Notification API operations */

// SetBucketNotification sets the bucket notification of a bucket
func (ms *MinioService) GetBucketNotification(ctx context.Context, bucketName string) (notification.Configuration, error) {
	return ms.Client.GetBucketNotification(ctx, bucketName)
}

// SetBucketNotification sets the bucket notification of a bucket
func (ms *MinioService) SetBucketNotification(ctx context.Context, bucketName string, config notification.Configuration) error {
	return ms.Client.SetBucketNotification(ctx, bucketName, config)
}

// RemoveAllBucketNotification removes the bucket notification of a bucket
func (ms *MinioService) RemoveAllBucketNotification(ctx context.Context, bucketName string) error {
	return ms.Client.RemoveAllBucketNotification(ctx, bucketName)
}

// ListenBucketNotification listens to bucket notification events
func (ms *MinioService) ListenBucketNotification(ctx context.Context, bucketName, prefix, suffix string, events []string) <-chan notification.Info {
	return ms.Client.ListenBucketNotification(ctx, bucketName, prefix, suffix, events)
}

// ListenNotification listens to notification events
func (ms *MinioService) ListenNotification(ctx context.Context, bucketName, objectName string, events []string) <-chan notification.Info {
	return ms.Client.ListenNotification(ctx, bucketName, objectName, events)
}

/* File API operations */

// FPutObject uploads a file to a bucket
func (ms *MinioService) FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return ms.Client.FPutObject(ctx, bucketName, objectName, filePath, opts)
}

// FGetObject downloads a file from a bucket
func (ms *MinioService) FGetObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.GetObjectOptions) error {
	return ms.Client.FGetObject(ctx, bucketName, objectName, filePath, opts)
}

/* Object API operations */

// GetObject gets an object from a bucket
func (ms *MinioService) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	return ms.Client.GetObject(ctx, bucketName, objectName, opts)
}

// PutObject uploads an object to a bucket
func (ms *MinioService) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return ms.Client.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
}

// RemoveObject removes an object from a bucket
func (ms *MinioService) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	return ms.Client.RemoveObject(ctx, bucketName, objectName, opts)
}

// RemoveObjects removes multiple objects from a bucket
func (ms *MinioService) RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan minio.ObjectInfo, opts minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
	return ms.Client.RemoveObjects(ctx, bucketName, objectsCh, opts)
}

// StatObject gets the metadata of an object from a bucket
func (ms *MinioService) StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	return ms.Client.StatObject(ctx, bucketName, objectName, opts)
}

// CopyObject copies an object from a source bucket to a destination bucket
func (ms *MinioService) CopyObject(ctx context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error) {
	return ms.Client.CopyObject(ctx, dst, src)
}

// PutObjectStream uploads an object to a bucket using a reader
func (ms *MinioService) PutObjectStream(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return ms.Client.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
}
