package requestinput

import "testing"

func TestNativeURLAcceptsSupportedAbsoluteLinks(t *testing.T) {
	for _, value := range []string{
		"https://example.com/path?q=value#section",
		"http://localhost:8000/",
		"http://intranet/path",
		"https://例え.テスト/path",
		"https://[2001:db8::1]:8443/",
		"ftp://user:password@example.com/file",
		"ftps://example.com/",
		"https://example.com/encoded%20path",
	} {
		t.Run(value, func(t *testing.T) {
			if !ValidURL(value) {
				t.Fatal("valid link rejected")
			}
		})
	}
}

func TestNativeURLRejectsMalformedAndUnsupportedLinks(t *testing.T) {
	for _, value := range []string{
		"", "/relative/path", "javascript:alert(1)", "file:///tmp/file", "https:///path",
		"https://example.com:65536/", "https://example.com:port/", "https://example.com:/",
		"https://[not-an-ip]/", "https://example.com/raw%", "https://example.com/raw space",
		"https://example.com/\nheader", "https://example.com/\x00", "https://example.com/\u00a0",
	} {
		t.Run(value, func(t *testing.T) {
			if ValidURL(value) {
				t.Fatal("invalid link accepted")
			}
		})
	}
}
