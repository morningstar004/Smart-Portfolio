package service

import (
	"io"

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
// size. This is the main entry point used by openPDFFromBytes in
// ingestion_service.go.
func pdfNewReader(ra io.ReaderAt, size int64) (*pdf.Reader, error) {
	return pdf.NewReader(ra, size)
}
