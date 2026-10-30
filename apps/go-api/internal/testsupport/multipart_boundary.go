package testsupport

// MultipartBoundaryCase records the native parser boundary and error behavior.
type MultipartBoundaryCase struct {
	Name      string
	Body      string
	WantError bool
	WantTitle string
}

// MultipartBoundaryCases supplies the shared native form-boundary contract.
func MultipartBoundaryCases() []MultipartBoundaryCase {
	const field = "--sample\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nhello"
	return []MultipartBoundaryCase{
		{Name: "empty complete form", Body: "--sample--\r\n"},
		{Name: "field with final newline", Body: field + "\r\n--sample--\r\n", WantTitle: "hello"},
		{Name: "field without final newline", Body: field + "\r\n--sample--", WantTitle: "hello"},
		{Name: "preamble without any boundary", Body: "ignored preamble\r\n", WantError: true},
		// NextPart reports bare EOF when no headers follow an initial boundary.
		{Name: "boundary without headers", Body: "--sample\r\n"},
		{Name: "invalid header", Body: "--sample\r\ninvalid header\r\n\r\n", WantError: true},
		{Name: "field without closing boundary", Body: field, WantError: true},
		{Name: "unfinished closing boundary", Body: field + "\r\n--sample", WantError: true},
	}
}
