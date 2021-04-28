package smtp

import (
	"crypto/rand"
	"encoding/base64"
)

func generateID() string {
	bytes := make([]byte, 18)
	rand.Read(bytes)

	return base64.RawURLEncoding.EncodeToString(bytes)
}
