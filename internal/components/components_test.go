package components

import (
	"encoding/base64"
	"testing"

	"github.com/carlmjohnson/be"
)

func TestDecodeSessionKey(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString(make([]byte, 32))
	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	long := base64.StdEncoding.EncodeToString(make([]byte, 48))

	for _, tc := range []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"valid 32 bytes", valid, false},
		{"empty", "", true},
		{"not base64", "!!!not-base64!!!", true},
		{"16 bytes", short, true},
		{"48 bytes", long, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key, err := decodeSessionKey(tc.raw)
			if tc.wantErr {
				be.Nonzero(t, err)
				return
			}
			be.NilErr(t, err)
			be.Equal(t, 32, len(key))
		})
	}
}
