package api

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/DaniFX/ssg-nexus-sdk/pkg/nexus"
	"github.com/DaniFX/ssg-nexus-sdk/pkg/nexus/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Interfaccia fittizia per GCS (sostituiscila con la tua implementazione reale dell'SDK)
type GCSClient interface {
	GenerateUploadURL(objectName string, mimeType string, expiresInMinutes int) (string, error)
}

type DocumentHandler struct {
	Repo      *repository.Repository
	GCSClient GCSClient
}

func NewDocumentHandler(repo *repository.Repository, gcs GCSClient) *DocumentHandler {
	return &DocumentHandler{Repo: repo, GCSClient: gcs}
}

// Request struct per l'URL di upload
type UploadURLRequest struct {
	FileName   string `json:"fileName" binding:"required"`
	MimeType   string `json:"mimeType" binding:"required"`
	Category   string `json:"category" binding:"required"`
	ParentType string `json:"parentType" binding:"required"`
	ParentID   string `json:"parentId" binding:"required"`
}

// HandleGetUploadURL: Genera il Signed URL di GCS
func (h *DocumentHandler) HandleGetUploadURL(c *gin.Context) {
	var req UploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		nexus.Failure(c, http.StatusBadRequest, nexus.ErrValidationFailed, "Dati richiesta non validi", err.Error())
		return
	}

	// Genera un ID univoco per il documento
	docID := uuid.New().String()

	// Costruisce il percorso logico su GCS: entities/{parentId}/docs/{docId}_filename.ext
	ext := filepath.Ext(req.FileName)
	storagePath := fmt.Sprintf("entities/%s/docs/%s%s", req.ParentID, docID, ext)

	// Genera l'URL (valido 15 minuti)
	uploadURL, err := h.GCSClient.GenerateUploadURL(storagePath, req.MimeType, 15)
	if err != nil {
		nexus.Failure(c, http.StatusInternalServerError, nexus.ErrInternal, "Errore generazione URL di caricamento", nil)
		return
	}

	nexus.Success(c, gin.H{
		"docId":       docID,
		"uploadUrl":   uploadURL,
		"storagePath": storagePath,
	}, nil)
}

// Request struct per finalizzare l'upload
type FinalizeRequest struct {
	DocID       string `json:"docId" binding:"required"`
	FileName    string `json:"fileName" binding:"required"`
	MimeType    string `json:"mimeType" binding:"required"`
	Size        int64  `json:"size" binding:"required"`
	StoragePath string `json:"storagePath" binding:"required"`
	Category    string `json:"category" binding:"required"`
	ParentType  string `json:"parentType" binding:"required"`
	ParentID    string `json:"parentId" binding:"required"`
}

// HandleFinalizeUpload: Salva i metadati su Firestore
func (h *DocumentHandler) HandleFinalizeUpload(c *gin.Context) {
	var req FinalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		nexus.Failure(c, http.StatusBadRequest, nexus.ErrValidationFailed, "Dati finalizzazione non validi", err.Error())
		return
	}

	// Costruisce la mappa dati da salvare su Firestore (NexusDoc gestirà createdAt/createdBy)
	data := map[string]interface{}{
		"fileName":    req.FileName,
		"mimeType":    req.MimeType,
		"size":        req.Size,
		"storagePath": req.StoragePath,
		"category":    req.Category,
		"relation": map[string]interface{}{
			"parentType": req.ParentType,
			"parentId":   req.ParentID,
		},
		"accessControl": map[string]interface{}{
			"isPublic":     false,
			"allowedRoles": []string{"admin"}, // Default prudenziale
		},
	}

	// Salva nel database usando il Repository dell'SDK
	ctx := c.Request.Context()
	err := h.Repo.Create(ctx, req.DocID, data)
	if err != nil {
		nexus.Failure(c, http.StatusInternalServerError, nexus.ErrInternal, "Impossibile salvare i metadati del documento", err.Error())
		return
	}

	nexus.Success(c, gin.H{"id": req.DocID, "status": "FINALIZED"}, nil)
}
