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

// Mock temporaneo per il GCSClient, da sostituire con il modulo GCS che metterai nell'SDK
type mockGCS struct{}

func (m *mockGCS) GenerateUploadURL(objectName, mimeType string, expiresIn int) (string, error) {
	return "https://storage.googleapis.com/test-bucket/" + objectName + "?signed=true", nil
}

func main() {
	// Carica il file .env se esiste (ignora l'errore in produzione su Cloud Run dove non c'è)
	_ = godotenv.Load()

	ctx := context.Background()
	// Recupera le configurazioni dall'ambiente

	projectID := os.Getenv("GCP_PROJECT_ID")
	bucketName := os.Getenv("FIREBASE_STORAGE_BUCKET") // Es: ssg-nexus.firebasestorage.app

	// 1. Inizializza Firestore (SDK)
	fsClient, _ := firestore.NewClient(ctx, projectID)
	docRepo := repository.NewRepository(fsClient, "documents")

	// 2. Inizializza il CLIENT REALE di GCS
	gcsClient, err := storage.NewGCSClient(ctx, bucketName)
	if err != nil {
		log.Fatalf("Errore storage client: %v", err)
	}

	// 3. Iniezione dell'handler reale
	docHandler := api.NewDocumentHandler(docRepo, gcsClient)

	router := gin.Default()

	serviceDef := nexus.ServiceDefinition{
		ServiceName: "document-service",
		Version:     "1.0.0",
		Endpoints:   []nexus.Endpoint{}, // (Lascia la def. di prima)
	}
	nexus.RegisterDiscovery(router, serviceDef)

	// Rotte protette
	apiGroup := router.Group("/api/v1")
	apiGroup.Use(nexus.Guard())
	{
		apiGroup.POST("/documents/upload-url", docHandler.HandleGetUploadURL)
		apiGroup.POST("/documents/finalize", docHandler.HandleFinalizeUpload)
	}

	nexus.StartGatewayHandshake(serviceDef)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
