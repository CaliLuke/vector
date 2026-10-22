package assets

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
)

type legacyUploadedFile struct {
	Name, ContentType string
	Contents          []byte
	Fields            map[string]string
}

func parseLegacyAssetFile(body io.Reader, contentType string) (legacyUploadedFile, error) {
	mediaType, parameters, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" || parameters["boundary"] == "" {
		return legacyUploadedFile{}, fmt.Errorf("invalid multipart content type")
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return legacyUploadedFile{}, err
	}
	reader := multipart.NewReader(bytes.NewReader(data), parameters["boundary"])
	result := legacyUploadedFile{Fields: map[string]string{}}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return legacyUploadedFile{}, fmt.Errorf("parse multipart asset: %w", err)
		}
		contents, readErr := io.ReadAll(part)
		name, filename := part.FormName(), part.FileName()
		partType := part.Header.Get("Content-Type")
		_ = part.Close()
		if readErr != nil {
			return legacyUploadedFile{}, readErr
		}
		if name == "asset" && filename != "" {
			result.Name, result.ContentType, result.Contents = filename, partType, contents
		} else if name != "" {
			// QueryDict.get() returns the last multipart value for repeated keys.
			result.Fields[name] = string(contents)
		}
	}
	return result, nil
}
