package resource

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
)

// Etag computes an etag based on containt of the payload.
func GenEtag(payload map[string]interface{}) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum(b)), nil
}
