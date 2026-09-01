package views

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/google/uuid"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// CollectionForm is the shared create/edit collection form view-model: the
// ajg/form decode target and the render model. Zero value renders a fresh form.
type CollectionForm struct {
	Name        string     `form:"name"`
	Description string     `form:"description"`
	Errors      FormErrors `form:"-"`
}

func ListCollectionsPage(collections []core.Collection, user core.User) Node {
	collectionItem := func(c core.Collection) Node {
		memberCount := len(c.Members)
		meta := fmt.Sprintf("%d bookmarks · %d member", c.BookmarkCount, memberCount)
		if memberCount != 1 {
			meta += "s"
		}

		return Li(
			A(
				Class("collection-card"),
				Href(fmt.Sprintf("/collections/%s", c.ID.String())),
				H3(Text(c.Name)),
				If(c.Description != "", P(Text(c.Description))),
				Small(Text(meta)),
			),
		)
	}

	content := Main(
		Header(
			H1(Text("Your Collections")),
			A(Href("/collections/new"), Class("button"), Text("＋ New")),
		),
		Ul(Class("collection-list"), Map(collections, collectionItem)),
	)
	return Page("Collections - Patchworks", AppShell(user, content))
}

func Avatar(email string) Node {
	initials := initialsFromEmail(email)
	return Div(
		Class("avatar"),
		Aria("label", email),
		Abbr(Title(email), Text(initials)),
	)
}

func initialsFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 0 || parts[0] == "" {
		return "??"
	}
	name := parts[0]
	if len(name) >= 2 {
		return strings.ToUpper(name[:2])
	}
	return strings.ToUpper(name)
}

// CreateCollectionsPage renders the create-collection form, preserving
// submitted values and field errors.
func CreateCollectionsPage(user core.User, f CollectionForm) Node {
	content := Main(
		backToCollectionsLink(),
		H1(Text("Create a Collection")),
		Form(Method("post"), Action("/collections"),
			If(f.Errors["form"] != "", Toast("error", f.Errors["form"], "")),
			TextInput("Name", "name", "text", f.Name, f.Errors, Placeholder("e.g. Work"), Required()),
			Label(Text("Description"),
				Textarea(Name("description"), Placeholder("What lives here?"), Text(f.Description)),
				FieldError("description", f.Errors),
			),
			Button(Type("submit"), Text("Create")),
		),
	)
	return Page("Create New Collection - Patchworks", AppShell(user, content))
}

func CollectionPage(collection core.Collection, bookmarks []core.Bookmark, user core.User) Node {
	memberSection := Section(
		H2(Text("Members")),
		Ul(Class("member-list"), Map(collection.Members, func(m core.CollectionMember) Node {
			return Li(
				Article(
					Header(
						Avatar(m.User.Email),
						Address(Text(m.User.Email)),
						Small(Text(m.Role)),
						Time(Attr("datetime", m.AddedAt.Format(time.RFC3339)), Text(relativeTime(m.AddedAt))),
					),
					If(m.User.ID != user.ID,
						Form(Method("post"), Action(fmt.Sprintf("/collections/%s/members/%s/delete", collection.ID.String(), m.User.ID.String())),
							Button(Type("submit"), Class("outline"), Text("Remove")),
						),
					),
				),
			)
		})),
		Div(
			Class("add-member"),
			Small(Text("Add member")),
			Form(
				Method("post"),
				Action(fmt.Sprintf("/collections/%s/members", collection.ID.String())),
				Label(Text("Email"), Input(Type("email"), Name("email"), Required())),
				Label(Text("Role"),
					Select(Name("role"),
						Option(Value("viewer"), Text("viewer"), Selected()),
						Option(Value("editor"), Text("editor")),
					),
				),
				Button(Type("submit"), Text("Add")),
			),
		),
	)

	bookmarkSection := Section(
		H2(Text("Bookmarks")),
		IfElse(len(bookmarks) > 0,
			Links(bookmarks),
			P(Class("muted"), Text("No bookmarks yet.")),
		),
	)

	content := Main(
		backToCollectionsLink(),
		Header(
			H1(Text(collection.Name)),
			A(Href("/collections/"+collection.ID.String()+"/edit"), Class("button outline"), Text("Edit")),
			Form(Method("post"), Action(fmt.Sprintf("/collections/%s/delete", collection.ID.String())),
				Button(Type("submit"), Class("danger"), Text("Delete")),
			),
		),
		If(collection.Description != "",
			P(Class("muted"), Text(collection.Description)),
		),
		Div(
			Class("detail-grid"),
			Div(Class("detail-main"), bookmarkSection),
			Aside(Class("detail-side"), memberSection),
		),
	)
	return Page(collection.Name+" - Patchworks", AppShell(user, content))
}

// EditCollectionPage renders the edit-collection form, preserving submitted
// values and field errors.
func EditCollectionPage(user core.User, f CollectionForm, id uuid.UUID) Node {
	content := Main(
		backToCollectionLink(id),
		H1(Text("Edit Collection")),
		Form(Method(http.MethodPost), Action(fmt.Sprintf("/collections/%s/edit", id.String())),
			If(f.Errors["form"] != "", Toast("error", f.Errors["form"], "")),
			TextInput("Name", "name", "text", f.Name, f.Errors, Placeholder("Name"), Required()),
			Label(Text("Description"),
				Textarea(Name("description"), Placeholder("Description"), Text(f.Description)),
				FieldError("description", f.Errors),
			),
			Button(Type("submit"), Text("Save")),
		),
	)
	return Page("Edit Collection - Patchworks", AppShell(user, content))
}

func backToCollectionsLink() Node {
	return A(Class("back-link"), Href("/collections"), Text("←  Collections"))
}

func backToCollectionLink(id uuid.UUID) Node {
	return A(Class("back-link"), Href(fmt.Sprintf("/collections/%s", id.String())), Text("←  Collection"))
}
