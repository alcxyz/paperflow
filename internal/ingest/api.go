package ingest

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// IngestAPI uploads a file to Paperless-ngx via the REST API.
func IngestAPI(filePath string, paperlessURL string, token string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	title := strings.TrimSuffix(filename, ext)

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Write multipart form in a goroutine to stream it.
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()

		part, err := writer.CreateFormFile("document", filename)
		if err != nil {
			errCh <- err
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			errCh <- err
			return
		}

		if err := writer.WriteField("title", title); err != nil {
			errCh <- err
			return
		}

		errCh <- writer.Close()
	}()

	url := strings.TrimRight(paperlessURL, "/") + "/api/documents/post_document/"
	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading to paperless: %w", err)
	}
	defer resp.Body.Close()

	// Wait for the multipart writer goroutine to finish.
	if err := <-errCh; err != nil {
		return fmt.Errorf("writing multipart form: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("paperless API returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
