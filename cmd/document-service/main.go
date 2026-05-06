package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/DaniFX/ssg-nexus-document-service/internal/api"
	"github.com/DaniFX/ssg-nexus-document-service/internal/storage"
	"github.com/DaniFX/ssg-nexus-sdk/pkg/nexus"
	"github.com/DaniFX/ssg-nexus-sdk/pkg/nexus/repository"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT_ID")
	bucketName := os.Getenv("FIREBASE_STORAGE_BUCKET")

	// 1. Firestore
	fsClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Firestore init error: %v", err)
	}
	defer fsClient.Close()
	docRepo := repository.NewRepository(fsClient, "documents")

	// 2. GCS
	gcsClient, err := storage.NewGCSClient(ctx, bucketName)
	if err != nil {
		log.Fatalf("GCS client error: %v", err)
	}

	// 3. Handler
	docHandler := api.NewDocumentHandler(docRepo, gcsClient)

	router := gin.Default()

	// 4. Service Discovery — completo
	serviceDef := nexus.ServiceDefinition{
		ServiceName: "document-service",
		Version:     "1.1.0",
		Endpoints: []nexus.Endpoint{
			{Path: "/api/v1/documents",                    Method: "GET",    AuthRequired: true, Summary: "Lista documenti (Navigator)"},
			{Path: "/api/v1/documents/:id",               Method: "GET",    AuthRequired: true, Summary: "Dettaglio documento"},
			{Path: "/api/v1/documents/:id/download-url",  Method: "GET",    AuthRequired: true, Summary: "Genera Signed URL download"},
			{Path: "/api/v1/documents/upload-url",        Method: "POST",   AuthRequired: true, Summary: "Richiedi Signed URL upload (step 1)"},
			{Path: "/api/v1/documents/finalize",          Method: "POST",   AuthRequired: true, Summary: "Finalizza upload e persisti metadati (step 3)"},
			{Path: "/api/v1/documents/:id",               Method: "DELETE", AuthRequired: true, Summary: "Soft delete documento"},
		},
	}
	nexus.RegisterDiscovery(router, serviceDef)
	nexus.StartGatewayHandshake(serviceDef)

	// 5. Rotte protette
	apiGroup := router.Group("/api/v1")
	apiGroup.Use(nexus.Guard())
	{
		apiGroup.GET("/documents",                   docHandler.HandleList)
		apiGroup.GET("/documents/:id",               docHandler.HandleGetByID)
		apiGroup.GET("/documents/:id/download-url",  docHandler.HandleGetDownloadURL)
		apiGroup.POST("/documents/upload-url",       docHandler.HandleGetUploadURL)
		apiGroup.POST("/documents/finalize",         docHandler.HandleFinalizeUpload)
		apiGroup.DELETE("/documents/:id",            docHandler.HandleDelete)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Document Service v1.1.0 avviato sulla porta %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
