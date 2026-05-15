package main

const (
	// PutAction - Put artifacts
	PutAction = "put"

	// DeleteAction - Delete artifacts
	DeleteAction = "delete"

	// GetAction - Get artifacts
	GetAction = "get"

	// ErrCodeNotFound - s3 Not found error code
	ErrCodeNotFound = "NotFound"

	// Compression algorithm options
	CompressZip     = "zip"
	CompressTarGzip = "tar+gzip"
	CompressTarZstd = "tar+zstd"
)

type (
	// Action - Input params
	Action struct {
		Action      string
		Bucket      string
		S3Class     string
		Key         string
		Artifacts   []string
		Compression string
	}
)
