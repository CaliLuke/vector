package notionsync

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type richText struct {
	Plain string `json:"plain_text"`
	Text  struct {
		Content string `json:"content"`
		Link    *struct {
			URL string `json:"url"`
		} `json:"link"`
	} `json:"text"`
	Annotations struct {
		Bold   bool `json:"bold"`
		Italic bool `json:"italic"`
		Code   bool `json:"code"`
		Strike bool `json:"strikethrough"`
	} `json:"annotations"`
}
type notionProperty struct {
	ID     string     `json:"id"`
	Type   string     `json:"type"`
	Title  []richText `json:"title"`
	Text   []richText `json:"rich_text"`
	Status *struct {
		Name string `json:"name"`
	} `json:"status"`
	Select *struct {
		Name string `json:"name"`
	} `json:"select"`
	Date *struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"date"`
	People []struct {
		ID string `json:"id"`
	} `json:"people"`
}
type notionPage struct {
	ID         string                    `json:"id"`
	URL        string                    `json:"url"`
	Archived   bool                      `json:"archived"`
	InTrash    bool                      `json:"in_trash"`
	EditedAt   string                    `json:"last_edited_time"`
	Properties map[string]notionProperty `json:"properties"`
	Parent     struct {
		DataSourceID string `json:"data_source_id"`
	} `json:"parent"`
}

type Notion struct {
	api             *apiClient
	followRelations bool
}

func NewNotion(token string, followRelations bool) (*Notion, error) {
	if token == "" {
		return nil, fmt.Errorf("NOTION_TOKEN is required")
	}
	return &Notion{followRelations: followRelations, api: newAPI("https://api.notion.com/v1", http.Header{
		"Authorization": {"Bearer " + token}, "Notion-Version": {"2026-03-11"},
	}, 350*time.Millisecond)}, nil
}

func (n *Notion) Page(ctx context.Context, id string) (Page, error) {
	id, err := PageID(id)
	if err != nil {
		return Page{}, err
	}
	var raw notionPage
	if err = n.api.request(ctx, http.MethodGet, "/pages/"+id, nil, &raw); err != nil {
		return Page{}, err
	}
	page := Page{ID: id, URL: raw.URL, Archived: raw.Archived || raw.InTrash, EditedAt: raw.EditedAt, Relations: map[string][]string{}}
	for name, property := range raw.Properties {
		if property.Type == "title" {
			page.Title = plainText(property.Title)
		}
		if name == "Status" && property.Status != nil {
			page.Status = property.Status.Name
		}
		if name == "Priority" && property.Select != nil {
			page.Priority = property.Select.Name
		}
	}
	for _, name := range []string{"🏗️ Projects", "🧑‍💻 Tasks", "Parent", "Project"} {
		if !n.followRelations {
			break
		}
		property, ok := raw.Properties[name]
		if !ok {
			continue
		}
		if property.Type != "relation" {
			return Page{}, fmt.Errorf("%s property %s must be a relation", id, name)
		}
		if name == "🏗️ Projects" || name == "🧑‍💻 Tasks" {
			if accessErr := n.checkRelationAccess(ctx, raw.Parent.DataSourceID, name); accessErr != nil {
				return Page{}, accessErr
			}
		}
		ids, relationErr := n.relation(ctx, id, property.ID)
		if relationErr != nil {
			return Page{}, relationErr
		}
		page.Relations[name] = ids
	}
	for _, name := range []string{"Assigned", "Who: Project Owner", "Assign"} {
		if property, ok := raw.Properties[name]; ok {
			for _, person := range property.People {
				page.People = append(page.People, person.ID)
			}
			break
		}
	}
	for _, name := range []string{"Timeline", "Date", "ETA"} {
		if date := raw.Properties[name].Date; date != nil {
			page.Start, page.End = day(date.Start), day(date.End)
			if name == "ETA" {
				page.Start, page.End = "", day(date.Start)
			}
			break
		}
	}
	for _, name := range []string{"Details", "Description", "Summary", "Notes"} {
		if text := raw.Properties[name].Text; len(text) > 0 {
			page.HTML += "<p>" + renderRichText(text) + "</p>"
		}
	}
	if !page.Archived {
		budget := 5000
		body, bodyErr := n.blocks(ctx, id, 0, &budget, &page.Warnings)
		if bodyErr != nil {
			return Page{}, bodyErr
		}
		page.HTML += body
	}
	// Construct links from the authenticated page identity, not source-supplied HTML.
	page.URL = "https://www.notion.so/" + strings.ReplaceAll(id, "-", "")
	page.HTML += "<p><a href=\"" + page.URL + "\">Source in Notion</a></p>"
	return page, nil
}

// An inaccessible related database can look like an empty relation. Verify it
// explicitly before interpreting an empty list as a source-side removal.
func (n *Notion) checkRelationAccess(ctx context.Context, sourceID, name string) error {
	if sourceID == "" {
		return fmt.Errorf("Notion page has no data source; expected API version 2026-03-11")
	}
	var source struct {
		Properties map[string]struct {
			Relation struct {
				DataSourceID string `json:"data_source_id"`
			} `json:"relation"`
		} `json:"properties"`
	}
	if err := n.api.request(ctx, http.MethodGet, "/data_sources/"+sourceID, nil, &source); err != nil {
		return err
	}
	target := source.Properties[name].Relation.DataSourceID
	if target == "" {
		return fmt.Errorf("Notion relation %s has no accessible target data source", name)
	}
	return n.api.request(ctx, http.MethodGet, "/data_sources/"+target, nil, nil)
}

func day(value string) string {
	if len(value) > 10 {
		return value[:10]
	}
	return value
}
func plainText(values []richText) string {
	var out strings.Builder
	for _, value := range values {
		if value.Plain != "" {
			out.WriteString(value.Plain)
		} else {
			out.WriteString(value.Text.Content)
		}
	}
	return out.String()
}

func (n *Notion) relation(ctx context.Context, pageID, propertyID string) ([]string, error) {
	// Notion returns URL-encoded property IDs, while PathEscape expects raw text.
	if decoded, err := url.PathUnescape(propertyID); err == nil {
		propertyID = decoded
	}
	path := "/pages/" + pageID + "/properties/" + url.PathEscape(propertyID)
	var ids []string
	cursor := ""
	seen := map[string]bool{}
	for {
		var result struct {
			Results []struct {
				Relation struct {
					ID string `json:"id"`
				} `json:"relation"`
			} `json:"results"`
			HasMore bool   `json:"has_more"`
			Next    string `json:"next_cursor"`
		}
		query := "?page_size=100"
		if cursor != "" {
			query += "&start_cursor=" + url.QueryEscape(cursor)
		}
		if err := n.api.request(ctx, http.MethodGet, path+query, nil, &result); err != nil {
			return nil, err
		}
		for _, item := range result.Results {
			id, err := PageID(item.Relation.ID)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		if !result.HasMore {
			return ids, nil
		}
		if result.Next == "" || seen[result.Next] {
			return nil, fmt.Errorf("Notion relation pagination did not advance")
		}
		seen[result.Next] = true
		cursor = result.Next
	}
}

func renderRichText(values []richText) string {
	var out strings.Builder
	for _, value := range values {
		text := html.EscapeString(plainText([]richText{value}))
		if value.Annotations.Code {
			text = "<code>" + text + "</code>"
		}
		if value.Annotations.Bold {
			text = "<strong>" + text + "</strong>"
		}
		if value.Annotations.Italic {
			text = "<em>" + text + "</em>"
		}
		if value.Annotations.Strike {
			text = "<s>" + text + "</s>"
		}
		if link := value.Text.Link; link != nil {
			if u, err := url.Parse(link.URL); err == nil && (u.Scheme == "https" || u.Scheme == "http" || u.Scheme == "mailto") {
				text = "<a href=\"" + html.EscapeString(link.URL) + "\">" + text + "</a>"
			}
		}
		out.WriteString(text)
	}
	return out.String()
}
