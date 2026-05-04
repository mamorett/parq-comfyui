package collector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectImages(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collector_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create some dummy PNG files
	files := []string{"test1.png", "test2.PNG", "not_an_image.txt"}
	for _, name := range files {
		err := os.WriteFile(filepath.Join(tempDir, name), []byte("dummy"), 0644)
		assert.NoError(t, err)
	}

	// Test directory collection
	collected, err := CollectImages(tempDir, false, "")
	assert.NoError(t, err)
	assert.Len(t, collected, 2)

	// Test recursive collection
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "sub.png"), []byte("dummy"), 0644)
	assert.NoError(t, err)

	collected, err = CollectImages(tempDir, true, "")
	assert.NoError(t, err)
	assert.Len(t, collected, 3)

	// Test file list collection
	listFile := filepath.Join(tempDir, "list.txt")
	err = os.WriteFile(listFile, []byte(filepath.Join(tempDir, "test1.png")+"\n"+filepath.Join(subDir, "sub.png")), 0644)
	assert.NoError(t, err)

	collected, err = CollectImages("", false, listFile)
	assert.NoError(t, err)
	assert.Len(t, collected, 2)
}
