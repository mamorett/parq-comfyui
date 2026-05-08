package parquet

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParquetSaveLoad(t *testing.T) {
	tempFile := "/tmp/test_basic.parquet"
	os.Remove(tempFile)
	os.Remove(tempFile + ".tmp")

	now := time.Now().Truncate(time.Millisecond)
	entries := []Entry{
		{ImagePath: "test1.png", Prompt: "prompt1", Description: "desc1", CreatedAt: now},
		{ImagePath: "test2.png", Prompt: "prompt2", Description: "desc2", CreatedAt: now, ModifiedAt: &now},
	}

	db := &ParquetDB{Entries: entries, Path: tempFile}
	db.buildIndex()

	err := db.Save()
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
	assert.NotNil(t, loadedDB.Entries[1].ModifiedAt)
	assert.True(t, now.Equal(*loadedDB.Entries[1].ModifiedAt))
}

func TestParquetDBIndex(t *testing.T) {
	db := &ParquetDB{Path: "test.parquet"}
	db.Entries = []Entry{
		{ImagePath: "a.png", Prompt: "a"},
		{ImagePath: "b.png", Prompt: "b"},
	}
	db.buildIndex()

	assert.True(t, db.Exists("a.png"))
	assert.True(t, db.Exists("b.png"))
	assert.False(t, db.Exists("c.png"))

	e, ok := db.GetEntry("a.png")
	assert.True(t, ok)
	assert.Equal(t, "a", e.Prompt)
}

func TestParquetDBAddEntries(t *testing.T) {
	db := &ParquetDB{}
	db.Entries = []Entry{
		{ImagePath: "a.png", Prompt: "old"},
	}
	db.buildIndex()

	now := time.Now()
	db.AddEntries([]Entry{{ImagePath: "b.png", Prompt: "new"}}, false)
	assert.Len(t, db.Entries, 2)
	assert.True(t, db.Exists("b.png"))

	db.AddEntries([]Entry{{ImagePath: "a.png", Prompt: "updated", CreatedAt: now}}, true)
	assert.Len(t, db.Entries, 2)
	e, _ := db.GetEntry("a.png")
	assert.Equal(t, "updated", e.Prompt)
	assert.True(t, now.Equal(e.CreatedAt))
}

func TestParquetSaveLoadEmpty(t *testing.T) {
	tempFile := "/tmp/test_empty.parquet"
	os.Remove(tempFile)
	os.Remove(tempFile + ".tmp")

	db := &ParquetDB{Path: tempFile}
	err := db.Save()
	assert.NoError(t, err)

	loadedDB, err := LoadParquetDB(tempFile)
	assert.NoError(t, err)
	assert.Len(t, loadedDB.Entries, 0)
}

func TestParquetSaveLoadNullable(t *testing.T) {
	tempFile := "/tmp/test_nullable.parquet"
	os.Remove(tempFile)
	os.Remove(tempFile + ".tmp")

	now := time.Now().Truncate(time.Millisecond)
	entries := []Entry{
		{ImagePath: "no_mod.png", Prompt: "test", Description: "", CreatedAt: now},
		{ImagePath: "with_mod.png", Prompt: "", Description: "desc", CreatedAt: now, ModifiedAt: &now},
	}

	db := &ParquetDB{Entries: entries, Path: tempFile}
	db.buildIndex()
	assert.NoError(t, db.Save())

	loadedDB, err := LoadParquetDB(tempFile)
	assert.NoError(t, err)
	assert.Len(t, loadedDB.Entries, 2)

	e0, ok := loadedDB.GetEntry("no_mod.png")
	assert.True(t, ok)
	assert.Equal(t, "test", e0.Prompt)
	assert.Empty(t, e0.Description)
	assert.Nil(t, e0.ModifiedAt)

	e1, ok := loadedDB.GetEntry("with_mod.png")
	assert.True(t, ok)
	assert.Empty(t, e1.Prompt)
	assert.Equal(t, "desc", e1.Description)
	assert.NotNil(t, e1.ModifiedAt)
	assert.True(t, now.Equal(*e1.ModifiedAt))
}

func TestRemoveMissingEntries(t *testing.T) {
	existingFile := "/tmp/test_clean_existing.png"
	os.Remove(existingFile)

	f, err := os.Create(existingFile)
	require.NoError(t, err)
	f.Close()
	defer os.Remove(existingFile)

	now := time.Now().Truncate(time.Millisecond)
	db := &ParquetDB{Path: "/tmp/test_clean.parquet"}
	db.Entries = []Entry{
		{ImagePath: existingFile, Prompt: "exists", CreatedAt: now},
		{ImagePath: "/tmp/test_clean_missing.png", Prompt: "missing", CreatedAt: now},
		{ImagePath: "/tmp/test_clean_missing2.png", Prompt: "missing2", CreatedAt: now, ModifiedAt: &now},
	}
	db.buildIndex()
	assert.Len(t, db.Entries, 3)

	removed := db.RemoveMissingEntries()
	assert.Equal(t, 2, removed)
	assert.Len(t, db.Entries, 1)
	assert.Equal(t, existingFile, db.Entries[0].ImagePath)
	assert.True(t, db.Exists(existingFile))
	assert.False(t, db.Exists("/tmp/test_clean_missing.png"))
	assert.False(t, db.Exists("/tmp/test_clean_missing2.png"))
}

func TestRemoveMissingEntriesNoneMissing(t *testing.T) {
	existingFile := "/tmp/test_clean_none_missing.png"
	os.Remove(existingFile)
	f, err := os.Create(existingFile)
	require.NoError(t, err)
	f.Close()
	defer os.Remove(existingFile)

	now := time.Now().Truncate(time.Millisecond)
	db := &ParquetDB{}
	db.Entries = []Entry{
		{ImagePath: existingFile, Prompt: "a", CreatedAt: now},
	}
	db.buildIndex()

	removed := db.RemoveMissingEntries()
	assert.Equal(t, 0, removed)
	assert.Len(t, db.Entries, 1)
}

func TestParquetMultiCycle(t *testing.T) {
	tempFile := "/tmp/test_multicycle.parquet"
	os.Remove(tempFile)
	os.Remove(tempFile + ".tmp")

	now := time.Now().Truncate(time.Millisecond)

	db := &ParquetDB{Path: tempFile}
	for i := 0; i < 50; i++ {
		entry := Entry{
			ImagePath:   fmt.Sprintf("img_%02d.png", i),
			Prompt:      fmt.Sprintf("prompt %d", i),
			Description: fmt.Sprintf("desc %d", i),
			CreatedAt:   now.Add(time.Duration(i) * time.Second),
		}
		if i%2 == 0 {
			mt := now.Add(time.Duration(i) * time.Hour)
			entry.ModifiedAt = &mt
		}
		db.Entries = append(db.Entries, entry)
	}
	db.buildIndex()
	require.NoError(t, db.Save())

	loaded, err := LoadParquetDB(tempFile)
	require.NoError(t, err)
	assert.Len(t, loaded.Entries, 50)

	loaded.AddEntries([]Entry{
		{ImagePath: "new_a.png", Prompt: "new a", Description: "new desc a", CreatedAt: now},
		{ImagePath: "new_b.png", Prompt: "", Description: "new desc b", CreatedAt: now, ModifiedAt: &now},
	}, false)
	require.NoError(t, loaded.Save())

	loaded2, err := LoadParquetDB(tempFile)
	require.NoError(t, err)
	assert.Len(t, loaded2.Entries, 52)

	loaded2.AddEntries([]Entry{
		{ImagePath: "new_a.png", Prompt: "OVERRIDDEN", Description: "overridden a", CreatedAt: now.Add(time.Hour)},
		{ImagePath: "new_b.png", Prompt: "OVERRIDDEN", Description: "overridden b", CreatedAt: now.Add(time.Hour)},
	}, true)
	require.NoError(t, loaded2.Save())

	final, err := LoadParquetDB(tempFile)
	require.NoError(t, err)
	assert.Len(t, final.Entries, 52)

	e, ok := final.GetEntry("new_a.png")
	assert.True(t, ok)
	assert.Equal(t, "OVERRIDDEN", e.Prompt)
}

func TestParquetStressLarge(t *testing.T) {
	tempFile := "/tmp/test_stress_large.parquet"
	os.Remove(tempFile)
	os.Remove(tempFile + ".tmp")

	now := time.Now().Truncate(time.Millisecond)
	n := 30000
	db := &ParquetDB{Path: tempFile}
	for i := 0; i < n; i++ {
		entry := Entry{
			ImagePath:   fmt.Sprintf("/wdblack/ARS/dgxcomfy/2025-12-04/image_%05d.png", i),
			Prompt:      fmt.Sprintf("prompt text number %d with unicode: 🎨 日本語 émoji café", i),
			Description: fmt.Sprintf("description %d: a beautiful painting of %d sunflowers, 8k, masterpiece", i, i),
			CreatedAt:   now.Add(time.Duration(i) * time.Second),
		}
		if i%2 == 0 {
			mt := now.Add(time.Duration(i) * time.Hour)
			entry.ModifiedAt = &mt
		}
		db.Entries = append(db.Entries, entry)
	}
	db.buildIndex()

	require.NoError(t, db.Save())

	loaded, err := LoadParquetDB(tempFile)
	require.NoError(t, err)
	require.Len(t, loaded.Entries, n, "loaded %d entries, expected %d", len(loaded.Entries), n)

	e0, ok := loaded.GetEntry(fmt.Sprintf("/wdblack/ARS/dgxcomfy/2025-12-04/image_%05d.png", 0))
	require.True(t, ok)
	require.Contains(t, e0.Prompt, "prompt text number 0")
}
