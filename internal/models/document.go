package models

import (
	"github.com/DaniFX/ssg-nexus-sdk/pkg/nexus/repository"
)

type Document struct {
	repository.NexusDoc // Campi base: ID, CreatedAt, CreatedBy, ecc.

	FileName    string                 `firestore:"fileName" json:"fileName"`
	MimeType    string                 `firestore:"mimeType" json:"mimeType"`
	Size        int64                  `firestore:"size" json:"size"`
	StoragePath string                 `firestore:"storagePath" json:"storagePath"`
	Category    string                 `firestore:"category" json:"category"`
	Metadata    map[string]interface{} `firestore:"metadata" json:"metadata"`
	Relation    Relation               `firestore:"relation" json:"relation"`
}

type Relation struct {
	ParentType string `firestore:"parentType" json:"parentType"` // ENTITY, INVOICE, PROJECT
	ParentID   string `firestore:"parentId" json:"parentId"`
}