package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExportAssetHandler(t *testing.T) {
	handler := NewExportAssetHandler("/vault", "/output", true)

	assert.Equal(t, "/vault", handler.vaultPath)
	assert.Equal(t, "/output", handler.outputPath)
	assert.True(t, handler.verbose)
	assert.Greater(t, len(handler.supportedExtensions), 10) // Should have many supported extensions
}

func TestAssetHandler_ResolveAssetPath(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "asset-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test asset files
	assetDir := filepath.Join(tmpDir, "assets")
	require.NoError(t, os.MkdirAll(assetDir, 0755))

	testImage := filepath.Join(assetDir, "test.png")
	require.NoError(t, os.WriteFile(testImage, []byte("fake image"), 0644))

	sameDirImage := filepath.Join(tmpDir, "notes", "image.jpg")
	require.NoError(t, os.MkdirAll(filepath.Dir(sameDirImage), 0755))
	require.NoError(t, os.WriteFile(sameDirImage, []byte("fake image"), 0644))

	handler := NewExportAssetHandler(tmpDir, "/output", false)

	tests := []struct {
		name               string
		target             string
		sourceRelativePath string
		expected           string
	}{
		{
			name:               "Asset in assets directory",
			target:             "assets/test.png",
			sourceRelativePath: "notes/note.md",
			expected:           "assets/test.png",
		},
		{
			name:               "Asset in same directory",
			target:             "image.jpg",
			sourceRelativePath: "notes/note.md",
			expected:           "notes/image.jpg",
		},
		{
			name:               "Absolute path",
			target:             "/assets/test.png",
			sourceRelativePath: "notes/note.md",
			expected:           "assets/test.png",
		},
		{
			name:               "Relative path with ../",
			target:             "../assets/test.png",
			sourceRelativePath: "notes/note.md",
			expected:           "assets/test.png",
		},
		{
			name:               "Unsupported file type",
			target:             "document.xyz",
			sourceRelativePath: "notes/note.md",
			expected:           "", // Not a supported asset type
		},
		{
			name:               "Asset with fragment",
			target:             "assets/test.png#section",
			sourceRelativePath: "notes/note.md",
			expected:           "assets/test.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.resolveAssetPath(tt.target, tt.sourceRelativePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAssetHandler_DiscoverAssets(t *testing.T) {
	// Create temp vault
	tmpDir, err := os.MkdirTemp("", "asset-discover-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test assets
	assetDir := filepath.Join(tmpDir, "assets")
	require.NoError(t, os.MkdirAll(assetDir, 0755))

	existingImage := filepath.Join(assetDir, "existing.png")
	require.NoError(t, os.WriteFile(existingImage, []byte("fake image"), 0644))

	documentPDF := filepath.Join(assetDir, "document.pdf")
	require.NoError(t, os.WriteFile(documentPDF, []byte("fake pdf"), 0644))

	// Create notes directory and an image there to test resolution priority
	notesDir := filepath.Join(tmpDir, "notes")
	require.NoError(t, os.MkdirAll(notesDir, 0755))

	notesImage := filepath.Join(notesDir, "existing.png")
	require.NoError(t, os.WriteFile(notesImage, []byte("fake image in notes"), 0644))

	handler := NewExportAssetHandler(tmpDir, "/output", false)

	note1Path := filepath.Join(notesDir, "note1.md")
	note1Content := `# Note 1
			
![Existing Image](../assets/existing.png)
![[missing.jpg]]
![External](https://example.com/image.png)`
	require.NoError(t, os.WriteFile(note1Path, []byte(note1Content), 0644))

	note2Path := filepath.Join(notesDir, "note2.md")
	note2Content := `# Note 2
			
![[existing.png]] - should resolve to assets/existing.png
![PDF](../assets/document.pdf)`
	require.NoError(t, os.WriteFile(note2Path, []byte(note2Content), 0644))

	// Create test files with asset links
	files := []*vault.VaultFile{
		{
			Path:         note1Path,
			RelativePath: "notes/note1.md",
			Body:         note1Content,
		},
		{
			Path:         note2Path,
			RelativePath: "notes/note2.md",
			Body:         note2Content,
		},
	}

	result := handler.DiscoverAssets(files)

	// Should find both existing images and the PDF
	assert.Contains(t, result.AssetFiles, "assets/existing.png")
	assert.Equal(t, "notes/note1.md", result.AssetFiles["assets/existing.png"])

	assert.Contains(t, result.AssetFiles, "notes/existing.png")
	assert.Equal(t, "notes/note2.md", result.AssetFiles["notes/existing.png"])

	assert.Contains(t, result.AssetFiles, "assets/document.pdf")
	assert.Equal(t, "notes/note2.md", result.AssetFiles["assets/document.pdf"])

	// Should track missing assets
	assert.Contains(t, result.MissingAssets, "notes/missing.jpg")
	assert.Equal(t, 1, len(result.MissingAssets), "Should have exactly one missing asset")

	// Should not include external URLs or unsupported types
	assert.NotContains(t, result.AssetFiles, "https://example.com/image.png")

	assert.Greater(t, result.TotalAssets, 0)
}

func TestAssetHandler_ProcessAssets(t *testing.T) {
	// Create temp directories
	vaultDir, err := os.MkdirTemp("", "vault-*")
	require.NoError(t, err)
	defer os.RemoveAll(vaultDir)

	outputDir, err := os.MkdirTemp("", "output-*")
	require.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create test assets in vault
	assetDir := filepath.Join(vaultDir, "assets")
	require.NoError(t, os.MkdirAll(assetDir, 0755))

	testImagePath := filepath.Join(assetDir, "test.png")
	testImageContent := []byte("fake image content")
	require.NoError(t, os.WriteFile(testImagePath, testImageContent, 0644))

	handler := NewExportAssetHandler(vaultDir, outputDir, false)

	// Create discovery result
	discovery := &AssetDiscoveryResult{
		AssetFiles: map[string]string{
			"assets/test.png": "notes/note.md",
		},
		MissingAssets: []string{"assets/missing.jpg"},
		TotalAssets:   2,
	}

	result := handler.ProcessAssets(discovery)

	// Should have copied one asset and noted one missing
	assert.Equal(t, 1, result.AssetsCopied)
	assert.Equal(t, 1, result.AssetsMissing)
	assert.Contains(t, result.CopiedAssets, "assets/test.png")
	assert.Contains(t, result.MissingAssets, "assets/missing.jpg")

	// Check that asset was actually copied
	copiedAssetPath := filepath.Join(outputDir, "assets", "test.png")
	assert.FileExists(t, copiedAssetPath)

	copiedContent, err := os.ReadFile(copiedAssetPath)
	require.NoError(t, err)
	assert.Equal(t, testImageContent, copiedContent)
}

func TestAssetHandler_CopyAssetFile(t *testing.T) {
	// Create temp directories
	srcDir, err := os.MkdirTemp("", "src-*")
	require.NoError(t, err)
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "dst-*")
	require.NoError(t, err)
	defer os.RemoveAll(dstDir)

	// Create source file
	srcFile := filepath.Join(srcDir, "test.png")
	srcContent := []byte("test image content")
	require.NoError(t, os.WriteFile(srcFile, srcContent, 0644))

	handler := NewExportAssetHandler("", "", false)

	// Test copying
	dstFile := filepath.Join(dstDir, "subdir", "test.png")
	err = handler.copyAssetFile(srcFile, dstFile)
	require.NoError(t, err)

	// Verify file was copied
	assert.FileExists(t, dstFile)

	dstContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcContent, dstContent)

	// Verify directory was created
	assert.DirExists(t, filepath.Dir(dstFile))
}

func TestAssetHandler_AssetExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "exists-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "assets", "test.png")
	require.NoError(t, os.MkdirAll(filepath.Dir(testFile), 0755))
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	handler := NewExportAssetHandler(tmpDir, "", false)

	// Test existing file
	assert.True(t, handler.assetExists("assets/test.png"))

	// Test non-existing file
	assert.False(t, handler.assetExists("assets/missing.png"))
	assert.False(t, handler.assetExists("nonexistent/path.png"))
}

func TestAssetHandler_SupportedExtensions(t *testing.T) {
	handler := NewExportAssetHandler("", "", false)

	// Test image extensions
	assert.Contains(t, handler.supportedExtensions, ".png")
	assert.Contains(t, handler.supportedExtensions, ".jpg")
	assert.Contains(t, handler.supportedExtensions, ".jpeg")
	assert.Contains(t, handler.supportedExtensions, ".gif")
	assert.Contains(t, handler.supportedExtensions, ".svg")

	// Test document extensions
	assert.Contains(t, handler.supportedExtensions, ".pdf")
	assert.Contains(t, handler.supportedExtensions, ".docx")
	assert.Contains(t, handler.supportedExtensions, ".xlsx")

	// Test media extensions
	assert.Contains(t, handler.supportedExtensions, ".mp4")
	assert.Contains(t, handler.supportedExtensions, ".mp3")

	// Test archive extensions
	assert.Contains(t, handler.supportedExtensions, ".zip")
	assert.Contains(t, handler.supportedExtensions, ".tar")
}
