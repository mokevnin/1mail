package problems

import (
	"net/http"
	"strings"

	collectapi "github.com/mokevnin/1mail/gen/collect"
	externalapi "github.com/mokevnin/1mail/gen/external"
	siteapi "github.com/mokevnin/1mail/gen/site"
)

type FieldErrors map[string][]string

type Problem struct {
	Status int32
	Title  string
	Detail string
	Errors FieldErrors
}

func BadRequest(detail string) Problem {
	return New(http.StatusBadRequest, detail)
}

func Unauthorized(detail string) Problem {
	return New(http.StatusUnauthorized, detail)
}

func Forbidden(detail string) Problem {
	return New(http.StatusForbidden, detail)
}

func NotFound(detail string) Problem {
	return New(http.StatusNotFound, detail)
}

func Conflict(detail string) Problem {
	return New(http.StatusConflict, detail)
}

func ConflictWithErrors(detail string, errors FieldErrors) Problem {
	return Conflict(detail).WithErrors(errors)
}

func Unprocessable(detail string) Problem {
	return New(http.StatusUnprocessableEntity, detail)
}

func UnprocessableWithErrors(detail string, errors FieldErrors) Problem {
	return Unprocessable(detail).WithErrors(errors)
}

func NotImplemented(detail string) Problem {
	return New(http.StatusNotImplemented, detail)
}

func New(status int, detail string) Problem {
	return Problem{
		Status: int32(status),
		Title:  http.StatusText(status),
		Detail: detail,
	}
}

func (p Problem) WithErrors(errors FieldErrors) Problem {
	p.Errors = errors
	return p
}

func (p Problem) Site() siteapi.ProblemDetails {
	detail := p.Detail
	title := p.Title
	status := p.Status
	res := siteapi.ProblemDetails{
		Status: &status,
		Title:  &title,
		Detail: &detail,
	}
	if p.Errors != nil {
		errors := map[string][]string(p.Errors)
		fields := toFormFields(p.Errors)
		res.Errors = &errors
		res.Form = &detail
		res.Fields = &fields
	}
	return res
}

func (p Problem) External() externalapi.ProblemDetails {
	detail := p.Detail
	title := p.Title
	status := p.Status
	res := externalapi.ProblemDetails{
		Status: &status,
		Title:  &title,
		Detail: &detail,
	}
	if p.Errors != nil {
		errors := map[string][]string(p.Errors)
		fields := toFormFields(p.Errors)
		res.Errors = &errors
		res.Form = &detail
		res.Fields = &fields
	}
	return res
}

func (p Problem) Collect() collectapi.ProblemDetails {
	detail := p.Detail
	title := p.Title
	status := p.Status
	res := collectapi.ProblemDetails{
		Status: &status,
		Title:  &title,
		Detail: &detail,
	}
	if p.Errors != nil {
		errors := map[string][]string(p.Errors)
		fields := toFormFields(p.Errors)
		res.Errors = &errors
		res.Form = &detail
		res.Fields = &fields
	}
	return res
}

func toFormFields(errors FieldErrors) map[string]string {
	fields := make(map[string]string, len(errors))
	for field, messages := range errors {
		fields[field] = strings.Join(messages, ", ")
	}
	return fields
}
