package views

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

// Page is the main HTML page template.
func Page(title string, children ...Node) Node {
	return HTML5(HTML5Props{
		Title:    title,
		Language: "en",
		Head: []Node{
			Link(Rel("stylesheet"), Href("https://unpkg.com/@knadh/oat/oat.min.css")),
			Script(Src("https://unpkg.com/@knadh/oat/oat.min.js"), Defer()),
		},
		Body: children,
	})
}

// HomePage is the home page of the application.
func HomePage() Node {
	return Page("Patchworks", Main(H1(Text("Hello Patchworks!"))))
}
