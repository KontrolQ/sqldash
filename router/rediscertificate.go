package router

import (
	"github.com/gofiber/fiber/v3"
	"sqldash/config"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")
	urls.Path(urls.Get, "/redis/ca", publicRedisCertificate, "redis.certificate")
}

func publicRedisCertificate(ctx fiber.Ctx) error {
	if !config.Redis.Enabled || !config.Redis.GatewayEnabled || config.Redis.CertificatePath != "" || config.Certificates.Automatic {
		return fiber.ErrNotFound
	}
	return publicNativeCertificate(ctx, config.RedisTLSPath(), "sqldash-redis-ca.crt")
}
