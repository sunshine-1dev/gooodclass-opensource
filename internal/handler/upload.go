package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *minio.Client
var minioBucket string

func InitMinio() error {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}
	minioBucket = os.Getenv("MINIO_BUCKET")
	if minioBucket == "" {
		minioBucket = "gclass"
	}
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("init minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, minioBucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
		policy := fmt.Sprintf(`{
			"Version":"2012-10-17",
			"Statement":[{
				"Effect":"Allow",
				"Principal":{"AWS":["*"]},
				"Action":["s3:GetObject"],
				"Resource":["arn:aws:s3:::%s/*"]
			}]
		}`, minioBucket)
		if err := client.SetBucketPolicy(ctx, minioBucket, policy); err != nil {
			return fmt.Errorf("set bucket policy: %w", err)
		}
	}

	minioClient = client
	return nil
}

func UploadHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if minioClient == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "upload service unavailable"})
			return
		}

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
			return
		}
		defer file.Close()

		if header.Size > 10<<20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 10MB)"})
			return
		}

		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
		if !allowed[ext] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
			return
		}

		objectName := fmt.Sprintf("uploads/%s/%s%s", time.Now().Format("2006/01"), uuid.New().String(), ext)

		contentType := "application/octet-stream"
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".webp":
			contentType = "image/webp"
		case ".gif":
			contentType = "image/gif"
		}

		_, err = minioClient.PutObject(c.Request.Context(), minioBucket, objectName, file, header.Size, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
			return
		}

		url := fmt.Sprintf("/api/image/%s", objectName)

		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

func ImageProxyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if minioClient == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "image service unavailable"})
			return
		}

		objectName := c.Param("path")
		if objectName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing path"})
			return
		}
		if objectName[0] == '/' {
			objectName = objectName[1:]
		}

		obj, err := minioClient.GetObject(c.Request.Context(), minioBucket, objectName, minio.GetObjectOptions{})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		defer obj.Close()

		info, err := obj.Stat()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.Header("Cache-Control", "public, max-age=31536000")
		c.DataFromReader(http.StatusOK, info.Size, info.ContentType, obj, nil)
	}
}
