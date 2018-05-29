package main

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// SupportedFileTypes - The file extensions we accept and can process.
var SupportedFileTypes = map[string]string{
	".jpeg": "",
	".jpg":  "",
	".png":  "",
}

func s3EventHandler(ctx context.Context, s3Event events.S3Event) (string, error) {
	bucket := os.Getenv("BUCKET")
	fromBucket := os.Getenv("FROM_BUCKET")
	toBucket := os.Getenv("TO_BUCKET")
	JPEGQuality, err := strconv.Atoi(os.Getenv("JPEG_QUALITY"))
	if err != nil {
		panic(err)
	}
	PNGQuality, err := strconv.Atoi(os.Getenv("PNG_QUALITY"))
	if err != nil {
		panic(err)
	}

	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	downloader := s3manager.NewDownloader(sess)
	uploader := s3manager.NewUploader(sess)

	originalKey := s3Event.Records[0].S3.Object.Key
	destinationKey := strings.Replace(originalKey, fromBucket, toBucket, 1)
	fileExtension := filepath.Ext(originalKey)

	// Check that the file type uploaded can be optimised, otherwise we just move it and delete the original.
	if _, ok := SupportedFileTypes[fileExtension]; !ok {
		fmt.Println("The file type: " + fileExtension + " is not supported. Moving instead of optimising image.")
		_, err := svc.CopyObject(&s3.CopyObjectInput{
			Bucket:     aws.String(bucket),
			CopySource: aws.String(bucket + "/" + originalKey),
			Key:        aws.String(destinationKey),
		})

		if err != nil {
			return "", fmt.Errorf("Failed copying unsupported file: %v", err)
		}

		return originalKey, nil
	}

	var tmpImageDownload = "/tmp/optimized-image-download" + fileExtension
	var tmpImageUpload = "/tmp/optimized-image-upload" + fileExtension

	fmt.Printf("======================================\n")
	fmt.Printf("Received Image: %v\n", originalKey)
	fmt.Printf("Delivering Image To: %v\n", destinationKey)

	// Create a temporary file to write the S3 file to.
	downloadedFromS3, err := os.Create(tmpImageDownload)
	if err != nil {
		return "", fmt.Errorf("failed to create file %q, %v", tmpImageDownload, err)
	}

	// Download the file that triggered the function from S3.
	_, err = downloader.Download(downloadedFromS3, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(originalKey),
	})
	if err != nil {
		return "", fmt.Errorf("failed to download file, %v", err)
	}

	// Convert the file into an Image
	encodedImageDownloadedFromS3, _, err := image.Decode(downloadedFromS3)
	if err != nil {
		return "", fmt.Errorf("Failed to decode image: %v", err)
	}

	finalImage, err := os.OpenFile(tmpImageUpload, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return "", fmt.Errorf("Error Opening tmpImageUpload File: %s", err)
	}

	// Compress the image based off of the extension.
	switch fileExtension {
	case ".jpeg", ".jpg":
		err = jpeg.Encode(finalImage, encodedImageDownloadedFromS3, &jpeg.Options{Quality: JPEGQuality})
		if err != nil {
			return "", fmt.Errorf("Failed to encode jpeg: %v", err)
		}

	case ".png":
		var Enc png.Encoder
		Enc.CompressionLevel = png.CompressionLevel(PNGQuality)
		err = png.Encode(finalImage, encodedImageDownloadedFromS3)
		if err != nil {
			return "", fmt.Errorf("Failed to encode png: %v", err)
		}
	}

	// Upload the compressed file to S3
	f, err := os.Open(tmpImageUpload)
	if err != nil {
		return "", fmt.Errorf("failed to open file %q, %v", tmpImageUpload, err)
	}

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(destinationKey),
		Body:   f,
	})
	if err != nil {
		return "", fmt.Errorf("FAILED UPLOAD: %v\nBucket: %s | Key: %s", err, bucket, destinationKey)
	}

	return originalKey, nil
}

func handler(ctx context.Context, s3Event events.S3Event) {
	originalKey, err := s3EventHandler(ctx, s3Event)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("SUCCESS UPLOAD")

	svc := s3.New(session.New())

	bucket := os.Getenv("BUCKET")

	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(originalKey),
	})
	if err != nil {
		fmt.Printf("ERROR DELETING: %v\n", err)
	}
}

func main() {
	lambda.Start(handler)
}
