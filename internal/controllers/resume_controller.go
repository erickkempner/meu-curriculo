package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/erick/curriculo/internal/middlewares"
	"github.com/erick/curriculo/internal/models"
	"github.com/erick/curriculo/internal/services"
	"github.com/erick/curriculo/internal/views/pages"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PDFService defines the contract for PDF generation (used by ResumeController).
type PDFService interface {
	// GeneratePDF will be implemented in the PDF service task.
}

// ResumeController handles resume-related HTTP requests.
type ResumeController struct {
	resumeSvc services.ResumeService
	pdfSvc    PDFService
}

// NewResumeController creates a new ResumeController with the given service dependencies.
func NewResumeController(resumeSvc services.ResumeService, pdfSvc PDFService) *ResumeController {
	return &ResumeController{
		resumeSvc: resumeSvc,
		pdfSvc:    pdfSvc,
	}
}

// createResumeRequest represents the JSON body sent by Alpine.js for resume creation/update.
type createResumeRequest struct {
	Title         string              `json:"title"`
	TemplateName  string              `json:"template_name"`
	PersonalName  string              `json:"personal_name"`
	PersonalTitle string              `json:"personal_title"`
	Email         string              `json:"email"`
	Phone         string              `json:"phone"`
	Location      string              `json:"location"`
	Summary       string              `json:"summary"`
	Experience    []experienceRequest `json:"experience"`
	Education     []educationRequest  `json:"education"`
	Skills        []string            `json:"skills"`
}

type experienceRequest struct {
	Company     string `json:"company"`
	Role        string `json:"role"`
	Period      string `json:"period"`
	Description string `json:"description"`
}

type educationRequest struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Period      string `json:"period"`
}

// List renders the user's resume list.
func (ctrl *ResumeController) List(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login")
		return
	}

	resumes, err := ctrl.resumeSvc.List(ctx.Request.Context(), userID)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "internal error")
		return
	}

	userName, _ := ctx.Cookie("user_name")
	render(ctx, http.StatusOK, pages.MyResumesWithDataAndUser(resumes, userName))
}

// CreatePage renders the create resume form.
func (ctrl *ResumeController) CreatePage(ctx *gin.Context) {
	csrfToken := middlewares.GetCSRFToken(ctx)
	render(ctx, http.StatusOK, pages.CreateResumeWithCSRF(csrfToken))
}

// Create handles resume creation from Alpine.js JSON body.
func (ctrl *ResumeController) Create(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createResumeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request body"})
		return
	}

	input := mapRequestToCreateInput(req)

	resume, err := ctrl.resumeSvc.Create(ctx.Request.Context(), userID, input)
	if err != nil {
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": validationErr.Fields})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	successURL := fmt.Sprintf("/resumes/%s/success", resume.ID.String())

	// HTMX redirect
	if isHTMXRequest(ctx) {
		ctx.Header("HX-Redirect", successURL)
		ctx.Status(http.StatusOK)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"id": resume.ID.String(), "redirect": successURL})
}

// EditPage renders the edit resume form.
func (ctrl *ResumeController) EditPage(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login")
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "invalid resume id")
		return
	}

	detail, err := ctrl.resumeSvc.GetByID(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		handleResumeError(ctx, err)
		return
	}

	csrfToken := middlewares.GetCSRFToken(ctx)
	render(ctx, http.StatusOK, pages.EditResumeWithCSRF(detail, csrfToken))
}

// Update handles resume update from Alpine.js JSON body.
func (ctrl *ResumeController) Update(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume id"})
		return
	}

	var req createResumeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request body"})
		return
	}

	input := mapRequestToCreateInput(req)

	err = ctrl.resumeSvc.Update(ctx.Request.Context(), userID, resumeID, input)
	if err != nil {
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": validationErr.Fields})
			return
		}
		if errors.Is(err, models.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// HTMX redirect
	if isHTMXRequest(ctx) {
		ctx.Header("HX-Redirect", "/resumes")
		ctx.Status(http.StatusOK)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"redirect": "/resumes"})
}

// Delete handles resume deletion.
func (ctrl *ResumeController) Delete(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume id"})
		return
	}

	err = ctrl.resumeSvc.Delete(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	ctx.String(http.StatusOK, "")
}

// Duplicate handles resume duplication.
func (ctrl *ResumeController) Duplicate(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login")
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "invalid resume id")
		return
	}

	_, err = ctrl.resumeSvc.Duplicate(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		handleResumeError(ctx, err)
		return
	}

	if isHTMXRequest(ctx) {
		ctx.Header("HX-Redirect", "/resumes")
		ctx.Status(http.StatusOK)
		return
	}

	ctx.Redirect(http.StatusFound, "/resumes")
}

// ExportPDF renders the resume as a printable HTML page that can be saved as PDF via browser.
func (ctrl *ResumeController) ExportPDF(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login")
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "invalid resume id")
		return
	}

	detail, err := ctrl.resumeSvc.GetByID(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		handleResumeError(ctx, err)
		return
	}

	render(ctx, http.StatusOK, pages.ResumePrintView(detail))
}

// Success renders the success page after creating a resume.
func (ctrl *ResumeController) Success(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login")
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "invalid resume id")
		return
	}

	detail, err := ctrl.resumeSvc.GetByID(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		handleResumeError(ctx, err)
		return
	}

	// Generate share token if not exists
	var shareURL string
	if detail.ShareToken != nil {
		shareURL = buildShareURL(ctx, *detail.ShareToken)
	} else {
		token, err := ctrl.resumeSvc.GenerateShareToken(ctx.Request.Context(), userID, resumeID)
		if err != nil {
			handleResumeError(ctx, err)
			return
		}
		shareURL = buildShareURL(ctx, token)
	}

	render(ctx, http.StatusOK, pages.ResumeSuccess(&detail.Resume, shareURL))
}

// Share generates a share link for the resume.
func (ctrl *ResumeController) Share(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume id"})
		return
	}

	token, err := ctrl.resumeSvc.GenerateShareToken(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	shareURL := buildShareURL(ctx, token)
	ctx.JSON(http.StatusOK, gin.H{"share_url": shareURL, "token": token})
}

// RevokeShare revokes the share link for the resume.
func (ctrl *ResumeController) RevokeShare(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume id"})
		return
	}

	err = ctrl.resumeSvc.RevokeShareToken(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	ctx.Status(http.StatusOK)
}

// RegenerateShare generates a new share link for the resume.
func (ctrl *ResumeController) RegenerateShare(ctx *gin.Context) {
	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume id"})
		return
	}

	token, err := ctrl.resumeSvc.RegenerateShareToken(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	shareURL := buildShareURL(ctx, token)
	ctx.JSON(http.StatusOK, gin.H{"share_url": shareURL, "token": token})
}

// PublicView renders a shared resume (no auth required).
func (ctrl *ResumeController) PublicView(ctx *gin.Context) {
	token := ctx.Param("token")
	if token == "" {
		ctx.String(http.StatusBadRequest, "missing token")
		return
	}

	detail, err := ctrl.resumeSvc.GetByShareToken(ctx.Request.Context(), token)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			ctx.String(http.StatusNotFound, "currículo não encontrado")
			return
		}
		ctx.String(http.StatusInternalServerError, "internal error")
		return
	}

	render(ctx, http.StatusOK, pages.ResumePrintView(detail))
}

// --- Internal helpers ---

// mapRequestToCreateInput converts the JSON request body to the service input struct.
func mapRequestToCreateInput(req createResumeRequest) services.CreateResumeInput {
	input := services.CreateResumeInput{
		Title:         req.Title,
		TemplateName:  req.TemplateName,
		PersonalName:  req.PersonalName,
		PersonalTitle: req.PersonalTitle,
		Email:         req.Email,
		Phone:         req.Phone,
		Location:      req.Location,
		Summary:       req.Summary,
		Skills:        req.Skills,
	}

	input.Experience = make([]services.ExperienceInput, len(req.Experience))
	for i, exp := range req.Experience {
		input.Experience[i] = services.ExperienceInput{
			Company:     exp.Company,
			Role:        exp.Role,
			Period:      exp.Period,
			Description: exp.Description,
		}
	}

	input.Education = make([]services.EducationInput, len(req.Education))
	for i, edu := range req.Education {
		input.Education[i] = services.EducationInput{
			Institution: edu.Institution,
			Degree:      edu.Degree,
			Period:      edu.Period,
		}
	}

	return input
}

// handleResumeError handles common resume errors for HTML page responses.
func handleResumeError(ctx *gin.Context, err error) {
	if errors.Is(err, models.ErrNotFound) {
		ctx.String(http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, models.ErrForbidden) {
		ctx.String(http.StatusForbidden, "forbidden")
		return
	}
	ctx.String(http.StatusInternalServerError, "internal error")
}

// buildShareURL constructs the public share URL from the request context.
func buildShareURL(ctx *gin.Context, token string) string {
	scheme := "http"
	if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/r/%s", scheme, ctx.Request.Host, token)
}
