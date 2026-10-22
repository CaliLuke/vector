# AUTH-002: Correct the space OAuth callback shadowing crash

The pristine pinned Django source at `a08553ccd1961c46c7ea25cc3bb7fc1fdd71a56a` raises `TypeError: 'NoneType' object is not callable` for invalid-state Google, GitHub, and GitLab callbacks under `/auth/spaces/`. Each callback assigns `base_host = request.session.get("host")`, then calls the imported `base_host(...)` helper while constructing its error redirect. Gitea does not shadow the helper and returns its normal provider error redirect.

The source recorder preserves the three HTTP 500 responses in `fixtures/authentication_redirect_routes/`. The corrected oracle under `fixtures/authentication_redirect_routes_corrected/` uses the corresponding app callback's intended provider-error redirect with the configured space base URL and `/spaces/` base path. The expected redirects are `https://space.example.test/spaces/?error_code=5115&error_message=GOOGLE_OAUTH_PROVIDER_ERROR`, `...?error_code=5120&error_message=GITHUB_OAUTH_PROVIDER_ERROR`, and `...?error_code=5121&error_message=GITLAB_OAUTH_PROVIDER_ERROR`.

The Go implementation must return those corrected redirects, retain the pristine source fixtures and capture log, and create no application database writes for invalid OAuth state.
