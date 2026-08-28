package views

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// LoginPage renders the login form.
func LoginPage() Node {
	return Page("Log in — Patchworks",
		Main(
			Form(Method("post"), Action("/auth/login"),
				H1(Text("Log in")),
				Label(Text("Email"), Input(Type("email"), Name("email"), Required(), Attr("autocomplete", "email"))),
				Label(Text("Password"), Input(Type("password"), Name("password"), Required(), Attr("autocomplete", "current-password"))),
				Button(Type("submit"), Text("Log in")),
			),
			P(
				Text("No account? "),
				A(Href("/auth/register"), Text("Register")),
			),
		),
	)
}

// RegisterPage renders the registration form.
func RegisterPage() Node {
	return Page("Register — Patchworks",
		Main(
			Form(Method("post"), Action("/auth/register"),
				H1(Text("Register")),
				Label(Text("Email"), Input(Type("email"), Name("email"), Required(), Attr("autocomplete", "email"))),
				Label(Text("Password"), Input(Type("password"), Name("password"), Required(), Attr("autocomplete", "new-password"))),
				Button(Type("submit"), Text("Register")),
			),
			P(
				Text("Already have an account? "),
				A(Href("/auth/login"), Text("Log in")),
			),
		),
	)
}
