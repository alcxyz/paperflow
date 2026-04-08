package bucket

import "strings"

// GetBucket returns the bucket name for a given file extension.
// Extensions are matched case-insensitively. If no bucket matches,
// "misc" is returned.
func GetBucket(ext string, buckets map[string][]string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	for bucket, exts := range buckets {
		for _, e := range exts {
			if strings.ToLower(e) == ext {
				return bucket
			}
		}
	}
	return "misc"
}

// IsIngestible returns whether a file extension is eligible for ingestion.
// Extensions are matched case-insensitively.
func IsIngestible(ext string, types []string) bool {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, t := range types {
		if strings.ToLower(t) == ext {
			return true
		}
	}
	return false
}
