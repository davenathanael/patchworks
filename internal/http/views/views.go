package views

import (
	"net/http"

	"github.com/davenathanael/patchwork/internal/core"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

// IsHtmx reports whether the request expects a partial HTML response.
func IsHtmx(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func Page(title string, children ...Node) Node {
	return HTML5(HTML5Props{
		Title:    title,
		Language: "en",
		Head: []Node{
			Link(Rel("stylesheet"), Href("https://unpkg.com/open-props")),
			Link(Rel("stylesheet"), Href("/static/css/app.css")),
			Script(Src("https://unpkg.com/htmx.org@2.0.10/dist/htmx.min.js"), Defer()),
			Script(Src("/static/js/app.js"), Defer()),
		},
		Body: children,
	})
}

func AppShell(user core.User, mainContent Node) Node {
	return Div(
		Attr("data-sidebar-layout"),
		Nav(Class("sidenav"), SideNav(user)),
		Nav(Attr("data-topnav"), TopNav(user)),
		mainContent,
		Div(ID("toast-container")),
	)
}

func SideNav(user core.User) Node {
	return Group{
		A(Class("brand"), Href("/"), Group{Text("Patch"), B(Text("works"))}),
		A(Class("nav-link"), Href("/"), Text("Home")),
		A(Class("nav-link"), Href("/collections"), Text("Collections")),
		A(Class("nav-link"), Href("/archived"), Text("Archived")),
		Div(
			Class("sid-user"),
			Span(Class("muted"), Text(user.Email)),
			A(Href("/auth/logout"), Class("button outline small"), Text("Logout")),
		),
	}
}

func TopNav(user core.User) Node {
	return Div(
		Class("topnav-inner"),
		Div(
			Class("nav-links"),
			A(Class("nav-link"), Href("/"), Text("Patchworks")),
			A(Class("nav-link"), Href("/collections"), Text("Collections")),
			A(Class("nav-link"), Href("/archived"), Text("Archived")),
		),
		Div(
			Class("nav-user"),
			Span(Class("muted"), Text(user.Email)),
			A(Href("/auth/logout"), Class("button outline"), Text("Logout")),
		),
	)
}
