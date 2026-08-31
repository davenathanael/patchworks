package client

import (
	"testing"

	"github.com/carlmjohnson/be"
)

func TestExtractTitle(t *testing.T) {
	for _, tc := range []struct {
		name string
		html string
		want string
	}{
		{"plain", "<html><head><title>Hello</title></head></html>", "Hello"},
		{"no title", "<html><body>nope</body></html>", ""},
		{"title with attribute", `<title id="main">Attr</title>`, "Attr"},
		{"unclosed tag", "<html><title>dangling", ""},
		{"unclosed content", "<html><title>Hello</html>", ""},
		{"uppercase tag", "<HTML><TITLE>UPPER</TITLE></HTML>", "UPPER"},
		{"trims whitespace", "<title>  padded  </title>", "padded"},
		{"angle bracket in content", "<title>worse &gt; than</title>", "worse &gt; than"},
		{"empty title", "<title></title>", ""},
		{"title after body content", "<body>x</body><title>late</title>", "late"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			be.Equal(t, tc.want, extractTitle(tc.html))
		})
	}
}
