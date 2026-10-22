package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	loomhttp "github.com/CaliLuke/loom/http"
)

// djangoRequestDecoder preserves Loom's media, size, null and type validation,
// then checks Python's integer conversion limit before invoking the endpoint.
// JSONValue retains integers that the former interface{} decoder rejected as
// float64 overflow. This check prevents the upgrade from accepting new writes.
// It does not change the existing transport error shape or move body parsing
// into the domain layer to reproduce Django's lazy request.data evaluation.
func djangoRequestDecoder(request *http.Request) loomhttp.Decoder {
	media, _, _ := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if request.Header.Get("Content-Type") != "" && media != "application/json" {
		return loomhttp.RequestDecoder(request)
	}
	var body bytes.Buffer
	copy := request.Clone(request.Context())
	copy.Body = struct {
		io.Reader
		io.Closer
	}{io.TeeReader(request.Body, &body), request.Body}
	decoder := loomhttp.RequestDecoder(copy)
	return loomhttp.EncodingFunc(func(value any) error {
		if err := decoder.Decode(value); err != nil {
			return err
		}
		if err := validateJSONIntegerLimit(body.Bytes()); err != nil {
			return err
		}
		return nil
	})
}

func validateJSONIntegerLimit(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		number, ok := token.(json.Number)
		if !ok || strings.ContainsAny(string(number), ".eE") {
			continue
		}
		digits := len(strings.TrimPrefix(string(number), "-"))
		if digits > 4300 {
			return fmt.Errorf("JSON integer exceeds Python's 4300-digit limit (%d digits)", digits)
		}
	}
}
