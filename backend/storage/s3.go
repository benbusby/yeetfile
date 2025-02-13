package storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go/endpoints"
	"io"
	"log"
	"net/url"
	"strings"
	"yeetfile/backend/db"
	"yeetfile/backend/utils"
)

const (
	testFileName    = "test-connection"
	testFileContent = "test"
)

type S3 struct {
	client      *s3.Client
	endpoint    string
	accessKeyID string
	secretKey   string
	bucketName  string
	regionName  string
}

type S3Resolver struct {
	Endpoint string
	Region   string
}

func (r *S3Resolver) ResolveEndpoint(_ context.Context, params s3.EndpointParameters) (smithy.Endpoint, error) {
	endpoint := r.Endpoint
	if strings.HasSuffix(endpoint, "/") {
		endpoint = strings.TrimSuffix(endpoint, "/")
	}

	fullPath := fmt.Sprintf("%s/%s", endpoint, *params.Bucket)
	uri, err := url.ParseRequestURI(fullPath)
	if err != nil {
		return smithy.Endpoint{}, err
	}

	return smithy.Endpoint{
		URI: *uri,
	}, nil
}

func (s3Backend *S3) Authorize() error {
	log.Println("Authorizing S3 backend...")
	credsProvider := credentials.NewStaticCredentialsProvider(
		s3Backend.accessKeyID,
		s3Backend.secretKey,
		"")
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(s3Backend.regionName),
		config.WithCredentialsProvider(credsProvider))

	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.EndpointResolverV2 = &S3Resolver{
			Endpoint: s3Backend.endpoint,
			Region:   s3Backend.regionName,
		}
		o.UsePathStyle = true
		o.Region = s3Backend.regionName
	})

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s3Backend.bucketName),
		Key:    aws.String(testFileName),
		Body:   bytes.NewReader([]byte(testFileContent)),
	})

	if err != nil {
		return err
	}

	s3Backend.client = client
	return nil
}

func (s3Backend *S3) Reauthorize() {}

func (s3Backend *S3) InitUpload(_ string) error {
	// No initialization needed for single-chunk uploads
	return nil
}

func (s3Backend *S3) InitLargeUpload(filename, metadataID string) error {
	input := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(s3Backend.bucketName),
		Key:    aws.String(filename),
	}

	output, err := s3Backend.client.CreateMultipartUpload(context.TODO(), input)
	if err != nil {
		log.Printf("Error initiating multipart upload: %v\n", err)
		return err
	}

	return db.UpdateUploadValues(
		metadataID,
		"",
		"",
		*output.UploadId,
		false)
}

func (s3Backend *S3) UploadSingleChunk(chunk FileChunk, _ db.Upload) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s3Backend.bucketName),
		Key:         aws.String(chunk.Filename),
		Body:        bytes.NewReader(chunk.Data),
		ContentType: aws.String("application/octet-stream"),
	}

	_, err := s3Backend.client.PutObject(context.TODO(), input)
	if err != nil {
		log.Printf("Failed to upload chunk: %v\n", err)
		return err
	}

	err = db.UpdateMetadata(
		chunk.FileID,
		"",
		int64(len(chunk.Data)))

	return err
}

func (s3Backend *S3) UploadMultiChunk(chunk FileChunk, upload db.Upload) (bool, error) {
	ctx := context.TODO()
	uploadInput := &s3.UploadPartInput{
		Bucket:        aws.String(s3Backend.bucketName),
		Key:           aws.String(chunk.Filename),
		UploadId:      aws.String(upload.UploadID),
		PartNumber:    aws.Int32(int32(chunk.ChunkNum)),
		Body:          bytes.NewReader(chunk.Data),
		ContentLength: aws.Int64(int64(len(chunk.Data))),
	}

	uploadOutput, err := s3Backend.client.UploadPart(ctx, uploadInput)
	if err != nil {
		log.Printf("Failed to upload file chunk: %v\n", err)
		return false, err
	}

	checksums, err := db.UpdateChecksums(chunk.FileID, chunk.ChunkNum, *uploadOutput.ETag)
	if err != nil {
		log.Printf("Failed to update S3 ETags: %v\n", err)
		return false, err
	}

	if len(checksums) == chunk.TotalChunks {
		var size int64
		_, size, err = s3Backend.FinishLargeUpload(
			upload.UploadID,
			chunk.Filename,
			checksums)

		if err != nil {
			log.Printf("Failed to finalize multipart upload: %v\n", err)
			return false, err
		}

		return true, db.UpdateMetadata(upload.MetadataID, upload.UploadID, size)
	}

	return false, nil
}

func (s3Backend *S3) CancelLargeFile(remoteID, filename string) (bool, error) {
	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s3Backend.bucketName),
		Key:      aws.String(filename),
		UploadId: aws.String(remoteID),
	}

	_, err := s3Backend.client.AbortMultipartUpload(context.TODO(), input)
	return true, err
}

func (s3Backend *S3) DeleteFile(_, filename string) (bool, error) {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s3Backend.bucketName),
		Key:    aws.String(filename),
	}

	_, err := s3Backend.client.DeleteObject(context.TODO(), input)
	if err != nil {
		log.Printf("Failed to delete file: %v\n", err)
		return false, err
	}

	return true, nil
}

func (s3Backend *S3) FinishLargeUpload(remoteID, filename string, checksums []string) (string, int64, error) {
	var completedParts []types.CompletedPart
	ctx := context.TODO()
	for i, checksum := range checksums {
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       aws.String(checksum),
			PartNumber: aws.Int32(int32(i + 1)),
		})
	}

	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s3Backend.bucketName),
		Key:      aws.String(filename),
		UploadId: aws.String(remoteID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}

	_, err := s3Backend.client.CompleteMultipartUpload(ctx, completeInput)
	if err != nil {
		log.Printf("Failed to finalize multipart upload: %v\n", err)
		return "", 0, err
	}

	headOutput, err := s3Backend.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s3Backend.bucketName),
		Key:    aws.String(filename),
	})

	if err != nil {
		return "", 0, err
	}

	return "", *headOutput.ContentLength, nil
}

func (s3Backend *S3) PartialDownloadById(_, name string, start, end int64) ([]byte, error) {
	ctx := context.TODO()

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	input := &s3.GetObjectInput{
		Bucket: aws.String(s3Backend.bucketName),
		Key:    aws.String(name),
		Range:  aws.String(rangeHeader),
	}

	output, err := s3Backend.client.GetObject(ctx, input)
	if err != nil {
		log.Printf("Error fetching object bytes: %v\n", err)
		return nil, err
	}

	defer output.Body.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, output.Body)
	if err != nil {
		log.Printf("Failed to copy over bytes from response: %v\n", err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func initS3() storage {
	var (
		endpoint    = utils.GetEnvVar("YEETFILE_S3_ENDPOINT", "")
		accessKeyID = utils.GetEnvVar("YEETFILE_S3_ACCESS_KEY_ID", "")
		secretKey   = utils.GetEnvVar("YEETFILE_S3_SECRET_KEY", "")
		bucketName  = utils.GetEnvVar("YEETFILE_S3_BUCKET_NAME", "")
		regionName  = utils.GetEnvVar("YEETFILE_S3_REGION_NAME", "")
	)

	if utils.IsAnyStringMissing(endpoint, accessKeyID, secretKey, bucketName) {
		log.Fatalf("Missing a required S3 environment variable. Must set:\n" +
			"- YEETFILE_S3_ENDPOINT\n" +
			"- YEETFILE_S3_ACCESS_KEY_ID\n" +
			"- YEETFILE_S3_SECRET_KEY\n" +
			"- YEETFILE_S3_BUCKET_NAME\n")
	}

	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}

	s3Backend := &S3{
		endpoint:    endpoint,
		accessKeyID: accessKeyID,
		secretKey:   secretKey,
		bucketName:  bucketName,
		regionName:  regionName,
	}

	err := s3Backend.Authorize()
	if err != nil {
		log.Println("Unable to authorize S3 backend")
		log.Fatal(err)
	}

	return s3Backend
}
