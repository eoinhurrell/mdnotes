package safety

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupManager_Create(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.md")
	originalContent := []byte("# Original Content\n\nThis is the original content")
	err := os.WriteFile(testFile, originalContent, 0644)
	require.NoError(t, err)

	// Create backup
	backupID, err := manager.CreateBackup(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, backupID)

	// Verify backup exists
	backup, exists := manager.GetBackup(backupID)
	assert.True(t, exists)
	assert.Equal(t, testFile, backup.OriginalPath)
	assert.Equal(t, originalContent, backup.Content)
	assert.WithinDuration(t, time.Now(), backup.CreatedAt, 5*time.Second)

	// Modify original file
	modifiedContent := []byte("# Modified Content\n\nThis has been changed")
	err = os.WriteFile(testFile, modifiedContent, 0644)
	require.NoError(t, err)

	// Restore from backup
	err = manager.Restore(backupID, testFile)
	assert.NoError(t, err)

	// Verify restored content
	restored, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, restored)
}

func TestBackupManager_CreateMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create multiple test files
	testFiles := []string{"file1.md", "file2.md", "file3.md"}
	var backupIDs []string

	for i, filename := range testFiles {
		testFile := filepath.Join(tmpDir, filename)
		content := []byte(filename + " content")
		err := os.WriteFile(testFile, content, 0644)
		require.NoError(t, err)

		backupID, err := manager.CreateBackup(testFile)
		require.NoError(t, err)
		backupIDs = append(backupIDs, backupID)

		// Each backup should have unique ID
		for j := 0; j < i; j++ {
			assert.NotEqual(t, backupIDs[j], backupID)
		}
	}

	// Verify all backups exist
	assert.Len(t, manager.ListBackups(), 3)
}

func TestBackupManager_RestoreNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	err := manager.Restore("nonexistent-backup", "/some/file")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup not found")
}

func TestBackupManager_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create test file and backup
	testFile := filepath.Join(tmpDir, "test.md")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	backupID, err := manager.CreateBackup(testFile)
	require.NoError(t, err)

	// Verify backup exists
	_, exists := manager.GetBackup(backupID)
	assert.True(t, exists)

	// Cleanup
	err = manager.Cleanup(backupID)
	assert.NoError(t, err)

	// Verify backup is gone
	_, exists = manager.GetBackup(backupID)
	assert.False(t, exists)
}

func TestBackupManager_CleanupOld(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.md")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	// Create backup
	backupID, err := manager.CreateBackup(testFile)
	require.NoError(t, err)

	// Manually set creation time to old
	backup, _ := manager.GetBackup(backupID)
	backup.CreatedAt = time.Now().Add(-25 * time.Hour) // Older than 24 hours

	// Cleanup old backups (older than 24 hours)
	cleaned := manager.CleanupOld(24 * time.Hour)
	assert.Equal(t, 1, cleaned)

	// Verify backup is gone
	_, exists := manager.GetBackup(backupID)
	assert.False(t, exists)
}

func TestBackupManager_DuplicateFile(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.md")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	// Create first backup
	backupID1, err := manager.CreateBackup(testFile)
	require.NoError(t, err)

	// Create second backup of same file
	backupID2, err := manager.CreateBackup(testFile)
	require.NoError(t, err)

	// Should have different IDs
	assert.NotEqual(t, backupID1, backupID2)

	// Both should exist
	_, exists1 := manager.GetBackup(backupID1)
	_, exists2 := manager.GetBackup(backupID2)
	assert.True(t, exists1)
	assert.True(t, exists2)
}

func TestBackupManager_BackupDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create test directory structure
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(subDir, "file2.md")

	err = os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	// Backup directory
	backupID, err := manager.CreateDirectoryBackup(tmpDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, backupID)

	// Verify backup exists
	backup, exists := manager.GetBackup(backupID)
	assert.True(t, exists)
	assert.Equal(t, tmpDir, backup.OriginalPath)
	assert.Contains(t, backup.Metadata, "files")
}

func TestBackupManager_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Try to backup non-existent file
	_, err := manager.CreateBackup("/nonexistent/file.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestBackupManager_RestoreToNonExistentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create test file and backup
	testFile := filepath.Join(tmpDir, "test.md")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	backupID, err := manager.CreateBackup(testFile)
	require.NoError(t, err)

	// Try to restore to non-existent directory
	nonExistentFile := filepath.Join(tmpDir, "nonexistent", "dir", "file.md")
	err = manager.Restore(backupID, nonExistentFile)
	assert.NoError(t, err) // Should create directories

	// Verify file was restored
	restored, err := os.ReadFile(nonExistentFile)
	require.NoError(t, err)
	assert.Equal(t, content, restored)
}
