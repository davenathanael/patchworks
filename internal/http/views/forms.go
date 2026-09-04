package views

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// TextInput renders a labelled input preserving its value and field error,
// wired up for aria-invalid / aria-describedby. Extra nodes (Required,
// autocomplete, ...) are appended to the input.
func TextInput(label, name, typ, value string, errs FormErrors, extra ...Node) Node {
	return Label(Text(label),
		Input(append([]Node{
			Type(typ), Name(name), Value(value),
			Attr("aria-describedby", name+"-error"),
			If(errs[name] != "", Attr("aria-invalid", "true")),
		}, extra...)...),
		FieldError(name, errs),
	)
}
