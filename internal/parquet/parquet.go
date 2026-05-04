package parquet

import (
	"context"
	"os"
	"time"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/apache/arrow/go/v18/parquet"
	"github.com/apache/arrow/go/v18/parquet/compress"
	"github.com/apache/arrow/go/v18/parquet/file"
	"github.com/apache/arrow/go/v18/parquet/pqarrow"
)

// Entry represents a database row
type Entry struct {
	ImagePath   string
	Prompt      string
	Description string
	CreatedAt   time.Time
	ModifiedAt  *time.Time
}

// ParquetDB represents the Parquet database
type ParquetDB struct {
	Entries []Entry
	Path    string
}

// LoadParquetDB loads existing Parquet database
func LoadParquetDB(path string) (*ParquetDB, error) {
	db := &ParquetDB{Path: path}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return db, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rdr, err := file.NewParquetReader(f)
	if err != nil {
		return nil, err
	}
	defer rdr.Close()

	reader, err := pqarrow.NewFileReader(rdr, pqarrow.ArrowReadProperties{BatchSize: 1024}, memory.DefaultAllocator)
	if err != nil {
		return nil, err
	}

	table, err := reader.ReadTable(context.Background())
	if err != nil {
		return nil, err
	}
	defer table.Release()

	// Parse table into Entries
	tr := array.NewTableReader(table, 1024)
	defer tr.Release()

	for tr.Next() {
		rec := tr.Record()
		for i := 0; i < int(rec.NumRows()); i++ {
			entry := Entry{}
			for j := 0; j < int(rec.NumCols()); j++ {
				colName := table.Schema().Field(j).Name
				col := rec.Column(j)
				
				if col.IsNull(i) {
					continue
				}

				switch colName {
				case "image_path":
					entry.ImagePath = col.(*array.String).Value(i)
				case "prompt":
					entry.Prompt = col.(*array.String).Value(i)
				case "description":
					entry.Description = col.(*array.String).Value(i)
				case "created_at":
					ts := col.(*array.Timestamp).Value(i)
					entry.CreatedAt = time.Unix(0, int64(ts)*int64(time.Millisecond))
				case "modified_at":
					ts := col.(*array.Timestamp).Value(i)
					t := time.Unix(0, int64(ts)*int64(time.Millisecond))
					entry.ModifiedAt = &t
				}
			}
			db.Entries = append(db.Entries, entry)
		}
	}

	return db, nil
}

// Save writes the database to Parquet
func (db *ParquetDB) Save(path string) error {
	schema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "image_path", Type: arrow.BinaryTypes.String},
			{Name: "prompt", Type: arrow.BinaryTypes.String},
			{Name: "description", Type: arrow.BinaryTypes.String},
			{Name: "created_at", Type: arrow.FixedWidthTypes.Timestamp_ms},
			{Name: "modified_at", Type: arrow.FixedWidthTypes.Timestamp_ms, Nullable: true},
		},
		nil,
	)

	pool := memory.NewGoAllocator()
	builder := array.NewRecordBuilder(pool, schema)
	defer builder.Release()

	for _, entry := range db.Entries {
		builder.Field(0).(*array.StringBuilder).Append(entry.ImagePath)
		builder.Field(1).(*array.StringBuilder).Append(entry.Prompt)
		builder.Field(2).(*array.StringBuilder).Append(entry.Description)
		builder.Field(3).(*array.TimestampBuilder).Append(arrow.Timestamp(entry.CreatedAt.UnixNano() / int64(time.Millisecond)))
		if entry.ModifiedAt != nil {
			builder.Field(4).(*array.TimestampBuilder).Append(arrow.Timestamp(entry.ModifiedAt.UnixNano() / int64(time.Millisecond)))
		} else {
			builder.Field(4).(*array.TimestampBuilder).AppendNull()
		}
	}

	rec := builder.NewRecord()
	defer rec.Release()

	tbl := array.NewTableFromRecords(schema, []arrow.Record{rec})
	defer tbl.Release()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer, err := pqarrow.NewFileWriter(tbl.Schema(), f, parquet.NewWriterProperties(parquet.WithCompression(compress.Codecs.Snappy)), pqarrow.DefaultWriterProps())
	if err != nil {
		return err
	}

	if err := writer.WriteTable(tbl, 1024); err != nil {
		return err
	}

	return writer.Close()
}

// Exists checks if an image path is already in the database
func (db *ParquetDB) Exists(imagePath string) bool {
	for _, entry := range db.Entries {
		if entry.ImagePath == imagePath {
			return true
		}
	}
	return false
}

// GetEntry retrieves an entry by image path
func (db *ParquetDB) GetEntry(imagePath string) (Entry, bool) {
	for _, entry := range db.Entries {
		if entry.ImagePath == imagePath {
			return entry, true
		}
	}
	return Entry{}, false
}

// AddEntries adds or updates entries in the database
func (db *ParquetDB) AddEntries(newEntries []Entry, override bool) {
	if !override {
		db.Entries = append(db.Entries, newEntries...)
		return
	}

	// Simple override: replace existing or append
	for _, newEntry := range newEntries {
		found := false
		for i, existing := range db.Entries {
			if existing.ImagePath == newEntry.ImagePath {
				db.Entries[i] = newEntry
				found = true
				break
			}
		}
		if !found {
			db.Entries = append(db.Entries, newEntry)
		}
	}
}
