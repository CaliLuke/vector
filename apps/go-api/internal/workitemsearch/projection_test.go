package workitemsearch

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestProjectionSanitizesContentButHashesExactSavedValue(t *testing.T) {
	saved := `<p>Alpha &amp; βeta</p><script>alert(1)</script>`
	text, hash, err := project(saved)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(saved))
	if !strings.Contains(text, "Alpha & βeta") || strings.Contains(text, "alert") || strings.Contains(text, "<") || hash != hex.EncodeToString(digest[:]) {
		t.Fatalf("projection=%q hash=%q", text, hash)
	}
}

func TestProjectionRejectsMalformedLegacyValuesAndSkipsEmptyContent(t *testing.T) {
	if _, _, err := project(42); err == nil {
		t.Fatal("numeric legacy rich text accepted")
	}
	text, _, err := project("<p></p>")
	if err != nil || text != "" {
		t.Fatalf("empty rich text=%q %v", text, err)
	}
}

func TestEscapeLikePreservesLiteralSearchMetacharacters(t *testing.T) {
	if got := EscapeLike(`a%b_c\d`); got != `a\%b\_c\\d` {
		t.Fatalf("literal search=%q", got)
	}
}
