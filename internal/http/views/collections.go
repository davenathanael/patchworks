package views

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/google/uuid"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func ListCollectionsPage(collections []core.Collection, user core.User) Node {
	collectionItem := func(c core.Collection) Node {
		return Div(
			Class("collection-row"),
			Div(
				Class("collection-info"),
				A(
					Class("collection-name"),
					Href(fmt.Sprintf("/collections/%s", c.ID.String())),
					Text(c.Name),
				),
				P(Class("collection-desc"), Text(c.Description)),
				Span(Class("bookmark-count"), Text(fmt.Sprintf("%d bookmarks", c.BookmarkCount))),
			),
			Div(
				Class("collection-meta"),
				AvatarStack(c.Members),
			),
		)
	}

	content := Main(Class("container"),
		Div(Class("page-heading"),
			H4(Text("Your Collections")),
			A(Href("/collections/new"), Class("button outline small"), Text("New")),
		),
		Div(Map(collections, collectionItem)),
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

	return Figure(
		Data("variant", "avatar"),
		Class("small"),
		Attr("role", "group"),
		Group(avatars),
	)
}

func Avatar(email string) Node {
	initials := initialsFromEmail(email)
	return Figure(
		Data("variant", "avatar"),
		Class("small"),
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
	content := Main(Class("container"),
		backToCollectionsLink(),
		H4(Text("Create a Collection")),
		Form(Method("post"), Action("/collections"),
			Input(Type("text"), Name("name"), Placeholder("Name"), Attr("data-field"), Required()),
			Textarea(Name("description"), Attr("data-field"), Placeholder("Description")),
			Button(Type("submit"), Text("Create")),
		),
	)
	return Page("Create New Collection - Patchworks", AppShell(user, content))
}

func CollectionPage(collection core.Collection, bookmarks []core.Bookmark, user core.User) Node {
	memberSection := Div(
		Class("mb-6"),
		H6(Text("Members")),
		Div(Map(collection.Members, func(m core.CollectionMember) Node {
			return Div(
				Class("member-row"),
				Div(
					Class("member-info"),
					Avatar(m.User.Email),
					Span(Class("member-email"), Text(m.User.Email)),
					Span(Class("member-role"), Text(m.Role)),
					Span(Class("member-added"), Text(relativeTime(m.AddedAt))),
				),
				If(m.User.ID != user.ID,
					Form(Method("post"), Action(fmt.Sprintf("/collections/%s/members/%s/delete", collection.ID.String(), m.User.ID.String())),
						Button(Type("submit"), Class("button outline small"), Text("Remove")),
					),
				),
			)
		})),
		Form(
			Class("add-member-form"),
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

	bookmarkSection := Div(
		H6(Text("Bookmarks")),
		IfElse(len(bookmarks) > 0,
			Links(ToLinkItems(bookmarks)),
			P(Class("text-muted"), Text("No bookmarks yet.")),
		),
	)

	content := Main(Class("container"),
		Div(
			backToCollectionsLink(),
			Div(
				Class("collection-detail-heading"),
				H4(Text(collection.Name)),
				Div(
					Class("actions"),
					A(Href("/collections/"+collection.ID.String()+"/edit"), Class("button outline small"), Text("Edit")),
					Form(Method("post"), Action(fmt.Sprintf("/collections/%s/delete", collection.ID.String())),
						Button(Type("submit"), Class("button outline small"), Text("Delete")),
					),
				),
			),
			If(collection.Description != "",
				P(Class("text-muted"), Text(collection.Description)),
			),
			memberSection,
			bookmarkSection,
		),
	)
	return Page(collection.Name+" - Patchworks", AppShell(user, content))
}

func EditCollectionPage(collection core.Collection, user core.User) Node {
	content := Main(Class("container"),
		backToCollectionLink(collection.ID),
		H4(Text("Edit Collection")),
		Form(Method(http.MethodPost), Action(fmt.Sprintf("/collections/%s/edit", collection.ID.String())),
			Label(Attr("data-field"), Text("Name"),
				Input(Type("text"), Name("name"), Value(collection.Name), Placeholder("Name"), Required()),
			),
			Label(Attr("data-field"), Text("Description"),
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
