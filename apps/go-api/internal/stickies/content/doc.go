// Package content reproduces the Sticky content validation performed by Plane's
// Django application.
//
// The HTML sanitizer is compatible with plane.utils.content_validator from the
// repository's pinned nh3 0.2.18 dependency. StripTags is compatible with
// plane.utils.html_processor, which uses Python's html.parser.HTMLParser.
// The implementation uses the audited internal/htmlcompat parser. That package
// is based on x/net/html v0.57.0. It replaces the upstream 512-element guard
// with a configurable limit. Sticky validation sets that limit to 4096. A broad
// sanitizer was rejected because it changes the allowlists and output.
//
// Sticky's Django serializer exposes description_binary as read-only. A future
// Sticky handler must ignore that request field instead of calling the binary
// validator. The binary functions preserve the source utility for other uses.
package content
