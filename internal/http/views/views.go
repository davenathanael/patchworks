package views

import (
	"github.com/davenathanael/patchwork/internal/core"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

//  This file contains all common functionalities for HTML-based views.

// Page is the main HTML page template.
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

// AppShell wraps the authenticated app layout with sidebar and main content.
func AppShell(user core.User, sidebar, mainContent Node) Node {
	return Div(
		Attr("data-sidebar-layout"),
		Nav(
			Attr("data-topnav"),
			TopNav(user),
		),
		Aside(Attr("data-sidebar"), sidebar),
		mainContent,
	)
}

func MobileSidebarToggleButton() Node {
	return Button(Attr("data-sidebar-toggle"), Aria("label", "Toggle menu"), Class("outline"), Text("menu"))
}

// TopNav renders the top navigation bar with app name and user controls.
func TopNav(user core.User) Node {
	return Div(
		Class("row"),
		Div(
			Class("col-4 items-center gap-2"),
			MobileSidebarToggleButton(),
			A(Href("/"), Text("Patchworks")),
		),
		Div(
			Class("col-8 justify-end hstack gap-2"),
			Span(Class("text-muted"), Text(user.Email)),
			A(Href("/auth/logout"), Class("button outline small"), Text("Logout")),
		),
	)
}

// SidebarNav renders collections and tags in the sidebar.
func SidebarNav(collections []CollectionItem, tags []TagItem) Node {
	return Nav(
		CollectionFilter(collections, ""),
		TagFilter(tags, ""),
	)
}
