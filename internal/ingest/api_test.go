package ingest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckAPI_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Token test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := CheckAPI(server.URL, "test-token"); err != nil {
		t.Errorf("CheckAPI: %v", err)
	}
}

func TestCheckAPI_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	err := CheckAPI(server.URL, "bad-token")
	if err == nil {
		t.Error("expected error for unauthorized response")
	}
}

func TestCheckAPI_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	err := CheckAPI(server.URL, "bad-token")
	if err == nil {
		t.Error("expected error for forbidden response")
	}
}

func TestCheckAPI_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := CheckAPI(server.URL, "test-token")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestCheckAPI_Unreachable(t *testing.T) {
	err := CheckAPI("http://127.0.0.1:1", "test-token")
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestIngestAPI_Success(t *testing.T) {
	var receivedFilename string
	var receivedTitle string
	var receivedContent []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/documents/post_document/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Token test-token" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}

		receivedTitle = r.FormValue("title")

		file, header, err := r.FormFile("document")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		receivedFilename = header.Filename
		receivedContent, _ = io.ReadAll(file)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmp := t.TempDir()
	srcFile := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(srcFile, []byte("pdf data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := IngestAPI(srcFile, server.URL, "test-token"); err != nil {
		t.Fatalf("IngestAPI: %v", err)
	}

	if receivedFilename != "invoice.pdf" {
		t.Errorf("filename = %q, want %q", receivedFilename, "invoice.pdf")
	}
	if receivedTitle != "invoice" {
		t.Errorf("title = %q, want %q", receivedTitle, "invoice")
	}
	if string(receivedContent) != "pdf data" {
		t.Errorf("content = %q, want %q", string(receivedContent), "pdf data")
	}
}

func TestIngestAPI_ServerRejectsUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("No file was submitted"))
	}))
	defer server.Close()

	tmp := t.TempDir()
	srcFile := filepath.Join(tmp, "test.pdf")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	err := IngestAPI(srcFile, server.URL, "test-token")
	if err == nil {
		t.Error("expected error for rejected upload")
	}
}

func TestIngestAPI_FileNotFound(t *testing.T) {
	err := IngestAPI("/nonexistent/file.pdf", "http://localhost", "token")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestIngestAPI_TrailingSlashURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/documents/post_document/" {
			t.Errorf("unexpected path: %s (double slash?)", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmp := t.TempDir()
	srcFile := filepath.Join(tmp, "test.pdf")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// URL with trailing slash should not cause double slash.
	if err := IngestAPI(srcFile, server.URL+"/", "test-token"); err != nil {
		t.Fatalf("IngestAPI: %v", err)
	}
}
