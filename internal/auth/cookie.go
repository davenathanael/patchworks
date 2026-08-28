package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
)

const (
	cookieName     = "session_id"
	cookieMaxAge   = 86400 * 30 // 30 days
	cookiePath     = "/"
	cookieHttpOnly = true
	cookieSecure   = true
)

// CookieConfig holds the encryption key for session cookies.
type CookieConfig struct {
	Key []byte // 32 bytes for AES-256
}

// SetSessionCookie encrypts the session ID and sets it in an HTTP cookie.
func SetSessionCookie(w http.ResponseWriter, cfg CookieConfig, sessionID string) error {
	encrypted, err := encryptAES(cfg.Key, sessionID)
	if err != nil {
		return fmt.Errorf("encrypt session: %w", err)
	}

	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    encrypted,
		Path:     cookiePath,
		MaxAge:   cookieMaxAge,
		HttpOnly: cookieHttpOnly,
		Secure:   cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	return nil
}

// GetSessionCookie retrieves and decrypts the session ID from the request's cookie.
func GetSessionCookie(r *http.Request, cfg CookieConfig) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", err // http.ErrNoCookie or other error
	}

	sessionID, err := decryptAES(cfg.Key, cookie.Value)
	if err != nil {
		return "", fmt.Errorf("decrypt session: %w", err)
	}
	return sessionID, nil
}

// DeleteSessionCookie clears the session cookie.
func DeleteSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     cookiePath,
		MaxAge:   -1,
		HttpOnly: cookieHttpOnly,
		Secure:   cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// encryptAES encrypts data using AES-GCM with the given key.
// Returns base64url-encoded ciphertext + nonce.
func encryptAES(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decryptAES decrypts base64url-encoded data using AES-GCM with the given key.
func decryptAES(key []byte, encrypted string) (string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
