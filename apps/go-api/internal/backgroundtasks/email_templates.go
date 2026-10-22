package backgroundtasks

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"regexp"
	"strings"

	htmltemplate "html/template"

	"golang.org/x/net/html"
)

// emailTemplates contains the source project's reviewed email markup. Keeping
// the exact HTML in the Go binary prevents deployment-time dependence on Django
// or a template filesystem.
//
//go:embed emailtemplates
var emailTemplates embed.FS

var (
	djangoVariablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
	styleBlockPattern     = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style\s*>`)
	emptyLinesPattern     = regexp.MustCompile(`\n[\t \r]*\n[\t \r]*\n+`)
)

func renderPlaneEmail(templateName string, values map[string]string) (string, string, error) {
	content, err := emailTemplates.ReadFile("emailtemplates/" + templateName)
	if err != nil {
		return "", "", fmt.Errorf("read email template %q: %w", templateName, err)
	}
	translated := djangoVariablePattern.ReplaceAllString(string(content), "{{.$1}}")
	template, err := htmltemplate.New(templateName).Option("missingkey=zero").Parse(translated)
	if err != nil {
		return "", "", fmt.Errorf("parse email template %q: %w", templateName, err)
	}
	var body bytes.Buffer
	if err := template.Execute(&body, values); err != nil {
		return "", "", fmt.Errorf("render email template %q: %w", templateName, err)
	}
	text, err := planeEmailPlainText(body.String())
	if err != nil {
		return "", "", fmt.Errorf("render plain text for email template %q: %w", templateName, err)
	}
	return body.String(), text, nil
}

func planeEmailPlainText(markup string) (string, error) {
	markup = styleBlockPattern.ReplaceAllString(markup, "")
	root, err := html.Parse(strings.NewReader(markup))
	if err != nil {
		return "", err
	}
	var text strings.Builder
	var walk func(*html.Node) error
	walk = func(node *html.Node) error {
		if node.Type == html.TextNode {
			if _, err := io.WriteString(&text, node.Data); err != nil {
				return err
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return "", err
	}
	content := emptyLinesPattern.ReplaceAllString(text.String(), "\n\n")
	return "\n\n" + strings.TrimSpace(content) + "\n\n", nil
}
