package auth

import (
	"strings"
	"testing"

	"github.com/carlmjohnson/be"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := hashPassword("correct horse battery staple")
	be.NilErr(t, err)
	be.True(t, strings.HasPrefix(hash, "$argon2id$v=19$"))

	be.True(t, verifyPassword("correct horse battery staple", hash))
	be.False(t, verifyPassword("wrong password", hash))
}

func TestVerifyPasswordRejectsMalformed(t *testing.T) {
	for _, tc := range []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"wrong algorithm", "$bcrypt$v=19$m=65536,t=1,p=4$c2FsdA$c2FsdA"},
		{"wrong version", "$argon2id$v=18$m=65536,t=1,p=4$c2FsdA$c2FsdA"},
		{"missing params", "$argon2id$v=19$c2FsdA$c2FsdA"},
		{"bad salt base64", "$argon2id$v=19$m=65536,t=1,p=4$!!!$c2FsdA"},
		{"bad hash base64", "$argon2id$v=19$m=65536,t=1,p=4$c2FsdA$!!!"},
		{"too few parts", "garbage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			be.False(t, verifyPassword("anything", tc.hash))
		})
	}
}
