// Package design contains the reverse-generated cycle lifecycle HTTP contract.
// The selected operations come from the committed full-server OpenAPI contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/cyclelifecycle/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var ImportedPatchedPlaneAPISerializersCycleCycleupdateserializerRequestRequest = Type("PatchedPlaneApiSerializersCycleCycleupdateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for updating cycles with enhanced ownership management.\n\nExtends cycle creation with update-specific features including ownership\nassignment and modification tracking for cycle lifecycle management.")
	Attribute("description", String, func() {
		Title("Cycle Description")
	})
	Attribute("end_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("name", String, func() {
		Title("Cycle Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("owned_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("timezone", ImportedTimezoneEnum)
})

var ImportedPatchedPlaneAppSerializersCycleCyclewriteserializerRequestRequest = Type("PatchedPlaneAppSerializersCycleCyclewriteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description", String, func() {
		Title("Cycle Description")
	})
	Attribute("end_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String, func() {
		Title("Cycle Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("progress_snapshot", Any, func() {
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("timezone", ImportedTimezoneEnum)
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("version", Int, func() {
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
	Attribute("view_props", Any, func() {
	})
})

var ImportedPlaneAPISerializersCycleCyclecreateserializerRequestRequest = Type("PlaneApiSerializersCycleCyclecreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for creating cycles with timezone handling and date validation.\n\nManages cycle creation including project timezone conversion, date range validation,\nand UTC normalization for time-bound iteration planning and sprint management.")
	Attribute("description", String, func() {
		Title("Cycle Description")
	})
	Attribute("end_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("name", String, func() {
		Title("Cycle Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("owned_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("timezone", ImportedTimezoneEnum)
	Required("name")
})

var ImportedPlaneAPISerializersCycleCycleserializerResponse = Type("PlaneApiSerializersCycleCycleserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Cycle serializer with comprehensive project metrics and time tracking.\n\nProvides cycle details including work item counts by status, progress estimates,\nand time-bound iteration data for project management and sprint planning.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("backlog_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("cancelled_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_estimates", Float64, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("Cycle Description")
	})
	Attribute("end_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String, func() {
		Title("Cycle Name")
		MaxLength(255)
	})
	Attribute("owned_by", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("progress_snapshot", Any, func() {
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("started_estimates", Float64, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("started_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("timezone", ImportedTimezoneEnum)
	Attribute("total_estimates", Float64, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("total_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("unstarted_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("version", Int, func() {
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("description")
})

var ImportedPlaneAppSerializersCycleCycleserializerResponse = Type("PlaneAppSerializersCycleCycleserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("assignee_ids", ArrayOf(String), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("Cycle Description")
	})
	Attribute("end_date", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("external_id", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("external_source", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_favorite", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("logo_props", Any, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Cycle Name")
		Meta("openapi:readOnly", "true")
	})
	Attribute("owned_by_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("progress_snapshot", Any, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("project_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("status", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("version", Int, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("total_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("view_props", Any, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace_id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("assignee_ids", "description")
})

// The inherited app PUT action serializes the read-only CycleSerializer
// projection, which differs from the values() projection used by create/PATCH.
var CycleAppReadonlyPutResponse = Type("CycleAppReadonlyPutResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("cancelled_issues", Int, func() { Meta("openapi:format", "") })
	Attribute("completed_issues", Int, func() { Meta("openapi:format", "") })
	Attribute("description", String, func() { Title("Cycle Description") })
	Attribute("end_date", String, func() { Nullable(); Format(FormatDateTime) })
	Attribute("external_id", String, func() { Nullable() })
	Attribute("external_source", String, func() { Nullable() })
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("is_favorite", Boolean)
	Attribute("logo_props", Any)
	Attribute("name", String, func() { Title("Cycle Name") })
	Attribute("owned_by_id", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("progress_snapshot", Any)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() { Nullable(); Format(FormatDateTime) })
	Attribute("status", String)
	Attribute("total_issues", Int, func() { Meta("openapi:format", "") })
	Attribute("view_props", Any)
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Required("description")
})

var ImportedPlaneAppSerializersCycleCyclewriteserializerRequestRequest = Type("PlaneAppSerializersCycleCyclewriteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("description", String, func() {
		Title("Cycle Description")
	})
	Attribute("end_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String, func() {
		Title("Cycle Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("progress_snapshot", Any, func() {
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("timezone", ImportedTimezoneEnum)
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("version", Int, func() {
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
	Attribute("view_props", Any, func() {
	})
	Required("name")
})

var ImportedRestFrameworkSerializersArchiveresponseResponse = Type("RestFrameworkSerializersArchiveresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String)
	Required("archived_at")
})

var ImportedRestFrameworkSerializersCycledatecheckrequestRequestRequest = Type("RestFrameworkSerializersCycledatecheckrequestRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("cycle_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("end_date", String, func() {
		Format(FormatDate)
	})
	Attribute("start_date", String, func() {
		Format(FormatDate)
	})
	Required("end_date", "start_date")
})

var ImportedRestFrameworkSerializersCycledatecheckresponseResponse = Type("RestFrameworkSerializersCycledatecheckresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("error", String)
	Attribute("status", Boolean)
	Required("status")
})

var ImportedTimezoneEnum = Type("TimezoneEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `Africa/Abidjan` - Africa/Abidjan\n* `Africa/Accra` - Africa/Accra\n* `Africa/Addis_Ababa` - Africa/Addis_Ababa\n* `Africa/Algiers` - Africa/Algiers\n* `Africa/Asmara` - Africa/Asmara\n* `Africa/Bamako` - Africa/Bamako\n* `Africa/Bangui` - Africa/Bangui\n* `Africa/Banjul` - Africa/Banjul\n* `Africa/Bissau` - Africa/Bissau\n* `Africa/Blantyre` - Africa/Blantyre\n* `Africa/Brazzaville` - Africa/Brazzaville\n* `Africa/Bujumbura` - Africa/Bujumbura\n* `Africa/Cairo` - Africa/Cairo\n* `Africa/Casablanca` - Africa/Casablanca\n* `Africa/Ceuta` - Africa/Ceuta\n* `Africa/Conakry` - Africa/Conakry\n* `Africa/Dakar` - Africa/Dakar\n* `Africa/Dar_es_Salaam` - Africa/Dar_es_Salaam\n* `Africa/Djibouti` - Africa/Djibouti\n* `Africa/Douala` - Africa/Douala\n* `Africa/El_Aaiun` - Africa/El_Aaiun\n* `Africa/Freetown` - Africa/Freetown\n* `Africa/Gaborone` - Africa/Gaborone\n* `Africa/Harare` - Africa/Harare\n* `Africa/Johannesburg` - Africa/Johannesburg\n* `Africa/Juba` - Africa/Juba\n* `Africa/Kampala` - Africa/Kampala\n* `Africa/Khartoum` - Africa/Khartoum\n* `Africa/Kigali` - Africa/Kigali\n* `Africa/Kinshasa` - Africa/Kinshasa\n* `Africa/Lagos` - Africa/Lagos\n* `Africa/Libreville` - Africa/Libreville\n* `Africa/Lome` - Africa/Lome\n* `Africa/Luanda` - Africa/Luanda\n* `Africa/Lubumbashi` - Africa/Lubumbashi\n* `Africa/Lusaka` - Africa/Lusaka\n* `Africa/Malabo` - Africa/Malabo\n* `Africa/Maputo` - Africa/Maputo\n* `Africa/Maseru` - Africa/Maseru\n* `Africa/Mbabane` - Africa/Mbabane\n* `Africa/Mogadishu` - Africa/Mogadishu\n* `Africa/Monrovia` - Africa/Monrovia\n* `Africa/Nairobi` - Africa/Nairobi\n* `Africa/Ndjamena` - Africa/Ndjamena\n* `Africa/Niamey` - Africa/Niamey\n* `Africa/Nouakchott` - Africa/Nouakchott\n* `Africa/Ouagadougou` - Africa/Ouagadougou\n* `Africa/Porto-Novo` - Africa/Porto-Novo\n* `Africa/Sao_Tome` - Africa/Sao_Tome\n* `Africa/Tripoli` - Africa/Tripoli\n* `Africa/Tunis` - Africa/Tunis\n* `Africa/Windhoek` - Africa/Windhoek\n* `America/Adak` - America/Adak\n* `America/Anchorage` - America/Anchorage\n* `America/Anguilla` - America/Anguilla\n* `America/Antigua` - America/Antigua\n* `America/Araguaina` - America/Araguaina\n* `America/Argentina/Buenos_Aires` - America/Argentina/Buenos_Aires\n* `America/Argentina/Catamarca` - America/Argentina/Catamarca\n* `America/Argentina/Cordoba` - America/Argentina/Cordoba\n* `America/Argentina/Jujuy` - America/Argentina/Jujuy\n* `America/Argentina/La_Rioja` - America/Argentina/La_Rioja\n* `America/Argentina/Mendoza` - America/Argentina/Mendoza\n* `America/Argentina/Rio_Gallegos` - America/Argentina/Rio_Gallegos\n* `America/Argentina/Salta` - America/Argentina/Salta\n* `America/Argentina/San_Juan` - America/Argentina/San_Juan\n* `America/Argentina/San_Luis` - America/Argentina/San_Luis\n* `America/Argentina/Tucuman` - America/Argentina/Tucuman\n* `America/Argentina/Ushuaia` - America/Argentina/Ushuaia\n* `America/Aruba` - America/Aruba\n* `America/Asuncion` - America/Asuncion\n* `America/Atikokan` - America/Atikokan\n* `America/Bahia` - America/Bahia\n* `America/Bahia_Banderas` - America/Bahia_Banderas\n* `America/Barbados` - America/Barbados\n* `America/Belem` - America/Belem\n* `America/Belize` - America/Belize\n* `America/Blanc-Sablon` - America/Blanc-Sablon\n* `America/Boa_Vista` - America/Boa_Vista\n* `America/Bogota` - America/Bogota\n* `America/Boise` - America/Boise\n* `America/Cambridge_Bay` - America/Cambridge_Bay\n* `America/Campo_Grande` - America/Campo_Grande\n* `America/Cancun` - America/Cancun\n* `America/Caracas` - America/Caracas\n* `America/Cayenne` - America/Cayenne\n* `America/Cayman` - America/Cayman\n* `America/Chicago` - America/Chicago\n* `America/Chihuahua` - America/Chihuahua\n* `America/Ciudad_Juarez` - America/Ciudad_Juarez\n* `America/Costa_Rica` - America/Costa_Rica\n* `America/Creston` - America/Creston\n* `America/Cuiaba` - America/Cuiaba\n* `America/Curacao` - America/Curacao\n* `America/Danmarkshavn` - America/Danmarkshavn\n* `America/Dawson` - America/Dawson\n* `America/Dawson_Creek` - America/Dawson_Creek\n* `America/Denver` - America/Denver\n* `America/Detroit` - America/Detroit\n* `America/Dominica` - America/Dominica\n* `America/Edmonton` - America/Edmonton\n* `America/Eirunepe` - America/Eirunepe\n* `America/El_Salvador` - America/El_Salvador\n* `America/Fort_Nelson` - America/Fort_Nelson\n* `America/Fortaleza` - America/Fortaleza\n* `America/Glace_Bay` - America/Glace_Bay\n* `America/Goose_Bay` - America/Goose_Bay\n* `America/Grand_Turk` - America/Grand_Turk\n* `America/Grenada` - America/Grenada\n* `America/Guadeloupe` - America/Guadeloupe\n* `America/Guatemala` - America/Guatemala\n* `America/Guayaquil` - America/Guayaquil\n* `America/Guyana` - America/Guyana\n* `America/Halifax` - America/Halifax\n* `America/Havana` - America/Havana\n* `America/Hermosillo` - America/Hermosillo\n* `America/Indiana/Indianapolis` - America/Indiana/Indianapolis\n* `America/Indiana/Knox` - America/Indiana/Knox\n* `America/Indiana/Marengo` - America/Indiana/Marengo\n* `America/Indiana/Petersburg` - America/Indiana/Petersburg\n* `America/Indiana/Tell_City` - America/Indiana/Tell_City\n* `America/Indiana/Vevay` - America/Indiana/Vevay\n* `America/Indiana/Vincennes` - America/Indiana/Vincennes\n* `America/Indiana/Winamac` - America/Indiana/Winamac\n* `America/Inuvik` - America/Inuvik\n* `America/Iqaluit` - America/Iqaluit\n* `America/Jamaica` - America/Jamaica\n* `America/Juneau` - America/Juneau\n* `America/Kentucky/Louisville` - America/Kentucky/Louisville\n* `America/Kentucky/Monticello` - America/Kentucky/Monticello\n* `America/Kralendijk` - America/Kralendijk\n* `America/La_Paz` - America/La_Paz\n* `America/Lima` - America/Lima\n* `America/Los_Angeles` - America/Los_Angeles\n* `America/Lower_Princes` - America/Lower_Princes\n* `America/Maceio` - America/Maceio\n* `America/Managua` - America/Managua\n* `America/Manaus` - America/Manaus\n* `America/Marigot` - America/Marigot\n* `America/Martinique` - America/Martinique\n* `America/Matamoros` - America/Matamoros\n* `America/Mazatlan` - America/Mazatlan\n* `America/Menominee` - America/Menominee\n* `America/Merida` - America/Merida\n* `America/Metlakatla` - America/Metlakatla\n* `America/Mexico_City` - America/Mexico_City\n* `America/Miquelon` - America/Miquelon\n* `America/Moncton` - America/Moncton\n* `America/Monterrey` - America/Monterrey\n* `America/Montevideo` - America/Montevideo\n* `America/Montserrat` - America/Montserrat\n* `America/Nassau` - America/Nassau\n* `America/New_York` - America/New_York\n* `America/Nome` - America/Nome\n* `America/Noronha` - America/Noronha\n* `America/North_Dakota/Beulah` - America/North_Dakota/Beulah\n* `America/North_Dakota/Center` - America/North_Dakota/Center\n* `America/North_Dakota/New_Salem` - America/North_Dakota/New_Salem\n* `America/Nuuk` - America/Nuuk\n* `America/Ojinaga` - America/Ojinaga\n* `America/Panama` - America/Panama\n* `America/Paramaribo` - America/Paramaribo\n* `America/Phoenix` - America/Phoenix\n* `America/Port-au-Prince` - America/Port-au-Prince\n* `America/Port_of_Spain` - America/Port_of_Spain\n* `America/Porto_Velho` - America/Porto_Velho\n* `America/Puerto_Rico` - America/Puerto_Rico\n* `America/Punta_Arenas` - America/Punta_Arenas\n* `America/Rankin_Inlet` - America/Rankin_Inlet\n* `America/Recife` - America/Recife\n* `America/Regina` - America/Regina\n* `America/Resolute` - America/Resolute\n* `America/Rio_Branco` - America/Rio_Branco\n* `America/Santarem` - America/Santarem\n* `America/Santiago` - America/Santiago\n* `America/Santo_Domingo` - America/Santo_Domingo\n* `America/Sao_Paulo` - America/Sao_Paulo\n* `America/Scoresbysund` - America/Scoresbysund\n* `America/Sitka` - America/Sitka\n* `America/St_Barthelemy` - America/St_Barthelemy\n* `America/St_Johns` - America/St_Johns\n* `America/St_Kitts` - America/St_Kitts\n* `America/St_Lucia` - America/St_Lucia\n* `America/St_Thomas` - America/St_Thomas\n* `America/St_Vincent` - America/St_Vincent\n* `America/Swift_Current` - America/Swift_Current\n* `America/Tegucigalpa` - America/Tegucigalpa\n* `America/Thule` - America/Thule\n* `America/Tijuana` - America/Tijuana\n* `America/Toronto` - America/Toronto\n* `America/Tortola` - America/Tortola\n* `America/Vancouver` - America/Vancouver\n* `America/Whitehorse` - America/Whitehorse\n* `America/Winnipeg` - America/Winnipeg\n* `America/Yakutat` - America/Yakutat\n* `Antarctica/Casey` - Antarctica/Casey\n* `Antarctica/Davis` - Antarctica/Davis\n* `Antarctica/DumontDUrville` - Antarctica/DumontDUrville\n* `Antarctica/Macquarie` - Antarctica/Macquarie\n* `Antarctica/Mawson` - Antarctica/Mawson\n* `Antarctica/McMurdo` - Antarctica/McMurdo\n* `Antarctica/Palmer` - Antarctica/Palmer\n* `Antarctica/Rothera` - Antarctica/Rothera\n* `Antarctica/Syowa` - Antarctica/Syowa\n* `Antarctica/Troll` - Antarctica/Troll\n* `Antarctica/Vostok` - Antarctica/Vostok\n* `Arctic/Longyearbyen` - Arctic/Longyearbyen\n* `Asia/Aden` - Asia/Aden\n* `Asia/Almaty` - Asia/Almaty\n* `Asia/Amman` - Asia/Amman\n* `Asia/Anadyr` - Asia/Anadyr\n* `Asia/Aqtau` - Asia/Aqtau\n* `Asia/Aqtobe` - Asia/Aqtobe\n* `Asia/Ashgabat` - Asia/Ashgabat\n* `Asia/Atyrau` - Asia/Atyrau\n* `Asia/Baghdad` - Asia/Baghdad\n* `Asia/Bahrain` - Asia/Bahrain\n* `Asia/Baku` - Asia/Baku\n* `Asia/Bangkok` - Asia/Bangkok\n* `Asia/Barnaul` - Asia/Barnaul\n* `Asia/Beirut` - Asia/Beirut\n* `Asia/Bishkek` - Asia/Bishkek\n* `Asia/Brunei` - Asia/Brunei\n* `Asia/Chita` - Asia/Chita\n* `Asia/Choibalsan` - Asia/Choibalsan\n* `Asia/Colombo` - Asia/Colombo\n* `Asia/Damascus` - Asia/Damascus\n* `Asia/Dhaka` - Asia/Dhaka\n* `Asia/Dili` - Asia/Dili\n* `Asia/Dubai` - Asia/Dubai\n* `Asia/Dushanbe` - Asia/Dushanbe\n* `Asia/Famagusta` - Asia/Famagusta\n* `Asia/Gaza` - Asia/Gaza\n* `Asia/Hebron` - Asia/Hebron\n* `Asia/Ho_Chi_Minh` - Asia/Ho_Chi_Minh\n* `Asia/Hong_Kong` - Asia/Hong_Kong\n* `Asia/Hovd` - Asia/Hovd\n* `Asia/Irkutsk` - Asia/Irkutsk\n* `Asia/Jakarta` - Asia/Jakarta\n* `Asia/Jayapura` - Asia/Jayapura\n* `Asia/Jerusalem` - Asia/Jerusalem\n* `Asia/Kabul` - Asia/Kabul\n* `Asia/Kamchatka` - Asia/Kamchatka\n* `Asia/Karachi` - Asia/Karachi\n* `Asia/Kathmandu` - Asia/Kathmandu\n* `Asia/Khandyga` - Asia/Khandyga\n* `Asia/Kolkata` - Asia/Kolkata\n* `Asia/Krasnoyarsk` - Asia/Krasnoyarsk\n* `Asia/Kuala_Lumpur` - Asia/Kuala_Lumpur\n* `Asia/Kuching` - Asia/Kuching\n* `Asia/Kuwait` - Asia/Kuwait\n* `Asia/Macau` - Asia/Macau\n* `Asia/Magadan` - Asia/Magadan\n* `Asia/Makassar` - Asia/Makassar\n* `Asia/Manila` - Asia/Manila\n* `Asia/Muscat` - Asia/Muscat\n* `Asia/Nicosia` - Asia/Nicosia\n* `Asia/Novokuznetsk` - Asia/Novokuznetsk\n* `Asia/Novosibirsk` - Asia/Novosibirsk\n* `Asia/Omsk` - Asia/Omsk\n* `Asia/Oral` - Asia/Oral\n* `Asia/Phnom_Penh` - Asia/Phnom_Penh\n* `Asia/Pontianak` - Asia/Pontianak\n* `Asia/Pyongyang` - Asia/Pyongyang\n* `Asia/Qatar` - Asia/Qatar\n* `Asia/Qostanay` - Asia/Qostanay\n* `Asia/Qyzylorda` - Asia/Qyzylorda\n* `Asia/Riyadh` - Asia/Riyadh\n* `Asia/Sakhalin` - Asia/Sakhalin\n* `Asia/Samarkand` - Asia/Samarkand\n* `Asia/Seoul` - Asia/Seoul\n* `Asia/Shanghai` - Asia/Shanghai\n* `Asia/Singapore` - Asia/Singapore\n* `Asia/Srednekolymsk` - Asia/Srednekolymsk\n* `Asia/Taipei` - Asia/Taipei\n* `Asia/Tashkent` - Asia/Tashkent\n* `Asia/Tbilisi` - Asia/Tbilisi\n* `Asia/Tehran` - Asia/Tehran\n* `Asia/Thimphu` - Asia/Thimphu\n* `Asia/Tokyo` - Asia/Tokyo\n* `Asia/Tomsk` - Asia/Tomsk\n* `Asia/Ulaanbaatar` - Asia/Ulaanbaatar\n* `Asia/Urumqi` - Asia/Urumqi\n* `Asia/Ust-Nera` - Asia/Ust-Nera\n* `Asia/Vientiane` - Asia/Vientiane\n* `Asia/Vladivostok` - Asia/Vladivostok\n* `Asia/Yakutsk` - Asia/Yakutsk\n* `Asia/Yangon` - Asia/Yangon\n* `Asia/Yekaterinburg` - Asia/Yekaterinburg\n* `Asia/Yerevan` - Asia/Yerevan\n* `Atlantic/Azores` - Atlantic/Azores\n* `Atlantic/Bermuda` - Atlantic/Bermuda\n* `Atlantic/Canary` - Atlantic/Canary\n* `Atlantic/Cape_Verde` - Atlantic/Cape_Verde\n* `Atlantic/Faroe` - Atlantic/Faroe\n* `Atlantic/Madeira` - Atlantic/Madeira\n* `Atlantic/Reykjavik` - Atlantic/Reykjavik\n* `Atlantic/South_Georgia` - Atlantic/South_Georgia\n* `Atlantic/St_Helena` - Atlantic/St_Helena\n* `Atlantic/Stanley` - Atlantic/Stanley\n* `Australia/Adelaide` - Australia/Adelaide\n* `Australia/Brisbane` - Australia/Brisbane\n* `Australia/Broken_Hill` - Australia/Broken_Hill\n* `Australia/Darwin` - Australia/Darwin\n* `Australia/Eucla` - Australia/Eucla\n* `Australia/Hobart` - Australia/Hobart\n* `Australia/Lindeman` - Australia/Lindeman\n* `Australia/Lord_Howe` - Australia/Lord_Howe\n* `Australia/Melbourne` - Australia/Melbourne\n* `Australia/Perth` - Australia/Perth\n* `Australia/Sydney` - Australia/Sydney\n* `Canada/Atlantic` - Canada/Atlantic\n* `Canada/Central` - Canada/Central\n* `Canada/Eastern` - Canada/Eastern\n* `Canada/Mountain` - Canada/Mountain\n* `Canada/Newfoundland` - Canada/Newfoundland\n* `Canada/Pacific` - Canada/Pacific\n* `Europe/Amsterdam` - Europe/Amsterdam\n* `Europe/Andorra` - Europe/Andorra\n* `Europe/Astrakhan` - Europe/Astrakhan\n* `Europe/Athens` - Europe/Athens\n* `Europe/Belgrade` - Europe/Belgrade\n* `Europe/Berlin` - Europe/Berlin\n* `Europe/Bratislava` - Europe/Bratislava\n* `Europe/Brussels` - Europe/Brussels\n* `Europe/Bucharest` - Europe/Bucharest\n* `Europe/Budapest` - Europe/Budapest\n* `Europe/Busingen` - Europe/Busingen\n* `Europe/Chisinau` - Europe/Chisinau\n* `Europe/Copenhagen` - Europe/Copenhagen\n* `Europe/Dublin` - Europe/Dublin\n* `Europe/Gibraltar` - Europe/Gibraltar\n* `Europe/Guernsey` - Europe/Guernsey\n* `Europe/Helsinki` - Europe/Helsinki\n* `Europe/Isle_of_Man` - Europe/Isle_of_Man\n* `Europe/Istanbul` - Europe/Istanbul\n* `Europe/Jersey` - Europe/Jersey\n* `Europe/Kaliningrad` - Europe/Kaliningrad\n* `Europe/Kirov` - Europe/Kirov\n* `Europe/Kyiv` - Europe/Kyiv\n* `Europe/Lisbon` - Europe/Lisbon\n* `Europe/Ljubljana` - Europe/Ljubljana\n* `Europe/London` - Europe/London\n* `Europe/Luxembourg` - Europe/Luxembourg\n* `Europe/Madrid` - Europe/Madrid\n* `Europe/Malta` - Europe/Malta\n* `Europe/Mariehamn` - Europe/Mariehamn\n* `Europe/Minsk` - Europe/Minsk\n* `Europe/Monaco` - Europe/Monaco\n* `Europe/Moscow` - Europe/Moscow\n* `Europe/Oslo` - Europe/Oslo\n* `Europe/Paris` - Europe/Paris\n* `Europe/Podgorica` - Europe/Podgorica\n* `Europe/Prague` - Europe/Prague\n* `Europe/Riga` - Europe/Riga\n* `Europe/Rome` - Europe/Rome\n* `Europe/Samara` - Europe/Samara\n* `Europe/San_Marino` - Europe/San_Marino\n* `Europe/Sarajevo` - Europe/Sarajevo\n* `Europe/Saratov` - Europe/Saratov\n* `Europe/Simferopol` - Europe/Simferopol\n* `Europe/Skopje` - Europe/Skopje\n* `Europe/Sofia` - Europe/Sofia\n* `Europe/Stockholm` - Europe/Stockholm\n* `Europe/Tallinn` - Europe/Tallinn\n* `Europe/Tirane` - Europe/Tirane\n* `Europe/Ulyanovsk` - Europe/Ulyanovsk\n* `Europe/Vaduz` - Europe/Vaduz\n* `Europe/Vatican` - Europe/Vatican\n* `Europe/Vienna` - Europe/Vienna\n* `Europe/Vilnius` - Europe/Vilnius\n* `Europe/Volgograd` - Europe/Volgograd\n* `Europe/Warsaw` - Europe/Warsaw\n* `Europe/Zagreb` - Europe/Zagreb\n* `Europe/Zurich` - Europe/Zurich\n* `GMT` - GMT\n* `Indian/Antananarivo` - Indian/Antananarivo\n* `Indian/Chagos` - Indian/Chagos\n* `Indian/Christmas` - Indian/Christmas\n* `Indian/Cocos` - Indian/Cocos\n* `Indian/Comoro` - Indian/Comoro\n* `Indian/Kerguelen` - Indian/Kerguelen\n* `Indian/Mahe` - Indian/Mahe\n* `Indian/Maldives` - Indian/Maldives\n* `Indian/Mauritius` - Indian/Mauritius\n* `Indian/Mayotte` - Indian/Mayotte\n* `Indian/Reunion` - Indian/Reunion\n* `Pacific/Apia` - Pacific/Apia\n* `Pacific/Auckland` - Pacific/Auckland\n* `Pacific/Bougainville` - Pacific/Bougainville\n* `Pacific/Chatham` - Pacific/Chatham\n* `Pacific/Chuuk` - Pacific/Chuuk\n* `Pacific/Easter` - Pacific/Easter\n* `Pacific/Efate` - Pacific/Efate\n* `Pacific/Fakaofo` - Pacific/Fakaofo\n* `Pacific/Fiji` - Pacific/Fiji\n* `Pacific/Funafuti` - Pacific/Funafuti\n* `Pacific/Galapagos` - Pacific/Galapagos\n* `Pacific/Gambier` - Pacific/Gambier\n* `Pacific/Guadalcanal` - Pacific/Guadalcanal\n* `Pacific/Guam` - Pacific/Guam\n* `Pacific/Honolulu` - Pacific/Honolulu\n* `Pacific/Kanton` - Pacific/Kanton\n* `Pacific/Kiritimati` - Pacific/Kiritimati\n* `Pacific/Kosrae` - Pacific/Kosrae\n* `Pacific/Kwajalein` - Pacific/Kwajalein\n* `Pacific/Majuro` - Pacific/Majuro\n* `Pacific/Marquesas` - Pacific/Marquesas\n* `Pacific/Midway` - Pacific/Midway\n* `Pacific/Nauru` - Pacific/Nauru\n* `Pacific/Niue` - Pacific/Niue\n* `Pacific/Norfolk` - Pacific/Norfolk\n* `Pacific/Noumea` - Pacific/Noumea\n* `Pacific/Pago_Pago` - Pacific/Pago_Pago\n* `Pacific/Palau` - Pacific/Palau\n* `Pacific/Pitcairn` - Pacific/Pitcairn\n* `Pacific/Pohnpei` - Pacific/Pohnpei\n* `Pacific/Port_Moresby` - Pacific/Port_Moresby\n* `Pacific/Rarotonga` - Pacific/Rarotonga\n* `Pacific/Saipan` - Pacific/Saipan\n* `Pacific/Tahiti` - Pacific/Tahiti\n* `Pacific/Tarawa` - Pacific/Tarawa\n* `Pacific/Tongatapu` - Pacific/Tongatapu\n* `Pacific/Wake` - Pacific/Wake\n* `Pacific/Wallis` - Pacific/Wallis\n* `US/Alaska` - US/Alaska\n* `US/Arizona` - US/Arizona\n* `US/Central` - US/Central\n* `US/Eastern` - US/Eastern\n* `US/Hawaii` - US/Hawaii\n* `US/Mountain` - US/Mountain\n* `US/Pacific` - US/Pacific\n* `UTC` - UTC")
	Enum("Africa/Abidjan", "Africa/Accra", "Africa/Addis_Ababa", "Africa/Algiers", "Africa/Asmara", "Africa/Bamako", "Africa/Bangui", "Africa/Banjul", "Africa/Bissau", "Africa/Blantyre", "Africa/Brazzaville", "Africa/Bujumbura", "Africa/Cairo", "Africa/Casablanca", "Africa/Ceuta", "Africa/Conakry", "Africa/Dakar", "Africa/Dar_es_Salaam", "Africa/Djibouti", "Africa/Douala", "Africa/El_Aaiun", "Africa/Freetown", "Africa/Gaborone", "Africa/Harare", "Africa/Johannesburg", "Africa/Juba", "Africa/Kampala", "Africa/Khartoum", "Africa/Kigali", "Africa/Kinshasa", "Africa/Lagos", "Africa/Libreville", "Africa/Lome", "Africa/Luanda", "Africa/Lubumbashi", "Africa/Lusaka", "Africa/Malabo", "Africa/Maputo", "Africa/Maseru", "Africa/Mbabane", "Africa/Mogadishu", "Africa/Monrovia", "Africa/Nairobi", "Africa/Ndjamena", "Africa/Niamey", "Africa/Nouakchott", "Africa/Ouagadougou", "Africa/Porto-Novo", "Africa/Sao_Tome", "Africa/Tripoli", "Africa/Tunis", "Africa/Windhoek", "America/Adak", "America/Anchorage", "America/Anguilla", "America/Antigua", "America/Araguaina", "America/Argentina/Buenos_Aires", "America/Argentina/Catamarca", "America/Argentina/Cordoba", "America/Argentina/Jujuy", "America/Argentina/La_Rioja", "America/Argentina/Mendoza", "America/Argentina/Rio_Gallegos", "America/Argentina/Salta", "America/Argentina/San_Juan", "America/Argentina/San_Luis", "America/Argentina/Tucuman", "America/Argentina/Ushuaia", "America/Aruba", "America/Asuncion", "America/Atikokan", "America/Bahia", "America/Bahia_Banderas", "America/Barbados", "America/Belem", "America/Belize", "America/Blanc-Sablon", "America/Boa_Vista", "America/Bogota", "America/Boise", "America/Cambridge_Bay", "America/Campo_Grande", "America/Cancun", "America/Caracas", "America/Cayenne", "America/Cayman", "America/Chicago", "America/Chihuahua", "America/Ciudad_Juarez", "America/Costa_Rica", "America/Creston", "America/Cuiaba", "America/Curacao", "America/Danmarkshavn", "America/Dawson", "America/Dawson_Creek", "America/Denver", "America/Detroit", "America/Dominica", "America/Edmonton", "America/Eirunepe", "America/El_Salvador", "America/Fort_Nelson", "America/Fortaleza", "America/Glace_Bay", "America/Goose_Bay", "America/Grand_Turk", "America/Grenada", "America/Guadeloupe", "America/Guatemala", "America/Guayaquil", "America/Guyana", "America/Halifax", "America/Havana", "America/Hermosillo", "America/Indiana/Indianapolis", "America/Indiana/Knox", "America/Indiana/Marengo", "America/Indiana/Petersburg", "America/Indiana/Tell_City", "America/Indiana/Vevay", "America/Indiana/Vincennes", "America/Indiana/Winamac", "America/Inuvik", "America/Iqaluit", "America/Jamaica", "America/Juneau", "America/Kentucky/Louisville", "America/Kentucky/Monticello", "America/Kralendijk", "America/La_Paz", "America/Lima", "America/Los_Angeles", "America/Lower_Princes", "America/Maceio", "America/Managua", "America/Manaus", "America/Marigot", "America/Martinique", "America/Matamoros", "America/Mazatlan", "America/Menominee", "America/Merida", "America/Metlakatla", "America/Mexico_City", "America/Miquelon", "America/Moncton", "America/Monterrey", "America/Montevideo", "America/Montserrat", "America/Nassau", "America/New_York", "America/Nome", "America/Noronha", "America/North_Dakota/Beulah", "America/North_Dakota/Center", "America/North_Dakota/New_Salem", "America/Nuuk", "America/Ojinaga", "America/Panama", "America/Paramaribo", "America/Phoenix", "America/Port-au-Prince", "America/Port_of_Spain", "America/Porto_Velho", "America/Puerto_Rico", "America/Punta_Arenas", "America/Rankin_Inlet", "America/Recife", "America/Regina", "America/Resolute", "America/Rio_Branco", "America/Santarem", "America/Santiago", "America/Santo_Domingo", "America/Sao_Paulo", "America/Scoresbysund", "America/Sitka", "America/St_Barthelemy", "America/St_Johns", "America/St_Kitts", "America/St_Lucia", "America/St_Thomas", "America/St_Vincent", "America/Swift_Current", "America/Tegucigalpa", "America/Thule", "America/Tijuana", "America/Toronto", "America/Tortola", "America/Vancouver", "America/Whitehorse", "America/Winnipeg", "America/Yakutat", "Antarctica/Casey", "Antarctica/Davis", "Antarctica/DumontDUrville", "Antarctica/Macquarie", "Antarctica/Mawson", "Antarctica/McMurdo", "Antarctica/Palmer", "Antarctica/Rothera", "Antarctica/Syowa", "Antarctica/Troll", "Antarctica/Vostok", "Arctic/Longyearbyen", "Asia/Aden", "Asia/Almaty", "Asia/Amman", "Asia/Anadyr", "Asia/Aqtau", "Asia/Aqtobe", "Asia/Ashgabat", "Asia/Atyrau", "Asia/Baghdad", "Asia/Bahrain", "Asia/Baku", "Asia/Bangkok", "Asia/Barnaul", "Asia/Beirut", "Asia/Bishkek", "Asia/Brunei", "Asia/Chita", "Asia/Choibalsan", "Asia/Colombo", "Asia/Damascus", "Asia/Dhaka", "Asia/Dili", "Asia/Dubai", "Asia/Dushanbe", "Asia/Famagusta", "Asia/Gaza", "Asia/Hebron", "Asia/Ho_Chi_Minh", "Asia/Hong_Kong", "Asia/Hovd", "Asia/Irkutsk", "Asia/Jakarta", "Asia/Jayapura", "Asia/Jerusalem", "Asia/Kabul", "Asia/Kamchatka", "Asia/Karachi", "Asia/Kathmandu", "Asia/Khandyga", "Asia/Kolkata", "Asia/Krasnoyarsk", "Asia/Kuala_Lumpur", "Asia/Kuching", "Asia/Kuwait", "Asia/Macau", "Asia/Magadan", "Asia/Makassar", "Asia/Manila", "Asia/Muscat", "Asia/Nicosia", "Asia/Novokuznetsk", "Asia/Novosibirsk", "Asia/Omsk", "Asia/Oral", "Asia/Phnom_Penh", "Asia/Pontianak", "Asia/Pyongyang", "Asia/Qatar", "Asia/Qostanay", "Asia/Qyzylorda", "Asia/Riyadh", "Asia/Sakhalin", "Asia/Samarkand", "Asia/Seoul", "Asia/Shanghai", "Asia/Singapore", "Asia/Srednekolymsk", "Asia/Taipei", "Asia/Tashkent", "Asia/Tbilisi", "Asia/Tehran", "Asia/Thimphu", "Asia/Tokyo", "Asia/Tomsk", "Asia/Ulaanbaatar", "Asia/Urumqi", "Asia/Ust-Nera", "Asia/Vientiane", "Asia/Vladivostok", "Asia/Yakutsk", "Asia/Yangon", "Asia/Yekaterinburg", "Asia/Yerevan", "Atlantic/Azores", "Atlantic/Bermuda", "Atlantic/Canary", "Atlantic/Cape_Verde", "Atlantic/Faroe", "Atlantic/Madeira", "Atlantic/Reykjavik", "Atlantic/South_Georgia", "Atlantic/St_Helena", "Atlantic/Stanley", "Australia/Adelaide", "Australia/Brisbane", "Australia/Broken_Hill", "Australia/Darwin", "Australia/Eucla", "Australia/Hobart", "Australia/Lindeman", "Australia/Lord_Howe", "Australia/Melbourne", "Australia/Perth", "Australia/Sydney", "Canada/Atlantic", "Canada/Central", "Canada/Eastern", "Canada/Mountain", "Canada/Newfoundland", "Canada/Pacific", "Europe/Amsterdam", "Europe/Andorra", "Europe/Astrakhan", "Europe/Athens", "Europe/Belgrade", "Europe/Berlin", "Europe/Bratislava", "Europe/Brussels", "Europe/Bucharest", "Europe/Budapest", "Europe/Busingen", "Europe/Chisinau", "Europe/Copenhagen", "Europe/Dublin", "Europe/Gibraltar", "Europe/Guernsey", "Europe/Helsinki", "Europe/Isle_of_Man", "Europe/Istanbul", "Europe/Jersey", "Europe/Kaliningrad", "Europe/Kirov", "Europe/Kyiv", "Europe/Lisbon", "Europe/Ljubljana", "Europe/London", "Europe/Luxembourg", "Europe/Madrid", "Europe/Malta", "Europe/Mariehamn", "Europe/Minsk", "Europe/Monaco", "Europe/Moscow", "Europe/Oslo", "Europe/Paris", "Europe/Podgorica", "Europe/Prague", "Europe/Riga", "Europe/Rome", "Europe/Samara", "Europe/San_Marino", "Europe/Sarajevo", "Europe/Saratov", "Europe/Simferopol", "Europe/Skopje", "Europe/Sofia", "Europe/Stockholm", "Europe/Tallinn", "Europe/Tirane", "Europe/Ulyanovsk", "Europe/Vaduz", "Europe/Vatican", "Europe/Vienna", "Europe/Vilnius", "Europe/Volgograd", "Europe/Warsaw", "Europe/Zagreb", "Europe/Zurich", "GMT", "Indian/Antananarivo", "Indian/Chagos", "Indian/Christmas", "Indian/Cocos", "Indian/Comoro", "Indian/Kerguelen", "Indian/Mahe", "Indian/Maldives", "Indian/Mauritius", "Indian/Mayotte", "Indian/Reunion", "Pacific/Apia", "Pacific/Auckland", "Pacific/Bougainville", "Pacific/Chatham", "Pacific/Chuuk", "Pacific/Easter", "Pacific/Efate", "Pacific/Fakaofo", "Pacific/Fiji", "Pacific/Funafuti", "Pacific/Galapagos", "Pacific/Gambier", "Pacific/Guadalcanal", "Pacific/Guam", "Pacific/Honolulu", "Pacific/Kanton", "Pacific/Kiritimati", "Pacific/Kosrae", "Pacific/Kwajalein", "Pacific/Majuro", "Pacific/Marquesas", "Pacific/Midway", "Pacific/Nauru", "Pacific/Niue", "Pacific/Norfolk", "Pacific/Noumea", "Pacific/Pago_Pago", "Pacific/Palau", "Pacific/Pitcairn", "Pacific/Pohnpei", "Pacific/Port_Moresby", "Pacific/Rarotonga", "Pacific/Saipan", "Pacific/Tahiti", "Pacific/Tarawa", "Pacific/Tongatapu", "Pacific/Wake", "Pacific/Wallis", "US/Alaska", "US/Arizona", "US/Central", "US/Eastern", "US/Hawaii", "US/Mountain", "US/Pacific", "UTC")
})

var _ = API("Cycle Lifecycle API", func() {
	Title("Plane Internal Django Migration API")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:Assets")
	Meta("openapi:tag:Assets:desc", "**File Upload & Presigned URLs**\n\nGenerate presigned URLs for direct file uploads to cloud storage. Handle user avatars, cover images, and generic project assets with secure upload workflows.\n\n*Key Features:*\n- Generate presigned URLs for S3 uploads\n- Support for user avatars and cover images\n- Generic asset upload for projects\n- File validation and size limits\n\n*Use Cases:* User profile images, project file uploads, secure direct-to-cloud uploads.")
	Meta("openapi:tag:Cycles")
	Meta("openapi:tag:Cycles:desc", "**Sprint & Development Cycles**\n\nCreate and manage development cycles (sprints) to organize work into time-boxed iterations. Track progress, assign work items, and monitor team velocity.\n\n*Key Features:*\n- Create and configure development cycles\n- Assign work items to cycles\n- Track cycle progress and completion\n- Generate cycle analytics and reports\n\n*Use Cases:* Sprint planning, iterative development, progress tracking, team velocity.")
	Meta("openapi:tag:Intake")
	Meta("openapi:tag:Intake:desc", "**Work Item Intake Queue**\n\nManage incoming work items through a dedicated intake queue for triage and review. Submit, update, and process work items before they enter the main project workflow.\n\n*Key Features:*\n- Submit work items to intake queue\n- Review and triage incoming work items\n- Update intake work item status and properties\n- Accept, reject, or modify work items before approval\n\n*Use Cases:* Work item triage, external submissions, quality review, approval workflows.")
	Meta("openapi:tag:Labels")
	Meta("openapi:tag:Labels:desc", "**Labels & Tags**\n\nCreate and manage labels to categorize and organize work items. Use color-coded labels for easy identification, filtering, and project organization.\n\n*Key Features:*\n- Create custom labels with colors and descriptions\n- Apply labels to work items for categorization\n- Filter and search by labels\n- Organize labels across projects\n\n*Use Cases:* Priority marking, feature categorization, bug classification, team organization.")
	Meta("openapi:tag:Members")
	Meta("openapi:tag:Members:desc", "**Team Member Management**\n\nManage team members, roles, and permissions within projects and workspaces. Control access levels and track member participation.\n\n*Key Features:*\n- Invite and manage team members\n- Assign roles and permissions\n- Control project and workspace access\n- Track member activity and participation\n\n*Use Cases:* Team setup, access control, role management, collaboration.")
	Meta("openapi:tag:Modules")
	Meta("openapi:tag:Modules:desc", "**Feature Modules**\n\nGroup related work items into modules for better organization and tracking. Plan features, track progress, and manage deliverables at a higher level.\n\n*Key Features:*\n- Create and organize feature modules\n- Group work items by module\n- Track module progress and completion\n- Manage module leads and assignments\n\n*Use Cases:* Feature planning, release organization, progress tracking, team coordination.")
	Meta("openapi:tag:Projects")
	Meta("openapi:tag:Projects:desc", "**Project Management**\n\nCreate and manage projects to organize your development work. Configure project settings, manage team access, and control project visibility.\n\n*Key Features:*\n- Create, update, and delete projects\n- Configure project settings and preferences\n- Manage team access and permissions\n- Control project visibility and sharing\n\n*Use Cases:* Project setup, team collaboration, access control, project configuration.")
	Meta("openapi:tag:States")
	Meta("openapi:tag:States:desc", "**Workflow States**\n\nDefine custom workflow states for work items to match your team's process. Configure state transitions and track work item progress through different stages.\n\n*Key Features:*\n- Create custom workflow states\n- Configure state transitions and rules\n- Track work item progress through states\n- Set state-based permissions and automation\n\n*Use Cases:* Custom workflows, status tracking, process automation, progress monitoring.")
	Meta("openapi:tag:Users")
	Meta("openapi:tag:Users:desc", "**Current User Information**\n\nGet information about the currently authenticated user including profile details and account settings.\n\n*Key Features:*\n- Retrieve current user profile\n- Access user account information\n- View user preferences and settings\n- Get authentication context\n\n*Use Cases:* Profile display, user context, account information, authentication status.")
	Meta("openapi:tag:Work Item Activity")
	Meta("openapi:tag:Work Item Activity:desc", "**Activity History & Search**\n\nView activity history and search for work items across the workspace. Get detailed activity logs and find work items using text search.\n\n*Key Features:*\n- View work item activity history\n- Search work items across workspace\n- Track changes and modifications\n- Filter search results by project\n\n*Use Cases:* Activity tracking, work item discovery, change history, workspace search.")
	Meta("openapi:tag:Work Item Attachments")
	Meta("openapi:tag:Work Item Attachments:desc", "**Work Item File Attachments**\n\nGenerate presigned URLs for uploading files directly to specific work items. Upload and manage attachments associated with work items.\n\n*Key Features:*\n- Generate presigned URLs for work item attachments\n- Upload files directly to work items\n- Retrieve and manage attachment metadata\n- Delete attachments from work items\n\n*Use Cases:* Screenshots, error logs, design files, supporting documents.")
	Meta("openapi:tag:Work Item Comments")
	Meta("openapi:tag:Work Item Comments:desc", "**Comments & Discussions**\n\nAdd comments and discussions to work items for team collaboration. Support threaded conversations, mentions, and rich text formatting.\n\n*Key Features:*\n- Add comments to work items\n- Thread conversations and replies\n- Mention users and trigger notifications\n- Rich text and markdown support\n\n*Use Cases:* Team discussions, progress updates, code reviews, decision tracking.")
	Meta("openapi:tag:Work Item Links")
	Meta("openapi:tag:Work Item Links:desc", "**External Links & References**\n\nLink work items to external resources like documentation, repositories, or design files. Maintain connections between work items and external systems.\n\n*Key Features:*\n- Add external URL links to work items\n- Validate and preview linked resources\n- Organize links by type and category\n- Track link usage and access\n\n*Use Cases:* Documentation links, repository connections, design references, external tools.")
	Meta("openapi:tag:Work Items")
	Meta("openapi:tag:Work Items:desc", "**Work Items & Tasks**\n\nCreate and manage work items like tasks, bugs, features, and user stories. The core entities for tracking work in your projects.\n\n*Key Features:*\n- Create, update, and manage work items\n- Assign to team members and set priorities\n- Track progress through workflow states\n- Set due dates, estimates, and relationships\n\n*Use Cases:* Bug tracking, task management, feature development, sprint planning.")
	Meta("openapi:tag:projects")
	Meta("openapi:tag:cycles")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var ImportedAPIKeyAuthSecurity = APIKeySecurity("ApiKeyAuth", func() {
	Description("Plane API token authentication.")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var cycleLifecycleError = Type("CycleLifecycleError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("kind", "body")
})

func cycleLifecycleErrors() {
	for _, item := range []struct {
		name string
		code int
	}{
		{"status_400", 400}, {"status_401", 401}, {"status_403", 403},
		{"status_404", 404}, {"status_405", 405}, {"status_409", 409},
		{"status_415", 415}, {"status_429", 429}, {"status_500", 500},
	} {
		Error(item.name, cycleLifecycleError, func() {
			if item.code == 429 {
				Temporary()
			}
			if item.code == 500 {
				Fault()
			}
		})
		HTTP(func() {
			Response(item.name, item.code, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body("body")
			})
		})
	}
}

var _ = Service("CycleLifecycleAPI", func() {
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCyclesByCycleIDUnarchive", func() {
		cycleLifecycleErrors()
		Description("Restore an archived cycle to active status, making it available for regular use.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_archived_cycles_by_cycle_id_unarchive")
		Meta("openapi:summary", "Unarchive cycle")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/archived-cycles/{cycle_id}/unarchive/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.cycle.CycleArchiveUnarchiveAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("cycle_id", "project_id", "slug")
		})
		Result(Empty)

		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:projects")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/archived-cycles/{cycle_id}/unarchive/")
			Param("cycle_id")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("Resource unarchived successfully")
			})

		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDCycles", func() {
		cycleLifecycleErrors()
		Description("Create a new development cycle with specified name, description, and date range. Supports external ID tracking for integration purposes.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_cycles")
		Meta("openapi:summary", "Create cycle")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/cycles/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.cycle.CycleListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("project_id", "slug")
		})
		Result(ImportedPlaneAPISerializersCycleCycleserializerResponse)

		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:cycles")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersCycleCyclecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				Description("Cycle created")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersCycleCycleserializerResponse)
			})

		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchive", func() {
		cycleLifecycleErrors()
		Description("Move a completed cycle to archived status for historical tracking. Only cycles that have ended can be archived.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_archive")
		Meta("openapi:summary", "Archive cycle")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.cycle.CycleArchiveUnarchiveAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("cycle_id", "project_id", "slug")
		})
		Result(Empty)

		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:cycles")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/")
			Param("cycle_id")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("Resource archived successfully")
			})

		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPk", func() {
		cycleLifecycleErrors()
		Description("Permanently remove a cycle and all its associated issue relationships")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_pk")
		Meta("openapi:summary", "Delete cycle")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/cycles/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.cycle.CycleDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("pk", "project_id", "slug")
		})
		Result(Empty)

		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:cycles")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("Resource deleted successfully")
			})

		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPk", func() {
		cycleLifecycleErrors()
		Description("Modify an existing cycle's properties like name, description, or date range. Completed cycles can only have their sort order changed.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_pk")
		Meta("openapi:summary", "Update cycle")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/cycles/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.cycle.CycleDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("pk", "project_id", "slug")
		})
		Result(ImportedPlaneAPISerializersCycleCycleserializerResponse)

		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:cycles")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersCycleCycleupdateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				Description("Cycle updated")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersCycleCycleserializerResponse)
			})

		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedCycles", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_archived_cycles")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/archived-cycles/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.archive.CycleArchiveUnarchiveEndpoint\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:projects")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDArchivedCycles", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_archived_cycles")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/archived-cycles/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.archive.CycleArchiveUnarchiveEndpoint\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersArchiveresponseResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:projects")
			POST("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersArchiveresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedCyclesByPk", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_archived_cycles_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.archive.CycleArchiveUnarchiveEndpoint\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:projects")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDArchivedCyclesByPk", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_archived_cycles_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.archive.CycleArchiveUnarchiveEndpoint\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersArchiveresponseResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:projects")
			POST("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersArchiveresponseResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDCycles", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_cycles")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/cycles/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.base.CycleViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersCycleCycleserializerResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:cycles")
			POST("/api/workspaces/{slug}/projects/{project_id}/cycles/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersCycleCyclewriteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersCycleCycleserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDCyclesDateCheck", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_cycles_date_check")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/cycles/date-check/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.base.CycleDateCheckEndpoint\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersCycledatecheckresponseResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:cycles")
			POST("/api/workspaces/{slug}/projects/{project_id}/cycles/date-check/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersCycledatecheckrequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersCycledatecheckresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchive", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.archive.CycleArchiveUnarchiveEndpoint\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("cycle_id", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:cycles")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/")
			Param("cycle_id")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchive", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.archive.CycleArchiveUnarchiveEndpoint\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("cycle_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("cycle_id", "project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersArchiveresponseResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:cycles")
			POST("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/")
			Param("cycle_id")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersArchiveresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByPk", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_cycles_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.base.CycleViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:cycles")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDCyclesByPk", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_cycles_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.base.CycleViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersCycleCycleserializerResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:cycles")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersCycleCyclewriteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersCycleCycleserializerResponse)
			})
		})
	})
	Method("PutAPIWorkspacesBySlugProjectsByProjectIDCyclesByPk", func() {
		cycleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_cycles_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.cycle.base.CycleViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(CycleAppReadonlyPutResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:cycles")
			PUT("/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersCycleCyclewriteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(CycleAppReadonlyPutResponse)
			})
		})
	})
})
