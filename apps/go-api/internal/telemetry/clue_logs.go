package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/logredact"
	cluelog "github.com/CaliLuke/loom/clue/log"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

// newClueLogFormatter adapts clue's public structured FormatFunc to OpenTelemetry.
// The paired output writer is io.Discard because the formatter emits the Entry
// directly instead of serializing it through stdout or a second JSON shape.
func newClueLogFormatter(logger otellog.Logger) cluelog.FormatFunc {
	return func(entry *cluelog.Entry) []byte {
		record := otellog.Record{}
		record.SetTimestamp(entry.Time)
		record.SetObservedTimestamp(time.Now())
		record.SetSeverity(clueSeverity(entry.Severity))
		record.SetSeverityText(entry.Severity.String())

		var message string
		var traceID, spanID string
		attributes := make([]attribute.KeyValue, 0, len(entry.KeyVals))
		for _, field := range entry.KeyVals {
			key := strings.TrimSpace(field.K)
			if key == "" || key == cluelog.TimestampKey || key == cluelog.SeverityKey {
				continue
			}
			if key == cluelog.MessageKey {
				message = safeLogText(field.V)
				continue
			}
			if key == cluelog.TraceIDKey {
				traceID = fmt.Sprint(field.V)
			}
			if key == cluelog.SpanIDKey {
				spanID = fmt.Sprint(field.V)
			}
			if secretLogKey(key) {
				attributes = append(attributes, attribute.String(key, "[redacted]"))
				continue
			}
			attributes = append(attributes, safeLogAttribute(key, field.V))
		}
		if message == "" {
			message = "application log"
		}
		record.SetBody(attribute.StringValue(message))
		record.AddAttributes(attributes...)
		logContext := context.Background()
		parsedTraceID, traceErr := trace.TraceIDFromHex(traceID)
		parsedSpanID, spanErr := trace.SpanIDFromHex(spanID)
		if traceErr == nil && spanErr == nil {
			spanContext := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: parsedTraceID,
				SpanID:  parsedSpanID,
			})
			logContext = trace.ContextWithSpanContext(logContext, spanContext)
		}
		logger.Emit(logContext, record)
		return nil
	}
}

// openTelemetryOutput builds an output that emits clue entries as structured OTLP logs.
// Clue invokes Format before writing, so the adapter emits from the structured Entry
// and its writer intentionally discards the empty formatted payload.
func openTelemetryOutput(logger otellog.Logger) cluelog.Output {
	return cluelog.Output{Writer: io.Discard, Format: newClueLogFormatter(logger)}
}

func clueSeverity(severity cluelog.Severity) otellog.Severity {
	switch severity {
	case cluelog.SeverityDebug:
		return otellog.SeverityDebug
	case cluelog.SeverityWarn:
		return otellog.SeverityWarn
	case cluelog.SeverityError:
		return otellog.SeverityError
	default:
		return otellog.SeverityInfo
	}
}

func safeLogAttribute(key string, value any) attribute.KeyValue {
	switch value := value.(type) {
	case string:
		return attribute.String(key, safeLogText(value))
	case bool:
		return attribute.Bool(key, value)
	case int:
		return attribute.Int(key, value)
	case int32:
		return attribute.Int64(key, int64(value))
	case int64:
		return attribute.Int64(key, value)
	case uint:
		return attribute.Int64(key, int64(value))
	case uint32:
		return attribute.Int64(key, int64(value))
	case uint64:
		return attribute.Int64(key, int64(value))
	case float32:
		return attribute.Float64(key, float64(value))
	case float64:
		return attribute.Float64(key, value)
	case nil:
		return attribute.String(key, "null")
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return attribute.String(key, safeLogText(fmt.Sprint(value)))
		}
		return attribute.String(key, safeLogText(string(encoded)))
	}
}

func safeLogText(value any) string {
	return logredact.RedactSensitiveText(fmt.Sprint(value))
}

func secretLogKey(key string) bool {
	key = strings.ToLower(key)
	for _, secret := range []string{"password", "secret", "authorization", "cookie", "token"} {
		if strings.Contains(key, secret) {
			return true
		}
	}
	return false
}
