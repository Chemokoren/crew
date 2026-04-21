package service_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)

func TestDocumentService_CreateDocument(t *testing.T) {
	repo := mock.NewDocumentRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewDocumentService(repo, logger)

	crewID := uuid.New()
	userID := uuid.New()

	input := service.CreateDocumentInput{
		CrewMemberID: &crewID,
		DocumentType: models.DocumentType("NATIONAL_ID"),
		FileName:     "id_front.jpg",
		FileSize:     1024,
		MimeType:     "image/jpeg",
		StoragePath:  "documents/id_front.jpg",
		UploadedByID: userID,
	}

	doc, err := svc.CreateDocument(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if doc.FileName != "id_front.jpg" {
		t.Errorf("expected id_front.jpg, got %s", doc.FileName)
	}
}

func TestDocumentService_GetAndDeleteDocument(t *testing.T) {
	repo := mock.NewDocumentRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewDocumentService(repo, logger)

	doc, _ := svc.CreateDocument(context.Background(), service.CreateDocumentInput{
		DocumentType: models.DocumentType("OTHER"),
		FileName:     "test.txt",
		UploadedByID: uuid.New(),
	})

	fetched, err := svc.GetDocument(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fetched.FileName != "test.txt" {
		t.Errorf("expected test.txt, got %s", fetched.FileName)
	}

	err = svc.DeleteDocument(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("expected no error on delete, got %v", err)
	}

	_, err = svc.GetDocument(context.Background(), doc.ID)
	if err == nil {
		t.Errorf("expected error fetching deleted document")
	}
}

func TestDocumentService_ListDocuments(t *testing.T) {
	repo := mock.NewDocumentRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewDocumentService(repo, logger)

	svc.CreateDocument(context.Background(), service.CreateDocumentInput{
		DocumentType: models.DocumentType("NATIONAL_ID"),
		FileName:     "doc1.jpg",
	})
	svc.CreateDocument(context.Background(), service.CreateDocumentInput{
		DocumentType: models.DocumentType("OTHER"),
		FileName:     "doc2.txt",
	})

	filter := repository.DocumentFilter{DocumentType: "NATIONAL_ID"}
	docs, total, err := svc.ListDocuments(context.Background(), filter, 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 document, got %d", total)
	}
	if len(docs) != 1 || docs[0].FileName != "doc1.jpg" {
		t.Errorf("expected doc1.jpg, got different")
	}
}
