package bucket

import "testing"

func defaultBuckets() map[string][]string {
	return map[string][]string{
		"pdf":    {"pdf"},
		"images": {"jpg", "jpeg", "png", "gif", "webp", "tiff", "tif"},
		"docx":   {"docx", "doc", "odt", "rtf"},
		"xlsx":   {"xlsx", "xls", "ods"},
	}
}

func TestGetBucket(t *testing.T) {
	buckets := defaultBuckets()

	tests := []struct {
		ext  string
		want string
	}{
		{"pdf", "pdf"},
		{"PDF", "pdf"},
		{".pdf", "pdf"},
		{".PDF", "pdf"},
		{"jpg", "images"},
		{"jpeg", "images"},
		{"png", "images"},
		{"gif", "images"},
		{"webp", "images"},
		{"tiff", "images"},
		{"tif", "images"},
		{"TIFF", "images"},
		{"docx", "docx"},
		{"doc", "docx"},
		{"odt", "docx"},
		{"rtf", "docx"},
		{"xlsx", "xlsx"},
		{"xls", "xlsx"},
		{"ods", "xlsx"},
		{"txt", "misc"},
		{"zip", "misc"},
		{"exe", "misc"},
		{"", "misc"},
		{".unknown", "misc"},
	}

	for _, tt := range tests {
		got := GetBucket(tt.ext, buckets)
		if got != tt.want {
			t.Errorf("GetBucket(%q) = %q, want %q", tt.ext, got, tt.want)
		}
	}
}

func TestIsIngestible(t *testing.T) {
	types := []string{"pdf", "jpg", "jpeg", "png", "gif", "webp", "tiff", "tif", "docx", "odt", "xlsx"}

	tests := []struct {
		ext  string
		want bool
	}{
		{"pdf", true},
		{"PDF", true},
		{".pdf", true},
		{"jpg", true},
		{"jpeg", true},
		{"png", true},
		{"docx", true},
		{"xlsx", true},
		{"odt", true},
		{"txt", false},
		{"zip", false},
		{"rtf", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsIngestible(tt.ext, types)
		if got != tt.want {
			t.Errorf("IsIngestible(%q) = %v, want %v", tt.ext, got, tt.want)
		}
	}
}
