package minio

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type StorageRepository struct {
	client   *minio.Client
	endpoint string
	log      *zap.Logger
}

func NewStorageRepository(endpoint, accessKey, secretKey string, useSSL bool, log *zap.Logger) (*StorageRepository, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &StorageRepository{
		client:   client,
		endpoint: endpoint,
		log:      log,
	}, nil
}

func (r *StorageRepository) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := r.client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}

	if !exists {
		if err := r.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}

		policy := fmt.Sprintf(`{
			"Version":"2012-10-17",
			"Statement":[{
				"Effect":"Allow",
				"Principal":{"AWS":["*"]},
				"Action":["s3:GetObject"],
				"Resource":["arn:aws:s3:::%s/*"]
			}]
		}`, bucket)

		if err := r.client.SetBucketPolicy(ctx, bucket, policy); err != nil {
			r.log.Warn("failed to set bucket policy", zap.String("bucket", bucket), zap.Error(err))
		}
	}

	return nil
}

func (r *StorageRepository) UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)

	_, err := r.client.PutObject(ctx, bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("http://%s/%s/%s", r.endpoint, bucket, objectName)
	return url, nil
}

func (r *StorageRepository) DeleteFile(ctx context.Context, bucket, objectName string) error {
	return r.client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
}

func (r *StorageRepository) GetPresignedURL(ctx context.Context, bucket, objectName string) (string, error) {
	url, err := r.client.PresignedGetObject(ctx, bucket, objectName, 0, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
