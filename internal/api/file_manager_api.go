package api

import (
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FileManagerAPI provides HTTP handlers for remote file management.
type FileManagerAPI struct {
	fileSvc *service.FileManagerService
}

// NewFileManagerAPI creates a new FileManagerAPI.
func NewFileManagerAPI(db *gorm.DB, sb *sandbox.Sandbox) *FileManagerAPI {
	return &FileManagerAPI{
		fileSvc: service.NewFileManagerService(db, sb),
	}
}

// ListFiles lists files and directories at a remote path.
// GET /api/v1/servers/:server_id/files?path=/home
func (f *FileManagerAPI) ListFiles(c *gin.Context) {
	serverID := c.Param("server_id")
	remotePath := c.DefaultQuery("path", "/home")

	entries, err := f.fileSvc.ListFiles(c.Request.Context(), serverID, remotePath)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"items": entries, "path": remotePath})
}

// ReadFile reads the content of a remote file.
// GET /api/v1/servers/:server_id/files/read?path=/home/app/config.yaml&max_bytes=1048576
func (f *FileManagerAPI) ReadFile(c *gin.Context) {
	serverID := c.Param("server_id")
	remotePath := c.Query("path")
	if remotePath == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	maxBytes, _ := strconv.ParseInt(c.DefaultQuery("max_bytes", "1048576"), 10, 64) // default 1MB

	content, err := f.fileSvc.ReadFile(c.Request.Context(), serverID, remotePath, maxBytes)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, content)
}

// WriteFile writes content to a remote file.
// PUT /api/v1/servers/:server_id/files/write
func (f *FileManagerAPI) WriteFile(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := f.fileSvc.WriteFile(c.Request.Context(), serverID, req.Path, req.Content); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"path": req.Path, "status": "written"})
}

// DeleteFile deletes a remote file or directory.
// DELETE /api/v1/servers/:server_id/files?path=/home/app/old
func (f *FileManagerAPI) DeleteFile(c *gin.Context) {
	serverID := c.Param("server_id")
	remotePath := c.Query("path")
	if remotePath == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := f.fileSvc.DeleteFile(c.Request.Context(), serverID, remotePath); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"path": remotePath, "status": "deleted"})
}

// CreateDirectory creates a remote directory.
// POST /api/v1/servers/:server_id/files/mkdir
func (f *FileManagerAPI) CreateDirectory(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := f.fileSvc.CreateDirectory(c.Request.Context(), serverID, req.Path); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"path": req.Path, "status": "created"})
}

// MoveFile moves/renames a remote file.
// POST /api/v1/servers/:server_id/files/move
func (f *FileManagerAPI) MoveFile(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		SrcPath string `json:"src_path" binding:"required"`
		DstPath string `json:"dst_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := f.fileSvc.MoveFile(c.Request.Context(), serverID, req.SrcPath, req.DstPath); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"src": req.SrcPath, "dst": req.DstPath, "status": "moved"})
}

// GetDiskUsage returns disk usage for a remote path.
// GET /api/v1/servers/:server_id/files/disk-usage?path=/
func (f *FileManagerAPI) GetDiskUsage(c *gin.Context) {
	serverID := c.Param("server_id")
	remotePath := c.DefaultQuery("path", "/")

	usage, err := f.fileSvc.GetDiskUsage(c.Request.Context(), serverID, remotePath)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, usage)
}

// GetFileInfo returns detailed info about a single remote file.
// GET /api/v1/servers/:server_id/files/info?path=/home/app/config.yaml
func (f *FileManagerAPI) GetFileInfo(c *gin.Context) {
	serverID := c.Param("server_id")
	remotePath := c.Query("path")
	if remotePath == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	info, err := f.fileSvc.GetFileInfo(c.Request.Context(), serverID, remotePath)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, info)
}

// SearchFiles searches for files matching a pattern.
// GET /api/v1/servers/:server_id/files/search?path=/home&pattern=config&max_results=50
func (f *FileManagerAPI) SearchFiles(c *gin.Context) {
	serverID := c.Param("server_id")
	searchPath := c.DefaultQuery("path", "/home")
	pattern := c.Query("pattern")
	if pattern == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	maxResults, _ := strconv.Atoi(c.DefaultQuery("max_results", "50"))

	entries, err := f.fileSvc.SearchFiles(c.Request.Context(), serverID, searchPath, pattern, maxResults)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"items": entries, "total": len(entries)})
}
