package sessions

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"strings"
)

type Reference struct {
	Key       string
	AccountID uint
	Version   uint
}

type referenceKey struct{}

func Capture(ctx fiber.Ctx) Reference {
	held := session.FromContext(ctx)
	if held == nil {
		return Reference{}
	}
	return Reference{Key: strings.Clone(held.ID()), AccountID: Remembered(ctx), Version: Version(ctx)}
}

func WithReference(ctx context.Context, reference Reference) context.Context {
	return context.WithValue(ctx, referenceKey{}, reference)
}

func ContextReference(ctx context.Context) Reference {
	reference, _ := ctx.Value(referenceKey{}).(Reference)
	return reference
}

func ValidReference(ctx context.Context, reference Reference) bool {
	if reference.Key == "" || reference.AccountID == 0 || sessionStore == nil {
		return false
	}
	held, err := sessionStore.GetByID(ctx, reference.Key)
	if err != nil || held == nil {
		return false
	}
	defer held.Release()
	accountID, matched := held.Get(AccountKey).(uint)
	version, versionMatched := held.Get("session-version").(uint)
	return matched && versionMatched && accountID == reference.AccountID && version == reference.Version
}
