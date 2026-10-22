// Package djangosession is the read-only M1 compatibility port for Plane's
// database-backed Django sessions.
//
// The port follows these pinned Django 5.2.15 sources:
//
//   - django/core/signing.py: Signer and TimestampSigner
//   - django/contrib/sessions/backends/base.py: SessionBase.encode and decode
//   - django/contrib/sessions/backends/db.py: expiry-filtered SessionStore.load
//   - django/contrib/auth/base_user.py: AbstractBaseUser session auth hashes
//   - django/contrib/auth/__init__.py: get_user fallback-key validation
//
// Plane adds the denormalized user_id in plane/db/models/session.py. Its
// settings select session-id and django.contrib.auth.backends.ModelBackend.
// This package never writes, cycles, flushes, or otherwise rotates a session.
package djangosession
