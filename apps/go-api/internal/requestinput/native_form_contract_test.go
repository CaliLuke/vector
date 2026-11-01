package requestinput

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/CaliLuke/go-plane/internal/apicontract"
)

func TestNativeFormsRejectMalformedEscapesAndUTF8(t *testing.T) {
	for _, body := range []string{"name=%ZZ", "name=%ff", "%ff=name"} {
		t.Run(body, func(t *testing.T) {
			_, err := Parse(strings.NewReader(body), "application/x-www-form-urlencoded")
			var invalid *apicontract.HTTPError
			if !errors.As(err, &invalid) || invalid.Status != 400 {
				t.Fatalf("malformed form accepted: %v", err)
			}
		})
	}
	body := []byte("--form\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\n")
	body = append(body, 0xff)
	body = append(body, []byte("\r\n--form--\r\n")...)
	_, err := Parse(bytes.NewReader(body), "multipart/form-data; boundary=form")
	var invalid *apicontract.HTTPError
	if !errors.As(err, &invalid) || invalid.Status != 400 {
		t.Fatalf("malformed multipart text accepted: %v", err)
	}
}

func TestNativeRequestPropagatesBodyReadFailure(t *testing.T) {
	expected := errors.New("body read failed")
	input, err := Parse(iotest.ErrReader(expected), "application/json")
	if !errors.Is(err, expected) || len(input.Fields) != 0 {
		t.Fatalf("read failure=%v fields=%#v", err, input.Fields)
	}
}

func TestNativeFormsKeepUnicodeAndBinaryUploadMetadata(t *testing.T) {
	input, err := Parse(strings.NewReader("name=Zo%C3%AB&name=last&note=a%3Bb+%26+c"), "application/x-www-form-urlencoded")
	if err != nil || input.Fields["name"] != "last" || input.FormValues["name"][0] != "Zoë" || input.Fields["note"] != "a;b & c" {
		t.Fatalf("form=%#v error=%v", input, err)
	}
	body := []byte("--form\r\nContent-Disposition: form-data; name=\"document\"; filename=\"data.bin\"\r\n\r\n")
	body = append(body, 0xff, 0x00)
	body = append(body, []byte("\r\n--form--\r\n")...)
	input, err = Parse(bytes.NewReader(body), "multipart/form-data; boundary=form")
	if err != nil || input.Fields["document"] != (UploadedFile{Name: "data.bin"}) {
		t.Fatalf("binary upload metadata=%#v error=%v", input.Fields, err)
	}
}
