package members

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type pendingLink struct {
	AccountID uint
	Link      *IssuedLink
	ExpiresAt time.Time
}

var issuedLinks = struct {
	sync.Mutex
	Values map[string]pendingLink
}{Values: make(map[string]pendingLink)}

func HoldIssuedLink(accountID uint, link *IssuedLink) (string, error) {
	secret := make([]byte, 24)
	if _, readError := rand.Read(secret); readError != nil {
		return "", readError
	}
	key := base64.RawURLEncoding.EncodeToString(secret)
	issuedLinks.Lock()
	defer issuedLinks.Unlock()
	now := time.Now()
	for key, held := range issuedLinks.Values {
		if !held.ExpiresAt.After(now) {
			delete(issuedLinks.Values, key)
		}
	}
	issuedLinks.Values[key] = pendingLink{AccountID: accountID, Link: link, ExpiresAt: now.Add(10 * time.Minute)}
	return key, nil
}

func TakeIssuedLink(accountID uint, key string) *IssuedLink {
	issuedLinks.Lock()
	defer issuedLinks.Unlock()
	held, exists := issuedLinks.Values[key]
	if !exists || held.AccountID != accountID || !held.ExpiresAt.After(time.Now()) {
		return nil
	}
	delete(issuedLinks.Values, key)
	return held.Link
}
