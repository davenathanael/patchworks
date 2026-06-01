package views

import (
	"github.com/davenathanael/patchwork/internal/core"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func Page(title string, children ...Node) Node {
	return HTML5(HTML5Props{
		Title:    title,
		Language: "en",
		Head: []Node{
			Link(Rel("stylesheet"), Href("/static/css/app.css")),
			Script(Src("/static/js/oat.min.js"), Defer()),
		},
		Body: children,
	})
}

func AppShell(user core.User, mainContent Node) Node {
	return Div(
		Attr("data-sidebar-layout"),
		Nav(Attr("data-topnav"), TopNav(user)),
		mainContent,
	)
}

func TopNav(user core.User) Node {
	return Div(
		Class("row"),
		Div(
			Class("col-4"),
			Div(
				Class("nav-links"),
				A(Class("nav-link"), Href("/"), Text("Patchworks")),
				A(Class("nav-link"), Href("/collections"), Text("Collections")),
			),
		),
		Div(
			Class("col-8"),
			Div(
				Class("nav-user"),
				Span(Class("text-muted"), Text(user.Email)),
				A(Href("/auth/logout"), Class("button outline small"), Text("Logout")),
			),
		),
	)
}
