// Package content validates and sanitizes collaborative document content.
//
// The HTML allowlist preserves editor markup and rejects executable content.
// The bounded internal/htmlcompat parser permits documents up to 4096 nested
// elements. Binary documents use standard padded base64 with size and content
// validation. Sticky description_binary remains a read-only API field.
package content
