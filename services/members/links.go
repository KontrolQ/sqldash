package members

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/services/mailing"
	"sqldash/utils/tokens"

	"gorm.io/gorm"
)

type IssuedLink struct {
	URL             string
	Email           string
	Sent            bool
	DeliveryProblem string
}

func Origin() (string, error) {
	origin := strings.TrimRight(config.Server.PublicURL, "/")
	if origin == "" {
		if config.Server.Domain == "localhost" {
			_, port, splitError := net.SplitHostPort(config.Server.WebAddress)
			if splitError != nil {
				return "", Invalid
			}
			origin = "http://localhost:" + port
		} else {
			origin = "https://" + config.Server.Domain
		}
	}
	parsed, parseError := url.Parse(origin)
	if parseError != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1"))) {
		return "", Invalid
	}
	return origin, nil
}

func Issue(actor *access.AccountAccess, kind string, email string, accountID uint, roles []uint, databases []uint, all bool, sendEmail ...bool) (*IssuedLink, error) {
	deliver := len(sendEmail) == 0 || sendEmail[0]
	return issueLink(actor, kind, email, accountID, roles, databases, all, 0, deliver)
}

func issueLink(actor *access.AccountAccess, kind string, email string, accountID uint, roles []uint, databases []uint, all bool, previousID uint, sendEmail bool) (*IssuedLink, error) {
	origin, originError := Origin()
	if originError != nil {
		return nil, originError
	}
	secret := make([]byte, 32)
	if _, readError := rand.Read(secret); readError != nil {
		return nil, readError
	}
	shown := base64.RawURLEncoding.EncodeToString(secret)
	path, subject := "/join", InvitationSubject
	link := &models.AccountLink{}
	saveError := database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := currentActor(transaction, actor, permissions.UsersManage)
		if findError != nil {
			return findError
		}
		if previousID != 0 {
			previous, findError := manageableLink(transaction, current, previousID)
			if findError != nil {
				return findError
			}
			kind, email, accountID = previous.Kind, previous.Email, previous.AccountID
			roles, databases, all = decodeIDs(previous.RoleIDs), decodeIDs(previous.DatabaseIDs), previous.AllDatabases
		}
		lifetime := InvitationLifetime
		switch kind {
		case Invitation:
			roles, databases, all = nil, nil, false
			email = strings.ToLower(strings.TrimSpace(email))
			var existing int64
			if countError := transaction.Model(&models.Account{}).Where("lower(email) = ?", email).Count(&existing).Error; countError != nil {
				return countError
			}
			if existing != 0 {
				return Duplicate
			}
		case Recovery:
			target, findError := access.LoadWith(transaction, accountID)
			if findError != nil || !current.CanManage(target) || !target.Account.Active {
				return Refused
			}
			email = target.Account.Email
			lifetime, path, subject = RecoveryLifetime, "/recover", RecoverySubject
		default:
			return Invalid
		}
		address, parseError := mail.ParseAddress(email)
		if parseError != nil || address.Address != email {
			return Invalid
		}
		now := time.Now()
		if revokeError := transaction.Model(&models.AccountLink{}).Where("kind = ? AND email = ? AND used_at IS NULL AND revoked_at IS NULL", kind, email).Update("revoked_at", now).Error; revokeError != nil {
			return revokeError
		}
		*link = models.AccountLink{Kind: kind, Digest: tokens.Digest(shown), Email: email, IssuerID: current.Account.ID, AccountID: accountID, RoleIDs: encodeIDs(roles), DatabaseIDs: encodeIDs(databases), AllDatabases: all, ExpiresAt: now.Add(lifetime)}
		return transaction.Create(link).Error
	})
	if saveError != nil {
		return nil, saveError
	}
	result := &IssuedLink{URL: origin + path + "?token=" + url.QueryEscape(shown), Email: link.Email}
	if !sendEmail {
		return result, nil
	}
	var deliveryError error
	result.Sent, deliveryError = mailing.Send(link.Email, subject, "Follow this single-use link to choose your own credentials:\n\n"+result.URL+"\n\nIf you did not expect this message, contact the administrator.")
	if deliveryError != nil {
		result.DeliveryProblem = mailing.DeliveryFailed.Error()
	}
	return result, nil
}

func FindLink(kind string, secret string) (*models.AccountLink, error) {
	return findLinkWith(database.DB, kind, secret)
}

func findLinkWith(store *gorm.DB, kind string, secret string) (*models.AccountLink, error) {
	if len(secret) < 32 || len(secret) > 128 {
		return nil, LinkInvalid
	}
	var link models.AccountLink
	findError := store.Where("kind = ? AND digest = ? AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ?", kind, tokens.Digest(secret), time.Now()).First(&link).Error
	if findError != nil {
		return nil, LinkInvalid
	}
	return &link, nil
}

func manageableLink(transaction *gorm.DB, actor *access.AccountAccess, identifier uint) (*models.AccountLink, error) {
	var link models.AccountLink
	if findError := transaction.Where("id = ? AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ?", identifier, time.Now()).First(&link).Error; findError != nil {
		return nil, LinkInvalid
	}
	issuer, findError := access.LoadWith(transaction, link.IssuerID)
	if findError != nil {
		return nil, Refused
	}
	if !actor.Owner && actor.Account.ID != link.IssuerID && !actor.CanManage(issuer) {
		return nil, Refused
	}
	return &link, nil
}

func CancelLink(actor *access.AccountAccess, identifier uint) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := currentActor(transaction, actor, permissions.UsersManage)
		if findError != nil {
			return findError
		}
		link, findError := manageableLink(transaction, current, identifier)
		if findError != nil {
			return findError
		}
		return transaction.Model(link).Update("revoked_at", time.Now()).Error
	})
}

func ReissueLink(actor *access.AccountAccess, identifier uint) (*IssuedLink, error) {
	return issueLink(actor, "", "", 0, nil, nil, false, identifier, true)
}
