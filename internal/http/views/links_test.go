package views

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/carlmjohnson/be"
	"github.com/davenathanael/patchworks/internal/core"
	"github.com/google/uuid"
	. "maragu.dev/gomponents"
)

func TestShouldShowNoteToggle(t *testing.T) {
	tests := []struct {
		name  string
		notes string
		want  bool
	}{
		{"empty", "", false},
		{"short", "quick reminder", false},
		{"at threshold", strings.Repeat("a", 45), false},
		{"one past threshold", strings.Repeat("a", 46), true},
		{"long", "This note is long enough that it will almost certainly wrap onto a second line in the row column, so the expand affordance must appear.", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			be.Equal(t, tc.want, shouldShowNoteToggle(tc.notes))
		})
	}
}

func TestPagerURL(t *testing.T) {
	cursor := core.BookmarkCursor{
		CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
		ID:        uuid.MustParse("33333333-3333-3333-3333-333333333333"),
	}
	token := core.EncodeCursor(cursor)
	tests := []struct {
		name string
		qs   url.Values
		key  string
		want string
	}{
		{"fresh view", url.Values{}, "older", "?older=" + token},
		{"filters kept, stale cursor replaced", url.Values{"older": {"stale"}, "tags": {"go"}}, "older", "?older=" + token + "&tags=go"},
		{"other direction dropped", url.Values{"newer": {"stale"}, "search": {"x"}}, "older", "?older=" + token + "&search=x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			be.Equal(t, tc.want, pagerURL(tc.qs, tc.key, cursor))
		})
	}
}

func TestPagerURLWithoutCursors(t *testing.T) {
	tests := []struct {
		name string
		qs   url.Values
		want string
	}{
		{"cursors stripped, filters kept", url.Values{"older": {"stale"}, "newer": {"stale"}, "search": {"x"}}, "?search=x"},
		{"no filters, no trailing ?", url.Values{}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			be.Equal(t, tc.want, pagerURLWithoutCursors(tc.qs))
		})
	}
}

func TestListPagerButtons(t *testing.T) {
	last := mustPagedBookmark(t, "Last")
	props := ListPagerProps{
		NavID:  "recent-pager",
		ListID: "recent-list",
		Base:   "/",
		Query:  url.Values{"tags": {"go"}},
		Page:   core.BookmarkPage{Items: []core.Bookmark{mustPagedBookmark(t, "First"), last}, HasOlder: true, HasNewer: true},
	}

	html := renderNode(t, ListPager(props))
	be.True(t, strings.Contains(html, "Back to latest"))
	be.True(t, strings.Contains(html, "Load more"))
	// the back link is plain: no cursor params, filters kept, no htmx swap
	be.True(t, strings.Contains(html, `href="/?tags=go"`))
	// both controls read as one group: same button style, flex gap via .pager
	be.True(t, strings.Count(html, `class="button outline small"`) == 2)
	be.True(t, strings.Contains(html, `hx-target="#recent-list"`))
	be.True(t, strings.Contains(html, `hx-swap="beforeend"`)) // more appends
	be.False(t, strings.Contains(html, `hx-push-url`))        // position is ephemeral — refresh returns to the head
	// the load-more fallback href carries the cursor of the last visible row
	be.True(t, strings.Contains(html, "older="+url.QueryEscape(core.EncodeCursor(core.CursorOf(last)))))
	be.True(t, strings.Contains(html, "tags=go")) // filters preserved
	be.False(t, strings.Contains(html, `hx-swap-oob`))
}

func TestListPagerVisibility(t *testing.T) {
	item := mustPagedBookmark(t, "Only")
	tests := []struct {
		name      string
		page      core.BookmarkPage
		wantOlder bool
		wantBack  bool
		wantNoNav bool
	}{
		{"first page, more below", core.BookmarkPage{Items: []core.Bookmark{item}, HasOlder: true}, true, false, false},
		{"older batch, mid-feed window", core.BookmarkPage{Items: []core.Bookmark{item}, HasOlder: true, HasNewer: true}, true, true, false},
		{"window at list end, no more below", core.BookmarkPage{Items: []core.Bookmark{item}, HasNewer: true}, false, true, false},
		{"exactly one page", core.BookmarkPage{Items: []core.Bookmark{item}}, false, false, true},
		{"empty", core.BookmarkPage{}, false, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			props := ListPagerProps{NavID: "p", ListID: "l", Base: "/", Page: tc.page}
			node := ListPager(props)
			if tc.wantNoNav {
				be.True(t, node == nil)
				return
			}
			html := renderNode(t, node)
			be.Equal(t, tc.wantOlder, strings.Contains(html, "Load more"))
			be.Equal(t, tc.wantBack, strings.Contains(html, "Back to latest"))
			be.False(t, strings.Contains(html, "Load previous"))
		})
	}
}

func TestListPagerOOB(t *testing.T) {
	props := ListPagerProps{
		NavID:  "recent-pager",
		ListID: "recent-list",
		Base:   "/",
		Page:   core.BookmarkPage{Items: []core.Bookmark{mustPagedBookmark(t, "A")}, HasOlder: true},
		OOB:    true,
	}
	be.True(t, strings.Contains(renderNode(t, ListPager(props)), `hx-swap-oob="true"`))
}

func mustPagedBookmark(t *testing.T, title string) core.Bookmark {
	t.Helper()
	return core.Bookmark{ID: uuid.New(), CreatedAt: time.Now(), Title: title}
}

func renderNode(t *testing.T, n Node) string {
	t.Helper()
	var sb strings.Builder
	be.NilErr(t, n.Render(&sb))
	return sb.String()
}
