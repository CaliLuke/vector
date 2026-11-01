// Package browsersession implements Vector's server-side browser sessions.
//
// Opaque cookie keys select expiring database rows. Versioned JSON payloads use
// purpose-separated signatures and password-version digests. Validation checks
// the current user's active state and password hash. Fallback signing keys can
// be rotated without accepting another framework's session format.
package browsersession
