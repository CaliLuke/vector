package assets

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttachmentMIMEInferenceUsesDocumentedFormats(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"note.txt", "text/plain"}, {"README.MD", "text/markdown"}, {"note.markdown", "text/markdown"},
		{"table.csv", "text/csv"}, {"page.htm", "text/html"}, {"page.html", "text/html"},
		{"data.xml", "application/xml"}, {"data.yaml", "application/yaml"}, {"data.yml", "application/yaml"}, {"data.json", "application/json"},
		{"image.png", "image/png"}, {"image.jpg", "image/jpeg"}, {"image.jpeg", "image/jpeg"}, {"image.gif", "image/gif"}, {"image.webp", "image/webp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, fallback := range []string{"", "application/octet-stream"} {
				require.Equal(t, tc.want, attachmentMIME(fallback, tc.name))
			}
		})
	}
}

func TestAttachmentMIMEPreservesDeclaredTypeAndUnknownFallback(t *testing.T) {
	require.Equal(t, "application/pdf", attachmentMIME("application/pdf; version=1.7", "report.txt"))
	require.Equal(t, "application/pdf", attachmentMIME("", "report.pdf"))
	require.Equal(t, "text/plain", attachmentMIME("text/plain; charset=utf-8", "report.bin"))
	require.Equal(t, "text/plain", attachmentMIME("not a MIME type", "report.txt"))
	require.Equal(t, "application/octet-stream", attachmentMIME("", "file.vector-unknown-extension"))
}
