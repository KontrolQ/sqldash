package postgresconnections

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type pendingPassword struct {
	AccountID uint
	Result    *IssuedPassword
	ExpiresAt time.Time
}

var issuedPasswords = struct {
	sync.Mutex
	Values map[string]pendingPassword
}{Values: make(map[string]pendingPassword)}

func HoldIssuedPassword(accountID uint, result *IssuedPassword) (string, error) {
	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	ticket := base64.RawURLEncoding.EncodeToString(secret)
	issuedPasswords.Lock()
	defer issuedPasswords.Unlock()
	now := time.Now()
	for ticket, pending := range issuedPasswords.Values {
		if !pending.ExpiresAt.After(now) {
			delete(issuedPasswords.Values, ticket)
		}
	}
	issuedPasswords.Values[ticket] = pendingPassword{AccountID: accountID, Result: result, ExpiresAt: now.Add(10 * time.Minute)}
	return ticket, nil
}

func TakeIssuedPassword(accountID, databaseID uint, ticket string) *IssuedPassword {
	issuedPasswords.Lock()
	defer issuedPasswords.Unlock()
	pending, exists := issuedPasswords.Values[ticket]
	if !exists || pending.AccountID != accountID || pending.Result.DatabaseID != databaseID || !pending.ExpiresAt.After(time.Now()) {
		return nil
	}
	delete(issuedPasswords.Values, ticket)
	return pending.Result
}
