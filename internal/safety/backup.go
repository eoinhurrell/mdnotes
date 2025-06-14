package safety

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Backup represents a file or directory backup
type Backup struct {
	ID           string                 `json:"id"`
	OriginalPath string                 `json:"original_path"`
	Content      []byte                 `json:"content,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	Type         BackupType             `json:"type"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BackupType represents the type of backup
type BackupType string

const (
	FileBackup      BackupType = "file"
	DirectoryBackup BackupType = "directory"
)

// BackupManager manages file and directory backups
type BackupManager struct {
	backupDir string
	backups   map[string]*Backup
	mutex     sync.RWMutex
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupDir string) *BackupManager {
	// Ensure backup directory exists
	os.MkdirAll(backupDir, 0755)

	return &BackupManager{
		backupDir: backupDir,
		backups:   make(map[string]*Backup),
	}
}

// CreateBackup creates a backup of a single file
func (bm *BackupManager) CreateBackup(filePath string) (string, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", filePath, err)
	}

	// Create backup
	backup := &Backup{
		ID:           uuid.New().String(),
		OriginalPath: filePath,
		Content:      content,
		CreatedAt:    time.Now(),
		Type:         FileBackup,
	}

	// Store backup
	bm.backups[backup.ID] = backup

	// Optionally write backup to disk for persistence
	if err := bm.writeToDisk(backup); err != nil {
		return "", fmt.Errorf("writing backup to disk: %w", err)
	}

	return backup.ID, nil
}

// CreateDirectoryBackup creates a backup of a directory
func (bm *BackupManager) CreateDirectoryBackup(dirPath string) (string, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	// Collect all files in directory
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(dirPath, path)
			files = append(files, relPath)
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("walking directory %s: %w", dirPath, err)
	}

	// Create backup metadata
	metadata := map[string]interface{}{
		"files": files,
		"count": len(files),
	}

	backup := &Backup{
		ID:           uuid.New().String(),
		OriginalPath: dirPath,
		CreatedAt:    time.Now(),
		Type:         DirectoryBackup,
		Metadata:     metadata,
	}

	// Store backup
	bm.backups[backup.ID] = backup

	// Create directory backup on disk
	if err := bm.writeDirectoryToDisk(backup, dirPath); err != nil {
		return "", fmt.Errorf("writing directory backup to disk: %w", err)
	}

	return backup.ID, nil
}

// Restore restores a backup to the specified path
func (bm *BackupManager) Restore(backupID, targetPath string) error {
	bm.mutex.RLock()
	backup, exists := bm.backups[backupID]
	bm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	switch backup.Type {
	case FileBackup:
		return bm.restoreFile(backup, targetPath)
	case DirectoryBackup:
		return bm.restoreDirectory(backup, targetPath)
	default:
		return fmt.Errorf("unknown backup type: %s", backup.Type)
	}
}

// restoreFile restores a file backup
func (bm *BackupManager) restoreFile(backup *Backup, targetPath string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	// Write content to target
	if err := os.WriteFile(targetPath, backup.Content, 0644); err != nil {
		return fmt.Errorf("writing restored file: %w", err)
	}

	return nil
}

// restoreDirectory restores a directory backup
func (bm *BackupManager) restoreDirectory(backup *Backup, targetPath string) error {
	// Read backup from disk
	backupPath := filepath.Join(bm.backupDir, backup.ID)
	
	// Copy all files from backup to target
	return filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(backupPath, path)
		if err != nil {
			return err
		}

		targetFile := filepath.Join(targetPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetFile, info.Mode())
		}

		return bm.copyFile(path, targetFile)
	})
}

// GetBackup retrieves a backup by ID
func (bm *BackupManager) GetBackup(backupID string) (*Backup, bool) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	backup, exists := bm.backups[backupID]
	return backup, exists
}

// ListBackups returns all backup IDs
func (bm *BackupManager) ListBackups() []string {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	var ids []string
	for id := range bm.backups {
		ids = append(ids, id)
	}
	return ids
}

// Cleanup removes a backup
func (bm *BackupManager) Cleanup(backupID string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	backup, exists := bm.backups[backupID]
	if !exists {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	// Remove from memory
	delete(bm.backups, backupID)

	// Remove from disk
	backupPath := filepath.Join(bm.backupDir, backupID)
	if backup.Type == DirectoryBackup {
		return os.RemoveAll(backupPath)
	} else {
		return os.Remove(backupPath + ".backup")
	}
}

// CleanupOld removes backups older than the specified duration
func (bm *BackupManager) CleanupOld(maxAge time.Duration) int {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var toDelete []string

	for id, backup := range bm.backups {
		if backup.CreatedAt.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(bm.backups, id)
		// Also remove from disk
		backup := bm.backups[id]
		backupPath := filepath.Join(bm.backupDir, id)
		if backup != nil && backup.Type == DirectoryBackup {
			os.RemoveAll(backupPath)
		} else {
			os.Remove(backupPath + ".backup")
		}
	}

	return len(toDelete)
}

// writeToDisk writes a file backup to disk for persistence
func (bm *BackupManager) writeToDisk(backup *Backup) error {
	backupPath := filepath.Join(bm.backupDir, backup.ID+".backup")
	return os.WriteFile(backupPath, backup.Content, 0644)
}

// writeDirectoryToDisk writes a directory backup to disk
func (bm *BackupManager) writeDirectoryToDisk(backup *Backup, sourceDir string) error {
	backupPath := filepath.Join(bm.backupDir, backup.ID)
	
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(backupPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return bm.copyFile(path, targetPath)
	})
}

// copyFile copies a file from source to destination
func (bm *BackupManager) copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}