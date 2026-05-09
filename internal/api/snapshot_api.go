package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SnapshotAPI provides HTTP handlers for system snapshot management.
type SnapshotAPI struct {
	snapSvc *service.SnapshotService
}

// NewSnapshotAPI creates a new SnapshotAPI.
func NewSnapshotAPI(db *gorm.DB) *SnapshotAPI {
	return &SnapshotAPI{
		snapSvc: service.NewSnapshotService(db),
	}
}

// CreateSnapshot creates a new system configuration snapshot.
// POST /api/v1/servers/:server_id/snapshots
func (s *SnapshotAPI) CreateSnapshot(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		Name        string                    `json:"name" binding:"required"`
		Description string                    `json:"description"`
		Config      *service.SnapshotConfig   `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	snapshot, err := s.snapSvc.CreateSnapshot(c.Request.Context(), serverID, req.Name, req.Description, req.Config)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, snapshot)
}

// ListSnapshots lists all snapshots for a server.
// GET /api/v1/servers/:server_id/snapshots
func (s *SnapshotAPI) ListSnapshots(c *gin.Context) {
	serverID := c.Param("server_id")

	snapshots, err := s.snapSvc.ListSnapshots(c.Request.Context(), serverID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, snapshots)
}

// GetSnapshot gets a snapshot by ID.
// GET /api/v1/servers/:server_id/snapshots/:id
func (s *SnapshotAPI) GetSnapshot(c *gin.Context) {
	id := c.Param("id")

	snapshot, err := s.snapSvc.GetSnapshot(c.Request.Context(), id)
	if err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}

	respondSuccess(c, snapshot)
}

// DeleteSnapshot deletes a snapshot.
// DELETE /api/v1/servers/:server_id/snapshots/:id
func (s *SnapshotAPI) DeleteSnapshot(c *gin.Context) {
	id := c.Param("id")

	if err := s.snapSvc.DeleteSnapshot(c.Request.Context(), id); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id, "status": "deleted"})
}

// DiffSnapshots compares two snapshots.
// GET /api/v1/servers/:server_id/snapshots/diff?id1=xxx&id2=yyy
func (s *SnapshotAPI) DiffSnapshots(c *gin.Context) {
	serverID := c.Param("server_id")
	id1 := c.Query("id1")
	id2 := c.Query("id2")

	if id1 == "" || id2 == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	diff, err := s.snapSvc.DiffSnapshots(c.Request.Context(), serverID, id1, id2)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, diff)
}

// RestoreSnapshot analyzes what would be restored from a snapshot.
// POST /api/v1/servers/:server_id/snapshots/:id/restore
func (s *SnapshotAPI) RestoreSnapshot(c *gin.Context) {
	serverID := c.Param("server_id")
	id := c.Param("id")

	files, err := s.snapSvc.RestoreSnapshot(c.Request.Context(), serverID, id)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"snapshot_id": id, "files": files, "status": "analysis_complete"})
}

// GetSnapshotFiles returns the current file list that would be snapshotted.
// GET /api/v1/servers/:server_id/snapshots/files
func (s *SnapshotAPI) GetSnapshotFiles(c *gin.Context) {
	serverID := c.Param("server_id")

	var config service.SnapshotConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		// Use default config if not provided
		config = *service.DefaultSnapshotConfig()
	}

	files, err := s.snapSvc.GetSnapshotFiles(c.Request.Context(), serverID, &config)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, files)
}
