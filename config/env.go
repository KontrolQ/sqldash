package config

import "time"

type serverSettings struct {
	Domain                  string `env:"DOMAIN" envDefault:"localhost"`
	PublicURL               string `env:"PUBLIC_URL"`
	HTTPPort                int    `env:"HTTP_PORT" envDefault:"8800"`
	HTTPSPort               int    `env:"HTTPS_PORT" envDefault:"8443"`
	WebAddress              string `env:"WEB_ADDRESS" envDefault:"127.0.0.1:7070"`
	RedirectToTLS           bool   `env:"REDIRECT_TO_TLS" envDefault:"true"`
	Debug                   bool   `env:"DEBUG" envDefault:"false"`
	MaximumImageUploadBytes int    `env:"MAXIMUM_IMAGE_UPLOAD_BYTES" envDefault:"10485760"`
}

type dataSettings struct {
	Directory string `env:"DATA_DIRECTORY" envDefault:"./data"`
}

type sqldSettings struct {
	Managed      bool   `env:"SQLD_MANAGED" envDefault:"true"`
	BinaryPath   string `env:"SQLD_BINARY" envDefault:"sqld"`
	Address      string `env:"SQLD_ADDRESS" envDefault:"127.0.0.1:8080"`
	AdminAddress string `env:"SQLD_ADMIN_ADDRESS" envDefault:"127.0.0.1:8081"`
}

type postgresSettings struct {
	Enabled               bool          `env:"POSTGRES_ENABLED" envDefault:"true"`
	Managed               bool          `env:"POSTGRES_MANAGED" envDefault:"true"`
	BinaryDirectory       string        `env:"POSTGRES_BINARY_DIRECTORY" envDefault:"/usr/lib/postgresql/18/bin"`
	Address               string        `env:"POSTGRES_ADDRESS" envDefault:"127.0.0.1:55432"`
	GatewayAddress        string        `env:"POSTGRES_GATEWAY_ADDRESS" envDefault:":5432"`
	GatewayEnabled        bool          `env:"POSTGRES_GATEWAY_ENABLED" envDefault:"true"`
	PublicHost            string        `env:"POSTGRES_PUBLIC_HOST"`
	PublicPort            int           `env:"POSTGRES_PUBLIC_PORT" envDefault:"5432"`
	AdminUsername         string        `env:"POSTGRES_ADMIN_USERNAME" envDefault:"sqldash_admin"`
	MaximumConnections    int           `env:"POSTGRES_MAXIMUM_CONNECTIONS" envDefault:"100"`
	CertificatePath       string        `env:"POSTGRES_CERTIFICATE_PATH"`
	PrivateKeyPath        string        `env:"POSTGRES_PRIVATE_KEY_PATH"`
	InitializationTimeout time.Duration `env:"POSTGRES_INITIALIZATION_TIMEOUT" envDefault:"2m"`
	StartupTimeout        time.Duration `env:"POSTGRES_STARTUP_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout       time.Duration `env:"POSTGRES_SHUTDOWN_TIMEOUT" envDefault:"45s"`
}

type certificateSettings struct {
	Automatic          bool   `env:"AUTOMATIC_CERTIFICATES" envDefault:"false"`
	Staging            bool   `env:"CERTIFICATE_STAGING" envDefault:"false"`
	Email              string `env:"CERTIFICATE_EMAIL"`
	CloudflareAPIToken string `env:"CLOUDFLARE_API_TOKEN"`
}

type hostingSettings struct {
	Address  string `env:"HOSTING_ADDRESS"`
	Password string `env:"HOSTING_PASSWORD"`
	App      string `env:"HOSTING_APP"`
}

type accessSettings struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`
}

type sessionSettings struct {
	CookieName string `env:"SESSION_COOKIE" envDefault:"sqldash"`
}
