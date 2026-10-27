package backgroundtasks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMilestoneEmailTitlesRemainPlainTextIncludingMentionMarkup(t *testing.T) {
	title := `<mention-component entity_identifier="not-a-user">Release</mention-component>`
	activities := []IssueEmailActivity{{Field: "milestone", NewValue: title}, {Field: "milestone", OldValue: "None"}}
	processed, err := (&issueEmailStore{}).processMentionValues(t.Context(), activities)
	require.NoError(t, err)
	require.Equal(t, activities, processed)
	html, text, err := renderIssueEmail(issueEmailTemplateData{Activities: processed})
	require.NoError(t, err)
	require.Contains(t, html, "&lt;mention-component")
	require.NotContains(t, html, "<mention-component")
	require.Contains(t, text, title)
	require.Contains(t, text, "removed the milestone")
	require.Contains(t, text, "None")
}
