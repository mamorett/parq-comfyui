# 🎨 parq-comfyui

[![Go Report Card](https://goreportcard.com/badge/github.com/trithemius/parq-comfyui)](https://goreportcard.com/report/github.com/trithemius/parq-comfyui)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**parq-comfyui** is a high-performance Go-based utility designed to extract positive prompts and metadata from ComfyUI and Automatic1111 generated PNG files and store them in a structured Parquet database.

Built for efficiency, it handles large datasets of images with ease, resolving complex ComfyUI node references and providing a robust, searchable archive of your generative art prompts.

---

## ✨ Features

- **🚀 High Performance:** Written in Go for lightning-fast PNG metadata parsing and Parquet serialization.
- **🧩 Deep ComfyUI Support:** Resolves node references in `prompt` JSON (handling `CLIPTextEncode`, `String`, `KepStringLiteral`, and more).
- **📝 A1111 Compatibility:** Robust extraction from standard `parameters` chunks.
- **📊 Parquet Storage:** Efficiently saves to Apache Parquet using Apache Arrow, making it easy to query with DuckDB, Pandas, or Spark.
- **🎯 Idempotency:** Automatically skips already-processed images unless `--override` is specified.
- **📦 Cross-Platform:** Pre-compiled binaries for Linux (amd64/arm64) and macOS (arm64).
- **🛑 Graceful Shutdown:** Press `Ctrl+C` at any time; the application will save current progress to the database before exiting.

---

## 🛠 Installation & Setup

### Prerequisites
- **Go**: Version 1.24+ is recommended (as specified in `go.mod`). [Download Go](https://go.dev/doc/install).

### 🛠 Installing gödel
This project uses [gödel](https://github.com/palantir/godel), a powerful build tool for Go. You have two options for using it:

#### 1. Using the project wrapper (Recommended)
You do not need to install gödel globally. The repository includes a `godelw` (gödel wrapper) script that automatically downloads and manages the correct version of gödel for this project.

To initialize gödel and check the version:
```bash
chmod +x godelw
./godelw version
```

#### 2. Installing the gödel CLI (Optional)
If you wish to use gödel in your own projects, you can install the `godelinit` tool to set up the wrapper in any repository:
```bash
go install github.com/palantir/godel/v2/godelinit@latest
```
Running `godelinit` in a project root will add the `godelw` script and the `godel/` configuration directory.

### 🏗 Building from Source
1. **Clone the repository:**
   ```bash
   git clone https://github.com/trithemius/parq-comfyui.git
   cd parq-comfyui
   ```

2. **Build the application:**
   ```bash
   ./godelw build
   ```
   The compiled binaries will be available in `out/build/parq-comfyui/unspecified/`.

---

## 🚀 Usage

### Basic Extraction
Process all PNG files in a directory and save them to a database:
```bash
parq-comfyui -i /path/to/my/renders --database prompts.parquet
```

### Advanced Options
| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--input` | `-i` | Input directory, single file, or glob pattern (e.g., `*.png`) |
| `--database` | `--db` | Path to the Parquet database file (Required) |
| `--recursive` | `-r` | Search subdirectories recursively |
| `--file-list` | `-f` | Process files listed in a text file |
| `--override` | | Reprocess and update existing entries in the database |
| `--use-parameters`| | Force A1111-style parameter extraction |

### Examples

**Recursive search in subdirectories:**
```bash
parq-comfyui -i ./outputs -r --db gallery.parquet
```

**Using a file list:**
```bash
parq-comfyui -f my_best_images.txt --db favorites.parquet
```

**Overriding existing data:**
```bash
parq-comfyui -i ./latest -db prompts.parquet --override
```

---

## 🏗 Project Structure

- `cmd/parq-comfyui`: Application entry point and CLI handling.
- `internal/pngreader`: Custom PNG chunk parser for `tEXt`, `zTXt`, and `iTXt`.
- `internal/extractor`: Heuristics for ComfyUI and A1111 prompt extraction.
- `internal/parquet`: Parquet I/O logic using Apache Arrow.
- `internal/collector`: File system discovery and path resolution.

---

## 🧪 Testing

Run the test suite to verify extraction and database logic:

```bash
./godelw test
```

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

*Generated with ❤️ by Gemini CLI.*
