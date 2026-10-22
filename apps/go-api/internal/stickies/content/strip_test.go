package content

import "testing"

func TestStripTagsMatchesDjangoCharacterization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "entities", input: "x &amp; y &#65; &#x41; &bogus; z", want: "x & y A A &bogus; z"},
		{name: "ordinary and void tags", input: "<p>one<br>two", want: "onetwo"},
		{name: "script body preserved raw", input: "<script>if (a < b && c > d) &amp;</script>X", want: "if (a < b && c > d) &amp;X"},
		{name: "style body preserved raw", input: "<style>a>b&amp;c</style>X", want: "a>b&amp;cX"},
		{name: "unclosed tags", input: "<p><b>unclosed", want: "unclosed"},
		{name: "broken markup", input: "plain < broken > tail", want: "plain < broken > tail"},
		{name: "comment", input: "a<!--c-->b", want: "ab"},
		{name: "declaration", input: "a<!DOCTYPE html>b", want: "ab"},
		{name: "processing instruction", input: "a<?pi data?>b", want: "ab"},
		{name: "malformed end tag", input: "a</ nottag>b", want: "ab"},
		{name: "textarea", input: "<textarea>&amp;<b>x</b></textarea>", want: "&x"},
		{name: "title", input: "<title>&amp;<b>x</b></title>", want: "&x"},
		{name: "less than text", input: "a < b", want: "a < b"},
		{name: "unfinished start tag", input: "a <b", want: "a "},
		{name: "empty angle brackets", input: "a <> b", want: "a <> b"},
		{name: "legacy named reference", input: "a &notit; b", want: "a ¬it; b"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := StripTags(test.input); got != test.want {
				t.Fatalf("StripTags() = %q, want %q", got, test.want)
			}
		})
	}
}

func FuzzStripTagsDoesNotPanic(f *testing.F) {
	f.Add("<p>Hello &amp; goodbye</p>")
	f.Add("<script>if (a < b) x()</script>")
	f.Add("plain < broken > tail")
	f.Fuzz(func(t *testing.T, input string) {
		_ = StripTags(input)
	})
}
