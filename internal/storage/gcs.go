package storage

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
)

type GCSClient struct {
	client     *storage.Client
	bucketName string
}

func NewGCSClient(ctx context.Context, bucketName string) (*GCSClient, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSClient{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// GenerateUploadURL crea un URL firmato per permettere al frontend l'upload diretto
func (g *GCSClient) GenerateUploadURL(objectName string, mimeType string, expiresInMinutes int) (string, error) {
	opts := &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      "PUT",
		Expires:     time.Now().Add(time.Duration(expiresInMinutes) * time.Minute),
		ContentType: mimeType,
	}

	// Firma l'URL usando il bucket configurato
	url, err := g.client.Bucket(g.bucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", fmt.Errorf("storage.SignedURL: %w", err)
	}

	return url, nil
}
