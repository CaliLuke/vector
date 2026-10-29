package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
	"unicode/utf8"

	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
)

func writeAPIActivityLog(
	ctx context.Context,
	logger publicprojectmetadatareadsstore.ActivityLogger,
	secret string,
	request *http.Request,
	requestBody []byte,
	responseCode int,
	responseBody []byte,
) {
	apiKey := djangoJoinedHeader(request, "X-Api-Key")
	if logger == nil || apiKey == "" {
		return
	}
	record := publicprojectmetadatareadsstore.APIActivityLog{
		TokenIdentifier: apiActivityTokenIdentifier(secret, apiKey),
		Path:            request.URL.Path,
		Method:          request.Method,
		QueryParams:     readServicesOriginalRawQuery(request),
		Headers:         djangoRequestHeaders(request),
		Body:            safeDecodeAPIActivityBody(requestBody),
		ResponseCode:    responseCode,
		ResponseBody:    safeDecodeAPIActivityBody(responseBody),
		IPAddress:       nullableString(apiActivityIPAddress(request)),
		UserAgent:       nullableString(djangoJoinedHeader(request, "User-Agent")),
	}
	// Wait for the durable publisher confirmation. Ignore client cancellation as
	// Django does while submitting the task, but retain the request log values.
	logCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := logger.Log(logCtx, record); err != nil {
		log.Error(ctx, err, log.KV{K: "msg", V: "publish API activity log"})
	}
}

func apiActivityTokenIdentifier(secret string, apiKey string) string {
	digest := hmac.New(sha256.New, []byte(secret))
	_, _ = digest.Write([]byte(apiKey))
	return hex.EncodeToString(digest.Sum(nil))
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func safeDecodeAPIActivityBody(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	if bytes.HasPrefix(value, []byte("\x89PNG")) || bytes.HasPrefix(value, []byte("\xff\xd8\xff")) || bytes.HasPrefix(value, []byte("%PDF")) {
		return "[Binary Content]"
	}
	if !utf8.Valid(value) {
		return "[Could not decode content]"
	}
	return string(value)
}

func apiActivityIPAddress(request *http.Request) string {
	return loomhttp.EffectiveClientAddress(request)
}
