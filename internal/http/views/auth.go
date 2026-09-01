package views

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// CredentialsForm is the shared login/register form view-model: the ajg/form
// decode target and the render model. Zero value renders a fresh form.
type CredentialsForm struct {
	Email    string     `form:"email"`
	Password string     `form:"password"`
	Errors   FormErrors `form:"-"`
}

// LoginPage renders the login form, preserving submitted values and field errors.
func LoginPage(f CredentialsForm) Node {
	return Page("Log in — Patchworks",
		Main(
			Form(Method("post"), Action("/auth/login"),
				If(f.Errors["form"] != "", Toast("error", f.Errors["form"], "")),
				H1(Text("Log in")),
				TextInput("Email", "email", "email", f.Email, f.Errors,
					Required(), Attr("autocomplete", "email"),
				),
				TextInput("Password", "password", "password", f.Password, f.Errors,
					Required(), Attr("autocomplete", "current-password"),
				),
				Button(Type("submit"), Text("Log in")),
			),
			P(
				Text("No account? "),
				A(Href("/auth/register"), Text("Register")),
			),
		),
	)
}

// RegisterPage renders the registration form, preserving submitted values and field errors.
func RegisterPage(f CredentialsForm) Node {
	return Page("Register — Patchworks",
		Main(
			Form(Method("post"), Action("/auth/register"),
				If(f.Errors["form"] != "", Toast("error", f.Errors["form"], "")),
				H1(Text("Register")),
				TextInput("Email", "email", "email", f.Email, f.Errors,
					Required(), Attr("autocomplete", "email"),
				),
				TextInput("Password", "password", "password", f.Password, f.Errors,
					Required(), Attr("autocomplete", "new-password"),
				),
				Button(Type("submit"), Text("Register")),
			),
			P(
				Text("Already have an account? "),
				A(Href("/auth/login"), Text("Log in")),
			),
		),
	)
}
