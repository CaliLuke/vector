package workitemsearch

import (
	"fmt"
	"strings"
)

// JoinSQL adds at most one current field match to an already-authorized item
// query. The caller must apply IssueVisibleSQL in the same SELECT before LIMIT.
// Arguments are trusted SQL expressions, never user-provided fragments.
func JoinSQL(issue, term string) string {
	return fmt.Sprintf(` LEFT JOIN LATERAL (
 SELECT jsonb_build_object('property_id',sp.property_id,'property_title',definition.definition->>'title',
 'text',substring(sp.search_text from greatest(1,strpos(lower(sp.search_text),lower(%[2]s))-60) for 240)) AS snippet
 FROM work_item_property_search sp
 JOIN work_item_properties definition ON definition.id=sp.property_id AND definition.workspace_id=sp.workspace_id
 JOIN work_item_type_properties attachment ON attachment.property_id=definition.id AND attachment.type_id=%[1]s.type_id
 JOIN work_item_type_settings settings ON settings.workspace_id=sp.workspace_id AND settings.enabled
 WHERE %[2]s<>'' AND sp.issue_id=%[1]s.id AND sp.workspace_id=%[1]s.workspace_id
 AND definition.definition->>'kind'='Rich text'
 AND COALESCE((effective_work_item_property(definition.definition,%[1]s.project_id,%[1]s.type_id)->>'active')::boolean,false)
 AND jsonb_typeof(%[1]s.custom_properties->sp.property_id::text)='string'
 AND sp.source_hash=encode(sha256(convert_to(%[1]s.custom_properties->>sp.property_id::text,'UTF8')),'hex')
 AND strpos(lower(sp.search_text),lower(%[2]s))>0
 ORDER BY sp.property_id LIMIT 1
 ) property_match ON true `, issue, term)
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func EscapeLike(term string) string { return likeEscaper.Replace(term) }
