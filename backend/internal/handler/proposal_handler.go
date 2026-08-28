package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"corporate-action-plan/backend/internal/repository"
	"corporate-action-plan/backend/internal/utils"
	"corporate-action-plan/backend/models"
)

type ProposalHandler struct {
	proposalRepo *repository.ProposalRepository
	projectRepo  *repository.ProjectRepository
}

func NewProposalHandler(proposalRepo *repository.ProposalRepository, projectRepo *repository.ProjectRepository) *ProposalHandler {
	return &ProposalHandler{proposalRepo: proposalRepo, projectRepo: projectRepo}
}

// ListProposals mengembalikan daftar proposal sesuai lingkup akses role. Super Admin:
// semua perusahaan. Admin Operasional: seluruh divisi di perusahaannya. Manager: divisinya
// sendiri (untuk approve/reject). Staff: hanya proposal miliknya. Secara default hanya
// menampilkan status "Pending Approval" kecuali query param ?status= diisi eksplisit
// (misal ?status=all untuk melihat semua status).
func (h *ProposalHandler) ListProposals(c *gin.Context) {
	auth := getAuthContext(c)

	filterPerusahaanID := auth.PerusahaanID
	var filterDivisiID *uint
	var filterCreatedByID uint

	switch {
	case auth.Role == utils.RoleSuperAdmin:
		filterPerusahaanID = 0
	case auth.Role == utils.RoleAdminOperasional:
		// seluruh divisi di perusahaannya
	case auth.Role == utils.RoleManager:
		filterDivisiID = &auth.DivisiID
	default: // Staff / PIC
		filterCreatedByID = auth.UserID
	}

	status := c.DefaultQuery("status", "Pending Approval")
	if status == "all" {
		status = ""
	}

	proposals, err := h.proposalRepo.List(filterPerusahaanID, filterDivisiID, filterCreatedByID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data proposal"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"proposals": proposals})
}

type createProposalRequest struct {
	ProjectID *uint  `json:"project_id"`
	Judul     string `json:"judul" binding:"required"`
	Deskripsi string `json:"deskripsi"`
}

// CreateProposal membuat proposal ide baru oleh Staff (route dibatasi RequireRole).
// Status awal selalu "Pending Approval", menunggu keputusan Manager.
func (h *ProposalHandler) CreateProposal(c *gin.Context) {
	var req createProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auth := getAuthContext(c)

	if req.ProjectID != nil {
		project, err := h.projectRepo.FindByID(*req.ProjectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi project"})
			return
		}
		if project == nil || project.PerusahaanID != auth.PerusahaanID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Project tidak ditemukan pada perusahaan ini"})
			return
		}
	}

	var divisiID *uint
	if auth.DivisiID != 0 {
		divisiID = &auth.DivisiID
	}

	proposal := models.Proposal{
		PerusahaanID: auth.PerusahaanID,
		DivisiID:     divisiID,
		ProjectID:    req.ProjectID,
		CreatedByID:  auth.UserID,
		Judul:        req.Judul,
		Deskripsi:    req.Deskripsi,
		Status:       "Pending Approval",
	}

	if err := h.proposalRepo.Create(&proposal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat proposal"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Proposal berhasil dibuat", "proposal": proposal})
}

type updateProposalRequest struct {
	Status          *string `json:"status" binding:"required,oneof=Approved Rejected"`
	CatatanApproval *string `json:"catatan_approval"`
}

// UpdateProposal melakukan approve/reject proposal oleh Manager/Admin/Super Admin
// (route dibatasi RequireRole). Manager hanya boleh memproses proposal di divisinya sendiri.
func (h *ProposalHandler) UpdateProposal(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID proposal tidak valid"})
		return
	}

	proposal, err := h.proposalRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data proposal"})
		return
	}
	if proposal == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Proposal tidak ditemukan"})
		return
	}

	auth := getAuthContext(c)
	if auth.Role != utils.RoleSuperAdmin && proposal.PerusahaanID != auth.PerusahaanID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola proposal di luar perusahaan Anda"})
		return
	}
	if auth.Role == utils.RoleManager && (proposal.DivisiID == nil || *proposal.DivisiID != auth.DivisiID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat mengelola proposal di luar divisi Anda"})
		return
	}
	if proposal.Status != "Pending Approval" {
		c.JSON(http.StatusConflict, gin.H{"error": "Proposal ini sudah diproses sebelumnya"})
		return
	}

	var req updateProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proposal.Status = *req.Status
	approverID := auth.UserID
	proposal.ApprovedByID = &approverID
	if req.CatatanApproval != nil {
		proposal.CatatanApproval = *req.CatatanApproval
	}

	if err := h.proposalRepo.Update(proposal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui proposal"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Proposal berhasil diperbarui", "proposal": proposal})
}
