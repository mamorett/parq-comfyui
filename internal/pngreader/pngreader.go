package pngreader

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
)

// Metadata map of PNG text chunks
type Metadata map[string]string

// ReadPNGMetadata reads PNG metadata (tEXt, zTXt, iTXt chunks) and image config
func ReadPNGMetadata(path string) (Metadata, image.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, image.Config{}, err
	}
	defer file.Close()

	// Read image config for width/height
	config, err := png.DecodeConfig(file)
	if err != nil {
		return nil, image.Config{}, fmt.Errorf("error decoding PNG config: %v", err)
	}

	// Seek back to start to read chunks
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, image.Config{}, err
	}

	// Verify PNG signature
	var signature [8]byte
	if _, err := io.ReadFull(file, signature[:]); err != nil {
		return nil, image.Config{}, err
	}
	if string(signature[:]) != "\x89PNG\r\n\x1a\n" {
		return nil, image.Config{}, fmt.Errorf("not a valid PNG file")
	}

	metadata := make(Metadata)

	for {
		var length uint32
		if err := binary.Read(file, binary.BigEndian, &length); err != nil {
			if err == io.EOF {
				break
			}
			return nil, image.Config{}, err
		}

		var chunkType [4]byte
		if _, err := io.ReadFull(file, chunkType[:]); err != nil {
			return nil, image.Config{}, err
		}

		chunkData := make([]byte, length)
		if _, err := io.ReadFull(file, chunkData); err != nil {
			return nil, image.Config{}, err
		}

		var crc uint32
		if err := binary.Read(file, binary.BigEndian, &crc); err != nil {
			return nil, image.Config{}, err
		}

		ctype := string(chunkType[:])
		switch ctype {
		case "tEXt":
			parts := bytes.SplitN(chunkData, []byte{0}, 2)
			if len(parts) == 2 {
				metadata[string(parts[0])] = string(parts[1])
			}
		case "zTXt":
			parts := bytes.SplitN(chunkData, []byte{0}, 2)
			if len(parts) == 2 && len(parts[1]) > 1 && parts[1][0] == 0 { // Compression method 0 (zlib)
				reader, err := zlib.NewReader(bytes.NewReader(parts[1][1:]))
				if err == nil {
					var decoded bytes.Buffer
					_, _ = io.Copy(&decoded, reader)
					reader.Close()
					metadata[string(parts[0])] = decoded.String()
				}
			}
		case "iTXt":
			parts := bytes.SplitN(chunkData, []byte{0}, 4)
			if len(parts) >= 4 {
				// keyword := string(parts[0])
				// compressionFlag := parts[1][0]
				// parts[2] is compression method, parts[3] is language tag, translated keyword (followed by null)
				// Actually iTXt structure: Keyword (1-79 bytes) | Null | Compression flag (1 byte) | Compression method (1 byte) | Language tag | Null | Translated keyword | Null | Text
				
				// Re-parsing iTXt more carefully
				nullIdx1 := bytes.IndexByte(chunkData, 0)
				if nullIdx1 != -1 && len(chunkData) > nullIdx1+3 {
					keyword := string(chunkData[:nullIdx1])
					compFlag := chunkData[nullIdx1+1]
					// compMethod := chunkData[nullIdx1+2]
					
					remaining := chunkData[nullIdx1+3:]
					nullIdx2 := bytes.IndexByte(remaining, 0)
					if nullIdx2 != -1 {
						// langTag := string(remaining[:nullIdx2])
						remaining = remaining[nullIdx2+1:]
						nullIdx3 := bytes.IndexByte(remaining, 0)
						if nullIdx3 != -1 {
							// transKeyword := string(remaining[:nullIdx3])
							textData := remaining[nullIdx3+1:]
							
							if compFlag == 1 {
								reader, err := zlib.NewReader(bytes.NewReader(textData))
								if err == nil {
									var decoded bytes.Buffer
									_, _ = io.Copy(&decoded, reader)
									reader.Close()
									metadata[keyword] = decoded.String()
								}
							} else {
								metadata[keyword] = string(textData)
							}
						}
					}
				}
			}
		case "IEND":
			return metadata, config, nil
		}
	}

	return metadata, config, nil
}
