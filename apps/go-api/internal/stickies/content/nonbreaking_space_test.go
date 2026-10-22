package content

import "testing"

func TestValidateHTMLMatchesNH3NonbreakingSpaces(t *testing.T) {
	// Captured with nh3 0.2.18 in the pinned Django source image. Exercise
	// decoded text and attributes as well as an already escaped entity.
	for _, testCase := range []struct{ input, want string }{
		{"<p>\u00a0</p>", "<p>&nbsp;</p>"},
		{"<p title=\"\u00a0\">A&amp;B</p>", "<p title=\"&nbsp;\">A&amp;B</p>"},
		{"<p>&nbsp; &amp;nbsp;</p>", "<p>&nbsp; &amp;nbsp;</p>"},
	} {
		got, present, err := ValidateHTML(testCase.input)
		if err != nil || !present || got != testCase.want {
			t.Errorf("ValidateHTML(%q) = %q,%t,%v; want %q,true,nil", testCase.input, got, present, err, testCase.want)
		}
	}
}
