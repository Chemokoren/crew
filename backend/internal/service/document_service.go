package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// DocumentService handles document metadata and storage references.
type DocumentService struct {
	docRepo repository.DocumentRepository
	logger  *slog.Logger
}

func NewDocumentService(docRepo repository.DocumentRepository, logger *slog.Logger) *DocumentService {
	return &DocumentService{docRepo: docRepo, logger: logger}
}

type CreateDocumentInput struct {
	CrewMemberID *uuid.UUID         `json:"crew_member_id"`
	SaccoID      *uuid.UUID         `json:"sacco_id"`
	VehicleID    *uuid.UUID         `json:"vehicle_id"`
	DocumentType models.DocumentType `json:"document_type" binding:"required"`
	FileName     string             `json:"file_name" binding:"required"`
	FileSize     int64              `json:"file_size"`
	MimeType     string             `json:"mime_type"`
	StoragePath  string             // Set by handler after MinIO upload
	UploadedByID uuid.UUID
}

func (s *DocumentService) CreateDocument(ctx context.Context, input CreateDocumentInput) (*models.Document, error) {
	doc := &models.Document{
		CrewMemberID: input.CrewMemberID,
		SaccoID:      input.SaccoID,
		VehicleID:    input.VehicleID,
		DocumentType: input.DocumentType,
		FileName:     input.FileName,
		FileSize:     input.FileSize,
		MimeType:     input.MimeType,
		StoragePath:  input.StoragePath,
		UploadedByID: input.UploadedByID,
	}
	if err := s.docRepo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	s.logger.Info("document created", slog.String("id", doc.ID.String()), slog.String("type", string(doc.DocumentType)))
	return doc, nil
}

func (s *DocumentService) GetDocument(ctx context.Context, id uuid.UUID) (*models.Document, error) {
	return s.docRepo.GetByID(ctx, id)
}

func (s *DocumentService) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	return s.docRepo.Delete(ctx, id)
}

func (s *DocumentService) ListDocuments(ctx context.Context, filter repository.DocumentFilter, page, perPage int) ([]models.Document, int64, error) {
	return s.docRepo.List(ctx, filter, page, perPage)
}
