package apiclient

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SubmitZipJob uploads a ZIP archive of a static site to POST /api/v1/jobs/zip
// and returns the created job. The archive is streamed, not buffered.
func (c *Client) SubmitZipJob(
	ctx context.Context,
	zipPath string,
	modules []string,
	screenshot bool,
) (SubmitJobResponse, error) {
	reqURL, err := c.BuildURL("/api/v1/jobs/zip")
	if err != nil {
		return SubmitJobResponse{}, err
	}

	file, err := os.Open(zipPath) // #nosec G304 -- uploading a user-supplied archive is the point
	if err != nil {
		return SubmitJobResponse{}, fmt.Errorf("open archive: %w", err)
	}

	bodyReader, contentType := zipUploadBody(file, filepath.Base(zipPath), modules, screenshot)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bodyReader)
	if err != nil {
		_ = file.Close()

		return SubmitJobResponse{}, err
	}

	req.Header.Set("Content-Type", contentType)

	resp, err := c.Do(req)
	if err != nil {
		return SubmitJobResponse{}, fmt.Errorf("failed to execute request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var out SubmitJobResponse
	if err = decodeJSONResponse(resp, &out); err != nil {
		return SubmitJobResponse{}, err
	}

	return out, nil
}

// zipUploadBody streams a multipart body: the archive under "file", plus the
// modules CSV and screenshot toggle fields. It closes archive when done.
func zipUploadBody(
	archive io.ReadCloser,
	filename string,
	modules []string,
	screenshot bool,
) (io.Reader, string) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		err := writeZipUploadParts(writer, archive, filename, modules, screenshot)

		_ = archive.Close()

		pw.CloseWithError(err)
	}()

	return pr, writer.FormDataContentType()
}

func writeZipUploadParts(
	writer *multipart.Writer,
	archive io.Reader,
	filename string,
	modules []string,
	screenshot bool,
) error {
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return err
	}

	if _, err = io.Copy(part, archive); err != nil {
		return err
	}

	if len(modules) > 0 {
		if err = writer.WriteField("modules", strings.Join(modules, ",")); err != nil {
			return err
		}
	}

	if err = writer.WriteField("screenshot", strconv.FormatBool(screenshot)); err != nil {
		return err
	}

	return writer.Close()
}
