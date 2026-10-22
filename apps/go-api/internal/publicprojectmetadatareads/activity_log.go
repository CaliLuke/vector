package publicprojectmetadatareads

import "context"

type APIActivityLog struct {
	TokenIdentifier string
	Path            string
	Method          string
	QueryParams     any
	Headers         any
	Body            any
	ResponseCode    int
	ResponseBody    any
	IPAddress       any
	UserAgent       any
}

type ActivityLogger interface {
	Log(context.Context, APIActivityLog) error
}
