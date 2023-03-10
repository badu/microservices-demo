package images

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/pkg/config"
)

const (
	imagesBucket = "images"
)

// Init new AWS S3 session
func NewS3Session(cfg *config.Config) *s3.S3 {
	// s3Config := &aws.Config{
	// 	Credentials:      credentials.NewStaticCredentials("minio", "minio", ""),
	// 	Endpoint:         aws.String("http://localhost:9000"),
	// 	Region:           aws.String("us-east-1"),
	// 	DisableSSL:       aws.Bool(true),
	// 	S3ForcePathStyle: aws.Bool(true),
	// }
	//
	// newSession, err := session.NewSession(s3Config)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// s3Client := s3Client.New(newSession)
	return s3.New(session.Must(session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials("minio123", "minio123", ""),
		Region:           aws.String(cfg.AWS.S3Region),
		Endpoint:         aws.String(cfg.AWS.S3EndPoint),
		DisableSSL:       aws.Bool(cfg.AWS.DisableSSL),
		S3ForcePathStyle: aws.Bool(cfg.AWS.S3ForcePathStyle),
	})))

}

type AWSS3Client struct {
	cfg      *config.Config
	s3Client *s3.S3
}

func NewAWSStorage(cfg *config.Config, s3 *s3.S3) AWSS3Client {
	return AWSS3Client{cfg: cfg, s3Client: s3}
}

func (i *AWSS3Client) PutObject(ctx context.Context, data []byte, fileType string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "aws_s3_client.PutObject")
	defer span.Finish()

	newFilename := uuid.NewV4().String()
	key := i.getFileKey(newFilename, fileType)

	object, err := i.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: aws.String(imagesBucket),
		Key:    aws.String(key),
		ACL:    aws.String(s3.BucketCannedACLPublicRead),
	})
	if err != nil {
		return "", errors.Join(err, errors.New("aws s3 client saving object to bucket"))
	}

	log.Printf("object : %-v", object)

	return i.getFilePublicURL(key), err
}

func (i *AWSS3Client) GetObject(ctx context.Context, key string) (*s3.GetObjectOutput, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "aws_s3_client.GetObject")
	defer span.Finish()

	obj, err := i.s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(imagesBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.Join(err, errors.New("aws s3 client getting object from bucket"))
	}

	return obj, nil
}

func (i *AWSS3Client) DeleteObject(ctx context.Context, key string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "aws_s3_client.DeleteObject")
	defer span.Finish()

	_, err := i.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(imagesBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return errors.Join(err, errors.New("aws s3 client deleting object from bucket"))
	}

	return nil
}

func (i *AWSS3Client) getFileKey(fileID string, fileType string) string {
	return fmt.Sprintf("%s.%s", fileID, fileType)
}

func (i *AWSS3Client) getFilePublicURL(key string) string {
	return i.cfg.AWS.S3EndPointMinio + "/" + imagesBucket + "/" + key
}
