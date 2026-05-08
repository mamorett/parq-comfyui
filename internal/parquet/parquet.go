package parquet

import (
	"os"
	"time"

	parquetgo "github.com/parquet-go/parquet-go"
)

type Entry struct {
	ImagePath   string     `parquet:"image_path,snappy"`
	Prompt      string     `parquet:"prompt,snappy"`
	Description string     `parquet:"description,snappy"`
	CreatedAt   time.Time  `parquet:"created_at"`
	ModifiedAt  *time.Time `parquet:"modified_at"`
}

type ParquetDB struct {
	Entries []Entry
	index   map[string]int
	Path    string
}

func (db *ParquetDB) buildIndex() {
	db.index = make(map[string]int, len(db.Entries))
	for i, entry := range db.Entries {
		db.index[entry.ImagePath] = i
	}
}

func (db *ParquetDB) BuildIndexPublic() {
	db.buildIndex()
}

func LoadParquetDB(path string) (*ParquetDB, error) {
	db := &ParquetDB{Path: path}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		db.buildIndex()
		return db, nil
	}

	rows, err := parquetgo.ReadFile[Entry](path)
	if err != nil {
		return nil, err
	}

	db.Entries = rows
	db.buildIndex()
	return db, nil
}

func (db *ParquetDB) Save() error {
	tempPath := db.Path + ".tmp"
	ok := false
	defer func() {
		if !ok {
			os.Remove(tempPath)
		}
	}()

	if err := parquetgo.WriteFile(tempPath, db.Entries); err != nil {
		return err
	}

	if err := os.Rename(tempPath, db.Path); err != nil {
		return err
	}
	ok = true
	return nil
}

func (db *ParquetDB) Exists(imagePath string) bool {
	_, ok := db.index[imagePath]
	return ok
}

func (db *ParquetDB) GetEntry(imagePath string) (Entry, bool) {
	idx, ok := db.index[imagePath]
	if !ok {
		return Entry{}, false
	}
	return db.Entries[idx], true
}

func (db *ParquetDB) RemoveMissingEntries() int {
	var kept []Entry
	removed := 0
	for _, e := range db.Entries {
		if _, err := os.Stat(e.ImagePath); os.IsNotExist(err) {
			removed++
		} else {
			kept = append(kept, e)
		}
	}
	db.Entries = kept
	db.buildIndex()
	return removed
}

func (db *ParquetDB) AddEntries(newEntries []Entry, override bool) {
	if !override {
		for _, e := range newEntries {
			db.index[e.ImagePath] = len(db.Entries)
			db.Entries = append(db.Entries, e)
		}
		return
	}

	for _, newEntry := range newEntries {
		if idx, ok := db.index[newEntry.ImagePath]; ok {
			db.Entries[idx] = newEntry
		} else {
			db.index[newEntry.ImagePath] = len(db.Entries)
			db.Entries = append(db.Entries, newEntry)
		}
	}
}
