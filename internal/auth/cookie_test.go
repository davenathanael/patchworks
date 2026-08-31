package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/carlmjohnson/be"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := testKey(t)
	ct, err := encryptAES(key, "session-id-123")
	be.NilErr(t, err)

	got, err := decryptAES(key, ct)
	be.NilErr(t, err)
	be.Equal(t, "session-id-123", got)
}

func TestEncryptIsRandomized(t *testing.T) {
	key := testKey(t)
	a, err := encryptAES(key, "same")
	be.NilErr(t, err)
	b, err := encryptAES(key, "same")
	be.NilErr(t, err)

	be.Unequal(t, a, b) // fresh nonce per call
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	key := testKey(t)
	ct, err := encryptAES(key, "session-id")
	be.NilErr(t, err)

	tampered := []byte(ct)
	tampered[len(tampered)-1] ^= 1
	_, err = decryptAES(key, string(tampered))
	be.Nonzero(t, err) // AES-GCM must reject any modification
}

func TestDecryptWrongKey(t *testing.T) {
	k1, k2 := testKey(t), testKey(t)
	ct, err := encryptAES(k1, "x")
	be.NilErr(t, err)

	_, err = decryptAES(k2, ct)
	be.Nonzero(t, err)
}

func TestDecryptGarbage(t *testing.T) {
	key := testKey(t)
	_, err := decryptAES(key, "not-base64!!!")
	be.Nonzero(t, err)

	_, err = decryptAES(key, "")
	be.Nonzero(t, err)
}

func TestDecryptShortCiphertext(t *testing.T) {
	key := testKey(t)
	// Valid base64 that decodes to fewer bytes than the GCM nonce.
	short := base64.URLEncoding.EncodeToString([]byte("tiny"))

	_, err := decryptAES(key, short)
	be.Nonzero(t, err)
}

func TestSetSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	err := SetSessionCookie(w, CookieConfig{Key: testKey(t), Secure: true}, "session-1")
	be.NilErr(t, err)

	cookies := w.Result().Cookies()
	be.Equal(t, 1, len(cookies))
	be.Equal(t, cookieName, cookies[0].Name)
	be.Equal(t, 30*24*3600, cookies[0].MaxAge) // 30 days, matches sessionLifetime
	be.True(t, cookies[0].HttpOnly)
	be.True(t, cookies[0].Secure)
	be.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
}

func TestGetSessionCookieRoundTrip(t *testing.T) {
	key := testKey(t)
	w := httptest.NewRecorder()
	be.NilErr(t, SetSessionCookie(w, CookieConfig{Key: key}, "session-1"))

	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}

	got, err := GetSessionCookie(r, CookieConfig{Key: key})
	be.NilErr(t, err)
	be.Equal(t, "session-1", got)
}

func TestSessionCookieSecureFlag(t *testing.T) {
	for _, tc := range []struct {
		name   string
		secure bool
	}{
		{"secure", true},
		{"plain http", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			be.NilErr(t, SetSessionCookie(w, CookieConfig{Key: testKey(t), Secure: tc.secure}, "s"))
			be.Equal(t, tc.secure, w.Result().Cookies()[0].Secure)
		})
	}
}

func TestDeleteSessionCookieSecureFlag(t *testing.T) {
	w := httptest.NewRecorder()
	DeleteSessionCookie(w, CookieConfig{Key: testKey(t), Secure: true})
	be.True(t, w.Result().Cookies()[0].Secure)
}

// --- helpers ---

// testKey returns a fresh 32-byte AES key.
func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	be.NilErr(t, err)
	return key
}
