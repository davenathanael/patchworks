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

func ListCollectionsPage(collections []core.Collection, user core.User) Node {
	collectionItem := func(c core.Collection) Node {
		return Li(
			Article(
				Header(
					H3(
						A(Href(fmt.Sprintf("/collections/%s", c.ID.String())), Text(c.Name)),
					),
					If(c.Description != "", P(Text(c.Description))),
					Small(Text(fmt.Sprintf("%d bookmarks", c.BookmarkCount))),
				),
				Footer(
					AvatarStack(c.Members),
				),
			),
		)
	}

	content := Main(
		Header(
			H1(Text("Your Collections")),
			A(Href("/collections/new"), Class("button outline small"), Text("New")),
		),
		Ul(Class("collection-list"), Map(collections, collectionItem)),
	)
	return Page("Collections - Patchworks", AppShell(user, content))
}

func AvatarStack(members []core.CollectionMember) Node {
	if len(members) == 0 {
		return Div()
	}

	if len(members) == 1 {
		return Avatar(members[0].User.Email)
	}

	avatars := make([]Node, 0, len(members))
	for _, m := range members {
		avatars = append(avatars, Avatar(m.User.Email))
	}

	return Div(
		Class("avatar-stack"),
		Attr("role", "group"),
		Group(avatars),
	)
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

func CreateCollectionsPage(user core.User) Node {
	content := Main(
		backToCollectionsLink(),
		H1(Text("Create a Collection")),
		Form(Method("post"), Action("/collections"),
			Input(Type("text"), Name("name"), Placeholder("Name"), Required()),
			Textarea(Name("description"), Placeholder("Description")),
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
							Button(Type("submit"), Class("outline small"), Text("Remove")),
						),
					),
				),
			)
		})),
		Form(
			Method("post"),
			Action(fmt.Sprintf("/collections/%s/members", collection.ID.String())),
			Input(Type("email"), Name("email"), Placeholder("Email"), Required()),
			Select(Name("role"),
				Option(Value("viewer"), Text("viewer"), Selected()),
				Option(Value("editor"), Text("editor")),
			),
			Button(Type("submit"), Text("Add")),
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
			A(Href("/collections/"+collection.ID.String()+"/edit"), Class("button outline small"), Text("Edit")),
			Form(Method("post"), Action(fmt.Sprintf("/collections/%s/delete", collection.ID.String())),
				Button(Type("submit"), Class("outline small"), Text("Delete")),
			),
		),
		If(collection.Description != "",
			P(Class("muted"), Text(collection.Description)),
		),
		memberSection,
		bookmarkSection,
	)
	return Page(collection.Name+" - Patchworks", AppShell(user, content))
}

func EditCollectionPage(collection core.Collection, user core.User) Node {
	content := Main(
		backToCollectionLink(collection.ID),
		H1(Text("Edit Collection")),
		Form(Method(http.MethodPost), Action(fmt.Sprintf("/collections/%s/edit", collection.ID.String())),
			Label(Text("Name"),
				Input(Type("text"), Name("name"), Value(collection.Name), Placeholder("Name"), Required()),
			),
			Label(Text("Description"),
				Textarea(Name("description"), Placeholder("Description"), Text(collection.Description)),
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
