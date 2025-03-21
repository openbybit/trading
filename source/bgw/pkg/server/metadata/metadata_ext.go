package metadata

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

var (
	metadataExts = map[string]MetadataExt{}
)

// MetadataExt is a metadata extension interface.
type MetadataExt interface {
	Extract(ctx context.Context) metadata.MD
}

// Register registers a metadata extension.
func Register(name string, ext MetadataExt) {
	metadataExts[strings.ToLower(name)] = ext
}

// GetMetadataExts returns all registered metadata extensions.
func GetMetadataExts() map[string]MetadataExt {
	return metadataExts
}
