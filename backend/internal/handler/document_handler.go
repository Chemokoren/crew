package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/storage"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
)

type DocumentHandler struct {
	docSvc      *service.DocumentService
	minioClient *storage.MinIOClient
}

func NewDocumentHandler(svc *service.DocumentService, minioClient *storage.MinIOClient) *DocumentHandler {
	return &DocumentHandler{docSvc: svc, minioClient: minioClient}
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	if h.minioClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "File storage (MinIO) is not available"})
		return
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		BadRequest(c, "No file uploaded")
		return
	}

	docTypeStr := c.PostForm("document_type")
	if docTypeStr == "" {
		BadRequest(c, "document_type is required")
		return
	}

	docType := models.DocumentType(docTypeStr)

	// Parse optional entities
	var crewMemberID, saccoID, vehicleID *uuid.UUID
	if cm := c.PostForm("crew_member_id"); cm != "" {
		if id, err := uuid.Parse(cm); err == nil {
			crewMemberID = &id
		}
	}
	if sm := c.PostForm("sacco_id"); sm != "" {
		if id, err := uuid.Parse(sm); err == nil {
			saccoID = &id
		}
	}
	if vm := c.PostForm("vehicle_id"); vm != "" {
		if id, err := uuid.Parse(vm); err == nil {
			vehicleID = &id
		}
	}

	// Open file stream
	f, err := file.Open()
	if err != nil {
		InternalError(c, "Failed to read file")
		return
	}
	defer f.Close()

	contentType := file.Header.Get("Content-Type")
	objectName := fmt.Sprintf("%s/%s", uuid.New().String(), file.Filename)

	// Upload to MinIO
	path, err := h.minioClient.UploadFile(c.Request.Context(), objectName, f, file.Size, contentType)
	if err != nil {
		MapServiceError(c, fmt.Errorf("upload to storage: %w", err))
		return
	}

	// Create document record
	doc, err := h.docSvc.CreateDocument(c.Request.Context(), service.CreateDocumentInput{
		CrewMemberID: crewMemberID,
		SaccoID:      saccoID,
		VehicleID:    vehicleID,
		DocumentType: docType,
		FileName:     file.Filename,
		FileSize:     file.Size,
		MimeType:     contentType,
		StoragePath:  path,
		UploadedByID: claims.UserID,
	})

	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, doc)
}

func (h *DocumentHandler) Download(c *gin.Context) {
	if h.minioClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "File storage (MinIO) is not available"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid document ID")
		return
	}

	doc, err := h.docSvc.GetDocument(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	url, err := h.minioClient.PresignedDownloadURL(c.Request.Context(), doc.StoragePath, time.Hour)
	if err != nil {
		MapServiceError(c, fmt.Errorf("generate download link: %w", err))
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{"download_url": url})
}

func (h *DocumentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	var filter repository.DocumentFilter
	if cm := c.Query("crew_member_id"); cm != "" {
		if id, err := uuid.Parse(cm); err == nil {
			filter.CrewMemberID = &id
		}
	}
	if sm := c.Query("sacco_id"); sm != "" {
		if id, err := uuid.Parse(sm); err == nil {
			filter.SaccoID = &id
		}
	}
	if vm := c.Query("vehicle_id"); vm != "" {
		if id, err := uuid.Parse(vm); err == nil {
			filter.VehicleID = &id
		}
	}
	filter.DocumentType = c.Query("document_type")

	docs, total, err := h.docSvc.ListDocuments(c.Request.Context(), filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	ListResponse(c, docs, buildMeta(page, perPage, total))
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid document ID")
		return
	}

	// For safety, could delete from MinIO first, but here we just delete metadata
	if err := h.docSvc.DeleteDocument(c.Request.Context(), id); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Document metadata deleted"})
}
