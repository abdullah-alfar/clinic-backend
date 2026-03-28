package attachment

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// StorageProvider abstracts where files are stored (Local vs S3)
type StorageProvider interface {
	Save(tenantID uuid.UUID, fileID uuid.UUID, filename string, reader io.Reader) (string, error)
	Delete(fileURL string) error
}

type LocalFileStorage struct {
	BaseDir string
}

func NewLocalFileStorage(baseDir string) *LocalFileStorage {
	return &LocalFileStorage{BaseDir: baseDir}
}

func (l *LocalFileStorage) Save(tenantID uuid.UUID, fileID uuid.UUID, filename string, reader io.Reader) (string, error) {
	tenantDir := filepath.Join(l.BaseDir, tenantID.String())
	if err := os.MkdirAll(tenantDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create tenant directory: %w", err)
	}

	ext := filepath.Ext(filename)
	uniqueFileName := fmt.Sprintf("%s%s", fileID.String(), ext)
	dstPath := filepath.Join(tenantDir, uniqueFileName)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, reader); err != nil {
		return "", fmt.Errorf("failed to write file body: %w", err)
	}

	// This assumes the API serves static files via /uploads/
	return fmt.Sprintf("/uploads/%s/%s", tenantID.String(), uniqueFileName), nil
}

func (l *LocalFileStorage) Delete(fileURL string) error {
	// fileURL format: /uploads/tenant_id/filename
	// A robust implementation should strip a base prefix and resolve absolute paths safely
	relPath := filepath.Clean(fileURL) // e.g., /uploads/tenant_id/uuid.pdf
	
	// Assuming BaseDir is "uploads", we strip leading '/uploads'
	basePrefix := "/uploads"
	if len(relPath) >= len(basePrefix) && relPath[:len(basePrefix)] == basePrefix {
		relPath = relPath[len(basePrefix):] // e.g., /tenant_id/uuid.pdf
	}
	absPath := filepath.Join(l.BaseDir, relPath)
	
	err := os.Remove(absPath)
	if err != nil && os.IsNotExist(err) {
		return nil // Already deleted or not found
	}
	return err
}
