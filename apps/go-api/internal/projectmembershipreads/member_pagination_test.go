package projectmembershipreads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemberCursorKeepsLegacyAcceptedSpellings(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		size int
		want MemberCursor
	}{
		{"100:1:0", 100, MemberCursor{Value: 100, Page: 1}},
		{"100.0:1:2", 100, MemberCursor{Value: 100, Page: 1, Previous: true}},
		{" 2 :3:-1", 100, MemberCursor{Value: 2, Page: 3, Previous: true}},
		{"0:1:0", 100, MemberCursor{Value: 0, Page: 1}},
		{"1:1:0", 100, MemberCursor{Value: 1, Page: 1}},
		{"2:-1:1", 2, MemberCursor{Value: 2, Page: -1, Previous: true}},
	} {
		got, err := ParseMemberCursor(&tc.raw, tc.size)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
	got, err := ParseMemberCursor(nil, 20)
	require.NoError(t, err)
	assert.Equal(t, MemberCursor{Value: 20}, got)
	for _, raw := range []string{"", ":1:0", "1:0", "1:0:0:0", "many:0:0", "1:many:0", "1:0:many", "1: 1:0"} {
		_, err = ParseMemberCursor(&raw, 100)
		assert.Error(t, err, raw)
	}
}

func TestMemberPageMatchesPinnedEnvelopeAcrossBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name              string
		items             []string
		total, size, page int
		want              string
	}{
		{"first with lookahead", []string{"a", "b", "c"}, 3, 2, 0, `{"count":2,"extra_stats":null,"grouped_by":null,"next_cursor":"2:1:0","next_page_results":true,"prev_cursor":"2:-1:1","prev_page_results":false,"results":["a","b"],"sub_grouped_by":null,"total_count":3,"total_pages":2,"total_results":3}`},
		{"last", []string{"c"}, 3, 2, 1, `{"count":1,"extra_stats":null,"grouped_by":null,"next_cursor":"2:2:0","next_page_results":false,"prev_cursor":"2:0:1","prev_page_results":true,"results":["c"],"sub_grouped_by":null,"total_count":3,"total_pages":2,"total_results":3}`},
		{"empty", nil, 0, 2, 0, `{"count":0,"extra_stats":null,"grouped_by":null,"next_cursor":"2:1:0","next_page_results":false,"prev_cursor":"2:-1:1","prev_page_results":false,"results":[],"sub_grouped_by":null,"total_count":0,"total_pages":0,"total_results":0}`},
		{"past end", nil, 3, 2, 4, `{"count":0,"extra_stats":null,"grouped_by":null,"next_cursor":"2:5:0","next_page_results":false,"prev_cursor":"2:3:1","prev_page_results":true,"results":[],"sub_grouped_by":null,"total_count":3,"total_pages":2,"total_results":3}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, err := NewMemberPage(tc.items, tc.total, tc.size, tc.page)
			require.NoError(t, err)
			body, err := json.Marshal(page)
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(body))
		})
	}
	_, err := NewMemberPage([]string{}, 0, 0, 0)
	assert.ErrorIs(t, err, ErrInvalidMemberQuery)
	_, err = NewMemberPage([]string{}, -1, 20, 0)
	assert.ErrorIs(t, err, ErrInvalidMemberQuery)
}
