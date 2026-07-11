package parquet

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	parquetgo "github.com/parquet-go/parquet-go"
)

type Format string

const (
	FormatParquet Format = "parquet"
	FormatJSONL   Format = "jsonl"
)

type Entry struct {
	ImagePath   string     `parquet:"image_path,snappy" json:"image_path"`
	Prompt      string     `parquet:"prompt,snappy" json:"prompt"`
	Description string     `parquet:"description,snappy" json:"description"`
	CreatedAt   time.Time  `parquet:"created_at" json:"created_at"`
	ModifiedAt  *time.Time `parquet:"modified_at" json:"modified_at,omitempty"`
}

type ParquetDB struct {
	Entries []Entry
	index   map[string]int
	Path    string
	Format  Format
}

func detectFormat(path string) Format {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jsonl" || ext == ".json" {
		return FormatJSONL
	}
	return FormatParquet
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
	db := &ParquetDB{
		Path:   path,
		Format: detectFormat(path),
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		db.buildIndex()
		return db, nil
	}

	var rows []Entry
	var err error
	if db.Format == FormatJSONL {
		rows, err = readJSONLFile(path)
	} else {
		rows, err = parquetgo.ReadFile[Entry](path)
	}
	if err != nil {
		return nil, err
	}

	db.Entries = rows
	db.buildIndex()
	return db, nil
}

func readJSONLFile(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	// Use a 10MB capacity buffer to prevent scanner token too long errors on large prompts
	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (db *ParquetDB) Save() error {
	tempPath := db.Path + ".tmp"
	ok := false
	defer func() {
		if !ok {
			os.Remove(tempPath)
		}
	}()

	var err error
	if db.Format == FormatJSONL {
		err = writeJSONLFile(tempPath, db.Entries)
	} else {
		err = parquetgo.WriteFile(tempPath, db.Entries)
	}
	if err != nil {
		return err
	}

	if err := os.Rename(tempPath, db.Path); err != nil {
		return err
	}
	ok = true
	return nil
}

func writeJSONLFile(path string, entries []Entry) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
		if _, err := writer.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return writer.Flush()
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

