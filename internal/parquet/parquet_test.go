package parquet

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParquetSaveLoad(t *testing.T) {
	tempFile := "test.parquet"
	defer os.Remove(tempFile)

	now := time.Now().Truncate(time.Millisecond)
	entries := []Entry{
		{
			ImagePath:   "test1.png",
			Prompt:      "prompt1",
			Description: "desc1",
			CreatedAt:   now,
		},
		{
			ImagePath:   "test2.png",
			Prompt:      "prompt2",
			Description: "desc2",
			CreatedAt:   now,
			ModifiedAt:  &now,
		},
	}

	db := &ParquetDB{
		Entries: entries,
		Path:    tempFile,
	}

	err := db.Save(tempFile)
	assert.NoError(t, err)

	loadedDB, err := LoadParquetDB(tempFile)
	assert.NoError(t, err)

	assert.Len(t, loadedDB.Entries, 2)
	assert.Equal(t, "test1.png", loadedDB.Entries[0].ImagePath)
	assert.Equal(t, "prompt1", loadedDB.Entries[0].Prompt)
	assert.Equal(t, "desc1", loadedDB.Entries[0].Description)
	assert.True(t, now.Equal(loadedDB.Entries[0].CreatedAt))
	assert.Nil(t, loadedDB.Entries[0].ModifiedAt)

	assert.Equal(t, "test2.png", loadedDB.Entries[1].ImagePath)
	assert.True(t, now.Equal(*loadedDB.Entries[1].ModifiedAt))
}
