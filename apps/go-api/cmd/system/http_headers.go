package main

import (
	"encoding/json"
	"math/big"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/requestinput"
)

func nativeRequestHeaders(request *http.Request) string {
	values := make(map[string]string, len(request.Header)+4)
	for name, headerValues := range request.Header {
		if strings.ContainsRune(name, '_') {
			continue
		}
		value := nativeLatin1String([]byte(strings.Join(headerValues, ",")))
		if strings.EqualFold(name, "X-Api-Key") || strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Cookie") {
			value = "[REDACTED]"
		}
		values[http.CanonicalHeaderKey(name)] = value
	}
	if request.Host != "" {
		values["Host"] = nativeLatin1String([]byte(request.Host))
	}
	if request.ContentLength > 0 {
		values["Content-Length"] = strconv.FormatInt(request.ContentLength, 10)
	}
	if len(request.TransferEncoding) > 0 {
		values["Transfer-Encoding"] = strings.Join(request.TransferEncoding, ",")
	}
	if request.Close {
		values["Connection"] = "close"
	}
	encoded, _ := json.Marshal(values)
	return string(encoded)
}

func nativeRequestAcceptHeader(request *http.Request) (string, bool) {
	values, present := request.Header[http.CanonicalHeaderKey("Accept")]
	return strings.Join(values, ","), present
}

func projectMembershipReadsJSONNegotiation(accept string, present bool) (bool, int) {
	if !present {
		accept = "*/*"
	}
	bestPrecedence := -1
	bestIndent := 0
	for _, value := range strings.Split(accept, ",") {
		if strings.TrimSpace(value) == "" {
			continue
		}
		mediaType, parameters, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err != nil {
			return false, 0
		}
		precedence := -1
		switch mediaType {
		case "*/*":
			precedence = 0
		case "application/*":
			precedence = 1
		case "application/json":
			precedence = 2
			for name := range parameters {
				if name != "q" {
					precedence = 3
					break
				}
			}
		}
		if precedence <= bestPrecedence {
			continue
		}
		bestPrecedence = precedence
		bestIndent = nativeJSONIndent(parameters["indent"])
	}
	return bestPrecedence >= 0, bestIndent
}

func nativeJSONIndent(value string) int {
	integer, err := requestinput.IntegerField(value)
	if err != nil || integer.Sign() <= 0 {
		return 0
	}
	maximum := big.NewInt(8)
	if integer.Cmp(maximum) >= 0 {
		return 8
	}
	return int(integer.Int64())
}

func nativeJoinedHeader(request *http.Request, name string) string {
	return nativeLatin1String([]byte(strings.Join(request.Header.Values(name), ",")))
}

func mergeVary(headers http.Header, required ...string) {
	seen := make(map[string]bool)
	values := make([]string, 0, len(required)+len(headers.Values("Vary")))
	for _, line := range append(headers.Values("Vary"), strings.Join(required, ", ")) {
		for _, raw := range strings.Split(line, ",") {
			value := strings.TrimSpace(raw)
			key := strings.ToLower(value)
			if value == "" || seen[key] {
				continue
			}
			seen[key] = true
			values = append(values, value)
		}
	}
	headers.Set("Vary", strings.Join(values, ", "))
}

func removeVary(headers http.Header, removed ...string) {
	blocked := make(map[string]bool, len(removed))
	for _, value := range removed {
		blocked[strings.ToLower(value)] = true
	}
	values := make([]string, 0, len(headers.Values("Vary")))
	seen := make(map[string]bool)
	for _, line := range headers.Values("Vary") {
		for _, raw := range strings.Split(line, ",") {
			value := strings.TrimSpace(raw)
			key := strings.ToLower(value)
			if value == "" || blocked[key] || seen[key] {
				continue
			}
			seen[key] = true
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		headers.Del("Vary")
		return
	}
	headers.Set("Vary", strings.Join(values, ", "))
}
