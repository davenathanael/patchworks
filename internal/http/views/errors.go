package views

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FormErrors maps a field name to its error message; the key "form" holds a
// top-level (non-field) error rendered as an alert.
type FormErrors map[string]string

// FieldError renders a field's inline error, paired with aria-invalid /
// aria-describedby on the associated input.
func FieldError(field string, errs FormErrors) Node {
	return If(errs[field] != "", P(Class("form-error"), ID(field+"-error"), Text(errs[field])))
}

// Toast renders an alert-style toast for the #toast-container; kind is
// "error" or "success". errorID links the toast to a logged error ref.
func Toast(kind, message, errorID string) Node {
	return Div(
		Class("toast toast-"+kind),
		Role("alert"),
		Text(message),
		If(errorID != "", Code(Text("Ref "+errorID))),
		Button(
			Type("button"),
			Class("toast-dismiss"),
			Text("\u00d7"),
			Attr("hx-on:click", "this.closest('.toast').remove()"),
		),
	)
}

// ErrorPage renders a standalone error page; unauthenticated, so no AppShell.
func ErrorPage(message, errorID string) Node {
	return Page("Error — Patchworks",
		Main(
			Class("error-page"),
			H1(Text("Something went wrong")),
			P(Text(message)),
			P(Code(Text("Ref "+errorID))),
			A(Class("button"), Href("/"), Text("Back to dashboard")),
		),
	)
}
