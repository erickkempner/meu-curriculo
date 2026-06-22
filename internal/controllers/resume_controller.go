package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	resumeSvc  services.ResumeService
	pdfSvc     PDFService
	uploadsDir string
}

// NewResumeController creates a new ResumeController with the given service dependencies.
func NewResumeController(resumeSvc services.ResumeService, pdfSvc PDFService, uploadsDir string) *ResumeController {
	return &ResumeController{
		resumeSvc:  resumeSvc,
		pdfSvc:     pdfSvc,
		uploadsDir: uploadsDir,
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

// maxPhotoSize is the maximum allowed photo file size (2MB).
const maxPhotoSize = 2 << 20

// maxThumbnailSize is the maximum allowed thumbnail base64 payload size (2MB).
const maxThumbnailSize = 2 << 20

// allowedPhotoTypes is the set of accepted MIME types for photo uploads.
var allowedPhotoTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
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

// UploadPhoto handles photo file upload for a resume.
// It saves the file to disk under uploads/<resumeID>.<ext> and updates the DB.
func (ctrl *ResumeController) UploadPhoto(ctx *gin.Context) {
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

	file, header, err := ctx.Request.FormFile("photo")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "foto é obrigatória"})
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > maxPhotoSize {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "foto deve ter no máximo 2MB"})
		return
	}

	// Validate MIME type
	contentType := header.Header.Get("Content-Type")
	if !allowedPhotoTypes[contentType] {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "formato inválido. Use JPEG, PNG ou WebP"})
		return
	}

	// Determine file extension
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	}

	// Save file to uploads directory
	filename := resumeID.String() + ext
	savePath := fmt.Sprintf("%s/photos/%s", ctrl.uploadsDir, filename)

	if err := ctx.SaveUploadedFile(header, savePath); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao salvar foto"})
		return
	}

	// Update DB with the public URL path
	photoURL := "/uploads/photos/" + filename
	err = ctrl.resumeSvc.UpdatePhotoURL(ctx.Request.Context(), userID, resumeID, photoURL)
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

	ctx.JSON(http.StatusOK, gin.H{"photo_url": photoURL})
}

// uploadThumbnailRequest represents the JSON body for thumbnail upload.
type uploadThumbnailRequest struct {
	Image string `json:"image" binding:"required"` // base64-encoded PNG data URL
}

// UploadThumbnail handles thumbnail image upload from client-side html2canvas capture.
// It receives a base64-encoded PNG data URL and stores it directly in the database.
func (ctrl *ResumeController) UploadThumbnail(ctx *gin.Context) {
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

	var req uploadThumbnailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "imagem é obrigatória"})
		return
	}

	// Validate size (rough check on base64 string length — ~400KB max)
	if len(req.Image) > int(maxThumbnailSize) {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "thumbnail muito grande (máx 512KB)"})
		return
	}

	// Validate it's a valid data URL (must start with data:image/)
	if len(req.Image) < 20 || req.Image[:11] != "data:image/" {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "formato de imagem inválido"})
		return
	}

	// Store the data URL directly in the database
	err = ctrl.resumeSvc.UpdateThumbnailURL(ctx.Request.Context(), userID, resumeID, req.Image)
	if err != nil {
		handleResumeErrorJSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"thumbnail_url": "stored"})
}

// ServeThumbnail serves a dynamically generated SVG thumbnail for a resume.
// This endpoint requires auth (the owner's resume list uses it).
func (ctrl *ResumeController) ServeThumbnail(ctx *gin.Context) {
	resumeID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	userID, err := middlewares.GetUserIDFromContext(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	detail, err := ctrl.resumeSvc.GetByID(ctx.Request.Context(), userID, resumeID)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	svg := generateResumeSVG(detail)
	ctx.Header("Cache-Control", "public, max-age=60")
	ctx.Data(http.StatusOK, "image/svg+xml", []byte(svg))
}

// generateResumeSVG creates an SVG representation of the resume for use as a thumbnail.
func generateResumeSVG(detail *models.ResumeDetail) string {
	r := detail.Resume

	// Truncate text helper
	truncate := func(s string, max int) string {
		runes := []rune(s)
		if len(runes) <= max {
			return s
		}
		return string(runes[:max]) + "…"
	}

	// Escape XML special chars
	esc := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		s = strings.ReplaceAll(s, "'", "&apos;")
		return s
	}

	// Choose accent color based on template
	accentColor := "#2563eb"
	accentLight := "#dbeafe"
	switch r.TemplateName {
	case "classico":
		accentColor = "#1f2937"
		accentLight = "#f3f4f6"
	case "minimalista":
		accentColor = "#6b7280"
		accentLight = "#f9fafb"
	}

	var b strings.Builder

	// Higher resolution SVG for crisp rendering — cropped to top half
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 280" width="400" height="280">`)

	// Background
	b.WriteString(`<rect width="400" height="280" fill="#ffffff" rx="4"/>`)

	y := 32.0

	// Name
	name := truncate(esc(r.PersonalName), 28)
	if name == "" {
		name = "Sem nome"
	}
	b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="20" font-weight="700" fill="#111827">%s</text>`, y, name))

	// Title
	y += 22
	if r.PersonalTitle != "" {
		title := truncate(esc(r.PersonalTitle), 40)
		b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="12" font-weight="500" fill="%s">%s</text>`, y, accentColor, title))
		y += 18
	}

	// Contact info
	contactParts := []string{}
	if r.Email != "" {
		contactParts = append(contactParts, esc(r.Email))
	}
	if r.Phone != "" {
		contactParts = append(contactParts, esc(r.Phone))
	}
	if r.Location != "" {
		contactParts = append(contactParts, esc(r.Location))
	}
	if len(contactParts) > 0 {
		contact := truncate(strings.Join(contactParts, "  ·  "), 55)
		b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="10" fill="#6b7280">%s</text>`, y, contact))
		y += 20
	}

	// Summary
	if r.Summary != "" {
		y += 6
		summary := truncate(esc(r.Summary), 90)
		b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="10" fill="#4b5563" opacity="0.9">%s</text>`, y, summary))
		y += 20
	}

	// Separator
	y += 4
	b.WriteString(fmt.Sprintf(`<line x1="24" y1="%.0f" x2="376" y2="%.0f" stroke="#e5e7eb" stroke-width="1"/>`, y, y))
	y += 20

	// Experience section
	if len(detail.Experience) > 0 {
		b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="11" font-weight="700" fill="%s" letter-spacing="1">EXPERIÊNCIA</text>`, y, accentColor))
		y += 20
		for i, exp := range detail.Experience {
			if i >= 2 || y > 250 {
				break
			}
			// Role
			role := truncate(esc(exp.Role), 35)
			if role == "" {
				role = truncate(esc(exp.Company), 35)
			}
			b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="11" font-weight="600" fill="#1f2937">%s</text>`, y, role))
			y += 15

			// Company + period
			companyLine := esc(exp.Company)
			if exp.Period != "" {
				companyLine += "  ·  " + esc(exp.Period)
			}
			companyLine = truncate(companyLine, 50)
			b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="9" fill="#6b7280">%s</text>`, y, companyLine))
			y += 18
		}
	}

	// Education section
	if len(detail.Education) > 0 && y < 240 {
		// Separator
		b.WriteString(fmt.Sprintf(`<line x1="24" y1="%.0f" x2="376" y2="%.0f" stroke="#e5e7eb" stroke-width="1"/>`, y, y))
		y += 20

		b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="11" font-weight="700" fill="%s" letter-spacing="1">EDUCAÇÃO</text>`, y, accentColor))
		y += 20
		for i, edu := range detail.Education {
			if i >= 1 || y > 260 {
				break
			}
			degree := truncate(esc(edu.Degree), 40)
			b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="11" font-weight="600" fill="#1f2937">%s</text>`, y, degree))
			y += 15
			instLine := esc(edu.Institution)
			if edu.Period != "" {
				instLine += "  ·  " + esc(edu.Period)
			}
			instLine = truncate(instLine, 50)
			b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="9" fill="#6b7280">%s</text>`, y, instLine))
			y += 20
		}
	}

	// Skills section
	if len(detail.Skills) > 0 && y < 250 {
		// Separator
		b.WriteString(fmt.Sprintf(`<line x1="24" y1="%.0f" x2="376" y2="%.0f" stroke="#e5e7eb" stroke-width="1"/>`, y, y))
		y += 20

		b.WriteString(fmt.Sprintf(`<text x="24" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="11" font-weight="700" fill="%s" letter-spacing="1">HABILIDADES</text>`, y, accentColor))
		y += 18

		// Skill pills as rounded rects with text
		x := 24.0
		for i, s := range detail.Skills {
			if i >= 6 || y > 270 {
				break
			}
			skillName := truncate(esc(s.Name), 15)
			pillWidth := float64(len([]rune(skillName))*7 + 16)
			if x+pillWidth > 376 {
				x = 24
				y += 22
			}
			if y > 270 {
				break
			}
			b.WriteString(fmt.Sprintf(`<rect x="%.0f" y="%.0f" width="%.0f" height="18" rx="9" fill="%s"/>`, x, y-12, pillWidth, accentLight))
			b.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" font-family="'Segoe UI', Arial, sans-serif" font-size="9" font-weight="500" fill="%s">%s</text>`, x+8, y, accentColor, skillName))
			x += pillWidth + 6
		}
	}

	b.WriteString(`</svg>`)
	return b.String()
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

// handleResumeErrorJSON handles common resume errors for JSON API responses.
func handleResumeErrorJSON(ctx *gin.Context, err error) {
	if errors.Is(err, models.ErrNotFound) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if errors.Is(err, models.ErrForbidden) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

// buildShareURL constructs the public share URL from the request context.
func buildShareURL(ctx *gin.Context, token string) string {
	scheme := "http"
	if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/r/%s", scheme, ctx.Request.Host, token)
}
