package assets

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"testing"
)

func TestParseLegacyAssetFilePreservesBinaryAndLastRepeatedFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, file := range []struct {
		name, contents string
	}{
		{"first.png", string([]byte{0, 0xff, 0x80, 1})},
		{"last.png", string([]byte{0x89, 'P', 'N', 'G', 0, 0xff})},
	} {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="asset"; filename="`+file.name+`"`)
		header.Set("Content-Type", "image/png")
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(file.contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	file, err := parseLegacyAssetFile(&body, writer.FormDataContentType())
	if err != nil {
		t.Fatalf("parse multipart file: %v", err)
	}
	if file.Name != "last.png" || file.ContentType != "image/png" || !bytes.Equal(file.Contents, []byte{0x89, 'P', 'N', 'G', 0, 0xff}) {
		t.Fatalf("parsed file = %#v", file)
	}
}
