package views

import (
	"strings"
	"testing"

	"github.com/carlmjohnson/be"
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
