BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE mcp_oauth_links (
 issuer text NOT NULL,
 subject text NOT NULL CHECK (subject <> ''),
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY (issuer, subject),
 UNIQUE (issuer, user_id)
);

CREATE TABLE mcp_oauth_link_flows (
 id uuid PRIMARY KEY,
 issuer text NOT NULL,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 session_hash bytea NOT NULL CHECK (octet_length(session_hash)=32),
 state_hash bytea NOT NULL UNIQUE CHECK (octet_length(state_hash)=32),
 nonce_hash bytea NOT NULL CHECK (octet_length(nonce_hash)=32),
 verifier_ciphertext text,
 phase text NOT NULL CHECK (phase IN ('pending','exchanging','verified')),
 subject text,
 proof_hash bytea CHECK (octet_length(proof_hash)=32),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL,
 UNIQUE (issuer, user_id),
 CHECK (expires_at > created_at AND expires_at <= created_at + INTERVAL '10 minutes'),
 CHECK (
  (phase IN ('pending','exchanging') AND verifier_ciphertext IS NOT NULL AND subject IS NULL AND proof_hash IS NULL)
  OR (phase='verified' AND verifier_ciphertext IS NULL AND subject IS NOT NULL AND subject<>'' AND proof_hash IS NOT NULL)
 )
);
CREATE INDEX mcp_oauth_link_flows_expiry ON mcp_oauth_link_flows(expires_at);
COMMIT;
