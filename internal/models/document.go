package models

import "github.com/DaniFX/ssg-nexus-sdk/pkg/nexus/repository"

// DocumentStatus rappresenta il ciclo di vita del documento
type DocumentStatus string

const (
	StatusDraft    DocumentStatus = "DRAFT"
	StatusPending  DocumentStatus = "PENDING"
	StatusApproved DocumentStatus = "APPROVED"
	StatusRejected DocumentStatus = "REJECTED"
)

// Relation collega il documento a un'entità padre (Entity, Invoice, Project...)
type Relation struct {
	ParentType string `firestore:"parentType" json:"parentType"` // ENTITY | INVOICE | PROJECT
	ParentID   string `firestore:"parentId"   json:"parentId"`
}

// Document è il documento archiviato nel sistema Nexus
type Document struct {
	repository.NexusDoc

	FileName    string                 `firestore:"fileName"    json:"fileName"`
	Description string                 `firestore:"description" json:"description,omitempty"`
	MimeType    string                 `firestore:"mimeType"    json:"mimeType"`
	Size        int64                  `firestore:"size"        json:"size"`
	StoragePath string                 `firestore:"storagePath" json:"storagePath"`
	Category    string                 `firestore:"category"    json:"category"`
	Status      DocumentStatus         `firestore:"status"      json:"status"`
	Metadata    map[string]interface{} `firestore:"metadata"    json:"metadata,omitempty"`
	Relation    Relation               `firestore:"relation"    json:"relation"`
}

// FinalizeRequest è il payload POST /documents/finalize
type FinalizeRequest struct {
	DocID       string `json:"docId"       binding:"required"`
	FileName    string `json:"fileName"    binding:"required"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"    binding:"required"`
	Size        int64  `json:"size"        binding:"required"`
	StoragePath string `json:"storagePath" binding:"required"`
	Category    string `json:"category"    binding:"required"`
	ParentType  string `json:"parentType"  binding:"required"`
	ParentID    string `json:"parentId"    binding:"required"`
}

// UploadURLRequest è il payload POST /documents/upload-url
type UploadURLRequest struct {
	FileName   string `json:"fileName"   binding:"required"`
	MimeType   string `json:"mimeType"   binding:"required"`
	Category   string `json:"category"   binding:"required"`
	ParentType string `json:"parentType" binding:"required"`
	ParentID   string `json:"parentId"   binding:"required"`
}
