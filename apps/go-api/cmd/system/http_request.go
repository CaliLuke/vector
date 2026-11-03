package main

import (
	"fmt"
	"html"
	"math"
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	loomhttp "github.com/CaliLuke/loom/http"
	encodingunicode "golang.org/x/text/encoding/unicode"
)

var requestMetadataPolicy = func() loomhttp.RequestMetadataPolicy {
	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie", "Content-Type"}, []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fc00::/7"),
	})
	if err != nil {
		panic(err)
	}
	return policy
}()

func nativeQueryValues(rawQuery string, name string) ([]string, error) {
	if strings.Count(rawQuery, "&") >= 1000 {
		return nil, fmt.Errorf("query exceeds the 1000-field limit")
	}
	values := make([]string, 0, 1)
	for _, field := range strings.Split(rawQuery, "&") {
		key, value, found := strings.Cut(field, "=")
		if !found {
			value = ""
		}
		if nativeQueryUnescape(key) == name {
			values = append(values, nativeQueryUnescape(value))
		}
	}
	return values, nil
}

func normalizeNativeTolerantQuery(request *http.Request) *http.Request {
	if request.URL.RawQuery == "" {
		return request
	}
	values := make(url.Values)
	for _, field := range strings.Split(request.URL.RawQuery, "&") {
		if field == "" {
			continue
		}
		key, value, found := strings.Cut(field, "=")
		if !found {
			value = ""
		}
		key = nativeQueryUnescape(key)
		values[key] = append(values[key], nativeQueryUnescape(value))
	}
	clone := request.Clone(request.Context())
	clonedURL := *request.URL
	clonedURL.RawQuery = values.Encode()
	clone.URL = &clonedURL
	return clone
}

func nativeQueryUnescape(value string) string {
	decoded := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		switch {
		case value[index] == '+':
			decoded = append(decoded, ' ')
		case value[index] == '%' && index+2 < len(value):
			high, highOK := queryHexDigit(value[index+1])
			low, lowOK := queryHexDigit(value[index+2])
			if highOK && lowOK {
				decoded = append(decoded, high<<4|low)
				index += 2
				continue
			}
			decoded = append(decoded, value[index])
		default:
			decoded = append(decoded, value[index])
		}
	}
	// Decode invalid UTF-8 with replacement characters for display-safe queries.
	text, _ := encodingunicode.UTF8.NewDecoder().String(string(decoded))
	return text
}

func queryHexDigit(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

var nativeBadRequestBody = []byte("\n<!doctype html>\n<html lang=\"en\">\n<head>\n  <title>Bad Request (400)</title>\n</head>\n<body>\n  <h1>Bad Request (400)</h1><p></p>\n</body>\n</html>\n")

var nativeNotFoundBody = []byte("\n<!doctype html>\n<html lang=\"en\">\n<head>\n  <title>Not Found</title>\n</head>\n<body>\n  <h1>Not Found</h1><p>The requested resource was not found on this server.</p>\n</body>\n</html>\n")

var nativeServerErrorBody = []byte("\n<!doctype html>\n<html lang=\"en\">\n<head>\n  <title>Server Error (500)</title>\n</head>\n<body>\n  <h1>Server Error (500)</h1><p></p>\n</body>\n</html>\n")

var nativeRequestTooLargeBody = []byte(`{"error": "REQUEST_BODY_TOO_LARGE", "detail": "The size of the request body exceeds the maximum allowed size."}`)

func nativeDebugAppendSlashResponse(request *http.Request) ([]byte, string) {
	pathWithSlash := nativeFullPath(request, true)
	message := fmt.Sprintf(
		"The %s request requires a trailing slash. Submit the request to %s%s.",
		request.Method, request.Host, pathWithSlash,
	)
	if !nativeRequestAcceptsHTML(request) {
		return []byte(fmt.Sprintf("RuntimeError at %s\n\n%s\n", request.URL.Path, message)), "text/plain; charset=utf-8"
	}
	return []byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>RuntimeError at %s</title></head>
<body><h1>RuntimeError at %s</h1><pre class="exception_value">%s</pre></body>
</html>`, html.EscapeString(request.URL.Path), html.EscapeString(request.URL.Path), html.EscapeString(message))), "text/html; charset=utf-8"
}

func nativeDebugAppendSlashBody(request *http.Request) []byte {
	body, _ := nativeDebugAppendSlashResponse(request)
	return body
}

func nativeRequestAcceptsHTML(request *http.Request) bool {
	accept, present := nativeRequestAcceptHeader(request)
	if !present {
		return true
	}
	for _, value := range strings.Split(accept, ",") {
		mediaType, parameters, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		quality := 1.0
		if rawQuality, exists := parameters["q"]; exists {
			if parsed, parseErr := strconv.ParseFloat(rawQuality, 64); parseErr == nil && parsed >= 0 && parsed <= 1 {
				quality = math.Round(parsed*1000) / 1000
			}
		}
		if quality == 0 {
			continue
		}
		parameterMismatch := false
		for name := range parameters {
			if name != "q" {
				parameterMismatch = true
				break
			}
		}
		if !parameterMismatch && (mediaType == "*/*" || mediaType == "text/*" || mediaType == "text/html") {
			return true
		}
	}
	return false
}

func nativeEscapeURIPath(path string) string {
	return nativePercentEncode(path, "/:@&+$,-.!~*'()")
}

func nativeFullPath(request *http.Request, appendSlash bool) string {
	path := request.URL.Path
	if appendSlash {
		path += "/"
	}
	fullPath := nativeEscapeURIPath(path)
	if request.URL.RawQuery != "" {
		fullPath += "?" + nativePercentEncode(request.URL.RawQuery, "/#%[]=:;$&()+,!?*@'~")
	}
	return fullPath
}

func nativePercentEncode(value string, safe string) string {
	const hexadecimal = "0123456789ABCDEF"
	var escaped strings.Builder
	escaped.Grow(len(value))
	for index := 0; index < len(value); index++ {
		octet := value[index]
		if (octet >= 'a' && octet <= 'z') || (octet >= 'A' && octet <= 'Z') ||
			(octet >= '0' && octet <= '9') || strings.ContainsRune("_-."+safe, rune(octet)) {
			escaped.WriteByte(octet)
			continue
		}
		escaped.WriteByte('%')
		escaped.WriteByte(hexadecimal[octet>>4])
		escaped.WriteByte(hexadecimal[octet&0x0f])
	}
	return escaped.String()
}
