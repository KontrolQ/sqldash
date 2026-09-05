package config

type serverSettings struct {
	Domain     string `env:"DOMAIN" envDefault:"localhost"`
	HTTPPort   int    `env:"HTTP_PORT" envDefault:"8800"`
	HTTPSPort  int    `env:"HTTPS_PORT" envDefault:"8443"`
	WebAddress string `env:"WEB_ADDRESS" envDefault:"127.0.0.1:7070"`
	Debug      bool   `env:"DEBUG" envDefault:"false"`
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

type certificateSettings struct {
	Automatic          bool   `env:"AUTOMATIC_CERTIFICATES" envDefault:"false"`
	Staging            bool   `env:"CERTIFICATE_STAGING" envDefault:"false"`
	Email              string `env:"CERTIFICATE_EMAIL"`
	CloudflareAPIToken string `env:"CLOUDFLARE_API_TOKEN"`
}

type accessSettings struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`
}

type sessionSettings struct {
	CookieName string `env:"SESSION_COOKIE" envDefault:"sqldash"`
}
