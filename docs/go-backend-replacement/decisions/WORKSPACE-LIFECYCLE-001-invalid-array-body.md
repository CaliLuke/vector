# Workspace lifecycle invalid array body correction

`POST /api/workspaces/` with a JSON array body (`[]`) reaches a Django handler that assumes an object and returns HTTP 500. The request produces no database writes or task publications. Go validates the body shape and returns HTTP 400 before persistence or effects.

The live transport comparison records the pristine source response as status 500 with body SHA-256 `d06853a8ad551a5a3795f75fc78bcac6c8606bb49f6fe78a4e39573259c241e2` and the corrected response as status 400 with body SHA-256 `1519e12b65c4ff03868472912866d2c4d6c790517633174b346432f3a5cbae86`. The contract projects only those exact status and body fields for `post_api_workspaces:canonical:POST`; all headers and effects remain compared normally. Any other source response continues to fail the differential.
