package api

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/DaniFX/ssg-nexus-document-service/internal/models"
	"github.com/DaniFX/ssg-nexus-sdk/pkg/nexus"
	"github.com/DaniFX/ssg-nexus-sdk/pkg/nexus/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GCSClient definisce i metodi necessari per il layer storage
type GCSClient interface {
	GenerateUploadURL(objectName, mimeType string, expiresInMinutes int) (string, error)
	GenerateDownloadURL(objectName string, expiresInMinutes int) (string, error)
}

type DocumentHandler struct {
	Repo      *repository.Repository
	GCSClient GCSClient
}

func NewDocumentHandler(repo *repository.Repository, gcs GCSClient) *DocumentHandler {
	return &DocumentHandler{Repo: repo, GCSClient: gcs}
}

// HandleGetUploadURL — POST /api/v1/documents/upload-url
// Step 1 del flow: restituisce un Signed URL PUT per upload diretto su GCS
func (h *DocumentHandler) HandleGetUploadURL(c *gin.Context) {
	var req models.UploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		nexus.Failure(c, http.StatusBadRequest, nexus.ErrValidationFailed, "Dati richiesta non validi", err.Error())
		return
	}

	docID := uuid.New().String()
	ext := filepath.Ext(req.FileName)
	storagePath := fmt.Sprintf("%s/%s/docs/%s%s",
		req.ParentType, req.ParentID, docID, ext)

	uploadURL, err := h.GCSClient.GenerateUploadURL(storagePath, req.MimeType, 15)
	if err != nil {
		nexus.Failure(c, http.StatusInternalServerError, nexus.ErrInternal, "Errore generazione upload URL", err.Error())
		return
	}

	nexus.Success(c, gin.H{
		"docId":       docID,
		"uploadUrl":   uploadURL,
		"storagePath": storagePath,
	}, nil)
}

// HandleFinalizeUpload — POST /api/v1/documents/finalize
// Step 3 del flow: persiste i metadati su Firestore dopo che il browser ha completato il PUT su GCS
func (h *DocumentHandler) HandleFinalizeUpload(c *gin.Context) {
	ctx := c.Request.Context()
	identity := nexus.FromContext(ctx)

	var req models.FinalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		nexus.Failure(c, http.StatusBadRequest, nexus.ErrValidationFailed, "Dati finalizzazione non validi", err.Error())
		return
	}

	data := map[string]interface{}{
		"fileName":    req.FileName,
		"description": req.Description,
		"mimeType":   req.MimeType,
		"size":        req.Size,
		"storagePath": req.StoragePath,
		"category":    req.Category,
		"status":      string(models.StatusDraft),
		"relation": map[string]interface{}{
			"parentType": req.ParentType,
			"parentId":   req.ParentID,
		},
		"accessControl": map[string]interface{}{
			"isPublic":     false,
			"allowedRoles": []string{"admin", "operator"},
		},
	}

	if err := h.Repo.Create(ctx, req.DocID, data); err != nil {
		nexus.Failure(c, http.StatusInternalServerError, nexus.ErrInternal, "Impossibile salvare i metadati", err.Error())
		return
	}

	nexus.Success(c, gin.H{"id": req.DocID, "status": models.StatusDraft}, gin.H{"createdBy": identity.UserID})
}

// HandleList — GET /api/v1/documents
// Supporta Navigator Pattern: ?parentType=ENTITY&parentId=xxx&status=DRAFT&category=contratto&limit=20&offset=0
func (h *DocumentHandler) HandleList(c *gin.Context) {
	ctx := c.Request.Context()

	var docs []models.Document
	meta, err := h.Repo.ApplyNavigator(ctx, c.Request.URL.Query(), &docs)
	if err != nil {
		nexus.Failure(c, http.StatusInternalServerError, nexus.ErrInternal, "Errore lettura lista documenti", err.Error())
		return
	}

	nexus.Success(c, docs, meta)
}

// HandleGetByID — GET /api/v1/documents/:id
func (h *DocumentHandler) HandleGetByID(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var doc models.Document
	if err := h.Repo.GetByID(ctx, id, &doc); err != nil {
		nexus.Failure(c, http.StatusNotFound, nexus.ErrNotFound, "Documento non trovato", err.Error())
		return
	}

	nexus.Success(c, doc, nil)
}

// HandleGetDownloadURL — GET /api/v1/documents/:id/download-url
// Genera un Signed URL GET valido 60 minuti per visualizzazione/download
func (h *DocumentHandler) HandleGetDownloadURL(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var doc models.Document
	if err := h.Repo.GetByID(ctx, id, &doc); err != nil {
		nexus.Failure(c, http.StatusNotFound, nexus.ErrNotFound, "Documento non trovato", err.Error())
		return
	}

	downloadURL, err := h.GCSClient.GenerateDownloadURL(doc.StoragePath, 60)
	if err != nil {
		nexus.Failure(c, http.StatusInternalServerError, nexus.ErrInternal, "Errore generazione download URL", err.Error())
		return
	}

	nexus.Success(c, gin.H{
		"id":          doc.NexusDoc.ID,
		"fileName":    doc.FileName,
		"downloadUrl": downloadURL,
		"expiresIn":   "60m",
	}, nil)
}

// HandleDelete — DELETE /api/v1/documents/:id (soft delete)
func (h *DocumentHandler) HandleDelete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var doc models.Document
	if err := h.Repo.GetByID(ctx, id, &doc); err != nil {
		nexus.Failure(c, http.StatusNotFound, nexus.ErrNotFound, "Documento non trovato", err.Error())
		return
	}

	if err := h.Repo.SoftDelete(ctx, id); err != nil {
		nexus.Failure(c, http.StatusInternalServerError, nexus.ErrInternal, "Errore eliminazione documento", err.Error())
		return
	}

	nexus.Success(c, gin.H{"id": id, "deleted": true}, nil)
}
