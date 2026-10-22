# Auth email-check fixture corrections

The first 59-case Go diagnostic exposed two recorder issues and an effective
test-setting mismatch. The seven session-CSRF fixtures had been recorded in a
single test while retaining each preceding `force_login` session. The recorder
now removes the session created for each case after its request snapshot, so
each independent fixture has one session before and after the request. The
pristine seven fixtures are retained under
`corrected-fixtures/pristine-email-session-csrf/`; isolated source capture
passed 7/7 at `source-20261022T131449Z.log`.

The source method-path recorder confirms `settings.DEBUG` is false under
pytest-django, even though `plane.settings.test` declares it true. The Go replay
now passes `--debug=false` to match that effective source runtime. Debug-mode
slashless unsafe requests remain covered by a system regression and retain the
existing debug failure behavior.

For raw-body fixtures, Django exposes the request media type as `CONTENT_TYPE`.
The recorder previously added a second `Content-Type` field. It now adds the
transport media type only when the caller has not already recorded that header,
and fails if the supplied values disagree. Four raw-codec cases were recaptured
4/4 at `source-20261022T132006Z.log`; their pristine copies are under
`corrected-fixtures/pristine-email-codecs/`.

The four app preflight fixtures use a dedicated replay profile that adds
`https://app.example.test` to CORS while keeping the other email-check profiles
at `http://localhost:3000`. CSRF trusted origins are configured independently
for every profile, matching Django's test override without granting CORS to an
origin that is trusted only for CSRF. The 59 fixtures now resolve into ten
target configuration profiles, including the dedicated CORS preflight profile.
