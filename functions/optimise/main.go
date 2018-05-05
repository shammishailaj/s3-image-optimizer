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

func handler(ctx context.Context, s3Event events.S3Event) {
	var Bucket = os.Getenv("BUCKET")
	var FromBucket = os.Getenv("FROM_BUCKET")
	var ToBucket = os.Getenv("TO_BUCKET")
	JpegQuality, _ := strconv.Atoi(os.Getenv("JPEG_QUALITY"))
	PngQuality, _ := strconv.Atoi(os.Getenv("PNG_QUALITY"))

	var originalKey = s3Event.Records[0].S3.Object.Key
	destinationKey := strings.Replace(originalKey, FromBucket, ToBucket, 1)
	var fileExtension = filepath.Ext(originalKey)

	var tmpImageDownload = "/tmp/optimised-image-download" + fileExtension
	var tmpImageUpload = "/tmp/optimised-image-upload" + fileExtension

	fmt.Printf("======================================\n")
	fmt.Printf("Received Image: %v\n", originalKey)
	fmt.Printf("Delivering Image To: %v\n", destinationKey)

	sess := session.Must(session.NewSession())

	downloader := s3manager.NewDownloader(sess)

	// Create a file to write the S3 file to.
	downloadedFromS3, err := os.Create(tmpImageDownload)
	if err != nil {
		fmt.Printf("failed to create file %q, %v", tmpImageDownload, err)
		return
	}

	// Download the file that triggered the function from S3.
	_, err = downloader.Download(downloadedFromS3, &s3.GetObjectInput{
		Bucket: aws.String("honest.jobs"),
		Key:    aws.String(originalKey),
	})
	if err != nil {
		fmt.Printf("failed to download file, %v", err)
		return
	}

	// Convert the file into an Image
	encodedImageDownloadedFromS3, _, err := image.Decode(downloadedFromS3)
	if err != nil {
		fmt.Printf("Failed to decode image")
		return
	}

	finalImage, err := os.OpenFile(tmpImageUpload, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		fmt.Printf("Error Opening tmpImageUpload File: %s", err)
		return
	}

	// switch on extension jpg/jpeg or png
	switch fileExtension {
	case ".jpeg":
	case ".jpg":
		err = jpeg.Encode(finalImage, encodedImageDownloadedFromS3, &jpeg.Options{Quality: JpegQuality})
		if err != nil {
			fmt.Printf("Failed to encode image: %v", err)
			return
		}

	case ".png":
		pngEncoder := &png.Encoder{
			CompressionLevel: PngQuality,
		}
		err = png.Encode(finalImage, encodedImageDownloadedFromS3)
		if err != nil {
			fmt.Printf("Failed to encode, %v", err)
			return
		}

	}

	// Upload the compressed file to S3
	//======================================
	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(tmpImageUpload)
	if err != nil {
		fmt.Printf("failed to open file %q, %v", tmpImageUpload, err)
		return
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(Bucket),
		Key:    aws.String(destinationKey),
		Body:   f,
	})
	if err != nil {
		fmt.Printf("FAILED UPLOAD: %v", err)
		return
	}

	fmt.Printf("SUCCESS UPLOAD: %s\n", result.Location)
	fmt.Printf("======================================\n")
}

func main() {
	lambda.Start(handler)
}
