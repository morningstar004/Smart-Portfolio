package service

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	pdf "github.com/ledongthuc/pdf"
)

// pdfBytesReader wraps a *pdf.Reader that was opened from an in-memory byte
// slice. It exposes the same API surface needed by the ingestion pipeline.
type pdfBytesReader struct {
	Reader *pdf.Reader
}

// NumPage returns the total number of pages in the PDF.
func (r *pdfBytesReader) NumPage() int {
	return r.Reader.NumPage()
}

// Page returns the page at the given 1-based index.
func (r *pdfBytesReader) Page(num int) pdf.Page {
	return r.Reader.Page(num)
}

// pdfNewReader constructs a new pdf.Reader from an io.ReaderAt and total byte
// size.
func pdfNewReader(ra io.ReaderAt, size int64) (*pdf.Reader, error) {
	return pdf.NewReader(ra, size)
}

// ExtractTextFromPDFBytes extracts text from a PDF byte slice.
// It first tries pdftotext (poppler) for best results, then falls
// back to the ledongthuc library if pdftotext is not available.
func ExtractTextFromPDFBytes(data []byte) (string, error) {
	// --- Strategy 1: pdftotext (poppler) ---
	if path, err := exec.LookPath("pdftotext"); err == nil {
		// Write bytes to a temp file
		tmp, err := os.CreateTemp("", "resume-*.pdf")
		if err == nil {
			defer os.Remove(tmp.Name())
			if _, err := tmp.Write(data); err == nil {
				tmp.Close()
				cmd := exec.Command(path, tmp.Name(), "-")
				var out bytes.Buffer
				var stderr bytes.Buffer
				cmd.Stdout = &out
				cmd.Stderr = &stderr
				if err := cmd.Run(); err == nil {
					text := out.String()
					if len(text) > 50 {
						return text, nil
					}
				}
			}
		}
	}

	// --- Strategy 2: ledongthuc/pdf fallback ---
	ra := bytes.NewReader(data)
	r, err := pdf.NewReader(ra, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pdf open failed: %w", err)
	}
	var buf bytes.Buffer
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err == nil {
			buf.WriteString(text)
		}
	}
	result := buf.String()
	if len(result) < 50 {
		return "", fmt.Errorf("no extractable text found in PDF")
	}
	return result, nil
}
