package webapi

import (
	"encoding/base64"
	"net/http"
	"regexp"
)

func AppendSpotifyBase64AuthCredentialsRequest(req *http.Request, client_id, client_secret string) {
	authCodeBase64 := base64.StdEncoding.EncodeToString([]byte(client_id + ":" + client_secret))
	req.Header.Add("Authorization", "Basic "+authCodeBase64)
}

// ASCII-only, letters + digits + underscores, in length range [3,30]
func UsernameIsValid(username string) bool {

	// make sure that username has at least one ascii letter
	hasAscii := false
	for _, curByte := range []byte(username) {
		if (curByte >= 65 && curByte <= 90) || (curByte >= 97 && curByte <= 122) {
			hasAscii = true
			break
		}
	}

	return hasAscii && regexp.MustCompile(`^([a-z]|[A-Z]|[0-9]|_){3,30}$`).MatchString(username)
}
