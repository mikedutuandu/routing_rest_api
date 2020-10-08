package services

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"janio-backend/config"
	"janio-backend/helper"
	"mime/multipart"
	"net/http"
)

func UploadS3(fileStream *multipart.FileHeader, pathStr string) string {
	file, err := fileStream.Open()
	if err != nil {
		return ""
	}
	defer file.Close()

	token := ""
	creds := credentials.NewStaticCredentials(config.GetConfig().AWS_ACCESS_KEY_ID, config.GetConfig().AWS_SECRET_ACCESS_KEY, token)
	_, err = creds.Get()
	if err != nil {
		// handle error
	}
	cfg := aws.NewConfig().WithRegion(config.GetConfig().AWS_REGION).WithCredentials(creds)
	svc := s3.New(session.New(), cfg)

	buffer := make([]byte, fileStream.Size) // read file content to buffer

	file.Read(buffer)
	fileBytes := bytes.NewReader(buffer)
	fileType := http.DetectContentType(buffer)
	newName := helper.BuildFileName(fileStream.Filename)
	path := "/" + pathStr + "/" + newName
	params := &s3.PutObjectInput{
		Bucket:        aws.String(config.GetConfig().AWS_BUCKET),
		Key:           aws.String(path),
		Body:          fileBytes,
		ContentLength: aws.Int64(fileStream.Size),
		ContentType:   aws.String(fileType),
	}
	resp, err := svc.PutObject(params)
	if err != nil {
		// handle error
	}

	fmt.Printf("response %s", awsutil.StringValue(resp))
	return newName
}

func GetFileContentType(fileStream *multipart.FileHeader) (string, error) {

	file, err := fileStream.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

func IsCsv(fileStream *multipart.FileHeader) bool {
	contentType, _ := GetFileContentType(fileStream)

	switch contentType {
	case "text/csv":
		return true

	case "text/plain; charset=utf-8":
		return true
	case "text/plain":
		return true

	default:
		return false
	}
}
func IsImage(fileStream *multipart.FileHeader) bool {
	contentType, _ := GetFileContentType(fileStream)

	switch contentType {
	case "image/jpeg", "image/jpg":
		return true

	case "image/png":
		return true

	default:
		return false
	}
}

func SizeAllow(fileStream *multipart.FileHeader, m int) bool {
	size := fileStream.Size / (1024 * 1024)
	if size > int64(m) {
		return false
	}
	return true
}
