package devtools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const maxLogMessageBytes = 8000
const maxExportBytes = 256 << 10
const maxExportRecords = 100

var (
	ansiPattern   = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	secretPattern = regexp.MustCompile(`(?i)(password|secret|authorization|cookie|token)([\w-]*["']?\s*[:=]\s*)("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\r\n]*)`)
	headerPattern = regexp.MustCompile(`(?i)((?:proxy-)?authorization|(?:set-)?cookie)(["']?\s*[:=]\s*["']?)[^\r\n,"']+`)
	errorPattern  = regexp.MustCompile(`(?i)\b(error|exception|fatal|panic)\b`)
	warnPattern   = regexp.MustCompile(`(?i)\bwarn(?:ing)?\b`)
)

// RecordLog turns one local process line into the OTLP/HTTP JSON shape accepted by Logal.
func RecordLog(line, service string, eventTime time.Time) map[string]any {
	message := headerPattern.ReplaceAllString(ansiPattern.ReplaceAllString(line, ""), `${1}${2}[redacted]`)
	message = secretPattern.ReplaceAllStringFunc(message, redactSecret)
	message = strings.TrimSpace(message)
	if len(message) > maxLogMessageBytes {
		message = message[:maxLogMessageBytes]
		for !utf8.ValidString(message) {
			message = message[:len(message)-1]
		}
	}
	if message == "" {
		return nil
	}
	level, number := "INFO", 9
	if errorPattern.MatchString(message) {
		level, number = "ERROR", 17
	} else if warnPattern.MatchString(message) {
		level, number = "WARN", 13
	}
	if eventTime.IsZero() {
		eventTime = time.Now()
	}
	return map[string]any{
		"resource": map[string]any{"attributes": []any{map[string]any{
			"key": "service.name", "value": map[string]any{"stringValue": service},
		}}},
		"scopeLogs": []any{map[string]any{
			"scope": map[string]any{"name": "plane.local.stdout"},
			"logRecords": []any{map[string]any{
				"timeUnixNano": fmt.Sprint(eventTime.UnixNano()), "severityText": level,
				"severityNumber": number, "body": map[string]any{"stringValue": message},
			}},
		}},
	}
}

func redactSecret(value string) string {
	match := secretPattern.FindStringSubmatch(value)
	if len(match) != 4 {
		return value
	}
	secret := match[3]
	quote := ""
	if len(secret) >= 2 && (secret[0] == '\'' || secret[0] == '"') && secret[len(secret)-1] == secret[0] {
		quote = secret[:1]
	}
	return match[1] + match[2] + quote + "[redacted]" + quote
}

// ExportLogs posts a bounded OTLP batch. The response body is deliberately capped.
func ExportLogs(ctx context.Context, endpoint string, records []map[string]any) error {
	for offset := 0; offset < len(records); {
		chunk := make([]map[string]any, 0, maxExportRecords)
		var data []byte
		for offset < len(records) && len(chunk) < maxExportRecords {
			candidate := append(chunk, records[offset])
			encoded, err := marshalLogBatch(candidate)
			if err != nil {
				return err
			}
			if len(encoded) > maxExportBytes {
				if len(chunk) == 0 {
					// Skip an individually oversized record so it cannot block the queue forever.
					offset++
					continue
				}
				break
			}
			chunk = candidate
			data = encoded
			offset++
		}
		if len(chunk) == 0 {
			continue
		}
		if err := postLogBatch(ctx, endpoint, data); err != nil {
			return err
		}
	}
	return nil
}

func marshalLogBatch(records []map[string]any) ([]byte, error) {
	return json.Marshal(map[string]any{"resourceLogs": records})
}

func postLogBatch(ctx context.Context, endpoint string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Logal returned HTTP %d", resp.StatusCode)
	}
	return nil
}
