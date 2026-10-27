package config

import "time"

type serverSettings struct {
	Domain                       string `env:"DOMAIN" envDefault:"localhost"`
	PublicURL                    string `env:"PUBLIC_URL"`
	HTTPPort                     int    `env:"HTTP_PORT" envDefault:"8800"`
	HTTPSPort                    int    `env:"HTTPS_PORT" envDefault:"8443"`
	WebAddress                   string `env:"WEB_ADDRESS" envDefault:"127.0.0.1:7070"`
	RedirectToTLS                bool   `env:"REDIRECT_TO_TLS" envDefault:"true"`
	Debug                        bool   `env:"DEBUG" envDefault:"false"`
	MaximumImageUploadBytes      int    `env:"MAXIMUM_IMAGE_UPLOAD_BYTES" envDefault:"0"`
	MaximumWebSocketMessageBytes int64  `env:"MAXIMUM_WEBSOCKET_MESSAGE_BYTES" envDefault:"0"`
}

type querySettings struct {
	MaximumResultBytes int64 `env:"QUERY_MAXIMUM_RESULT_BYTES" envDefault:"0"`
	MaximumRows        int   `env:"QUERY_MAXIMUM_ROWS" envDefault:"0"`
}

type historySettings struct {
	Retention       time.Duration `env:"HISTORY_RETENTION" envDefault:"0"`
	MaximumPictures int           `env:"PICTURE_HISTORY_LIMIT" envDefault:"0"`
}

type transferSettings struct {
	MaximumBytes          int64         `env:"TRANSFER_MAXIMUM_BYTES" envDefault:"0"`
	Timeout               time.Duration `env:"TRANSFER_TIMEOUT" envDefault:"0"`
	MaximumConcurrentJobs int           `env:"TRANSFER_MAXIMUM_CONCURRENT_JOBS" envDefault:"0"`
}

type dataSettings struct {
	Directory string `env:"DATA_DIRECTORY" envDefault:"./data"`
}

type sqldSettings struct {
	Managed      bool          `env:"SQLD_MANAGED" envDefault:"true"`
	BinaryPath   string        `env:"SQLD_BINARY" envDefault:"sqld"`
	Address      string        `env:"SQLD_ADDRESS" envDefault:"127.0.0.1:8080"`
	AdminAddress string        `env:"SQLD_ADMIN_ADDRESS" envDefault:"127.0.0.1:8081"`
	QueryTimeout time.Duration `env:"SQLD_QUERY_TIMEOUT" envDefault:"0"`
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
	MaximumConnections    int           `env:"POSTGRES_MAXIMUM_CONNECTIONS" envDefault:"0"`
	CertificatePath       string        `env:"POSTGRES_CERTIFICATE_PATH"`
	PrivateKeyPath        string        `env:"POSTGRES_PRIVATE_KEY_PATH"`
	InitializationTimeout time.Duration `env:"POSTGRES_INITIALIZATION_TIMEOUT" envDefault:"2m"`
	StartupTimeout        time.Duration `env:"POSTGRES_STARTUP_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout       time.Duration `env:"POSTGRES_SHUTDOWN_TIMEOUT" envDefault:"45s"`
}

type redisSettings struct {
	Enabled              bool          `env:"REDIS_ENABLED" envDefault:"true"`
	BinaryPath           string        `env:"REDIS_BINARY" envDefault:"/usr/local/bin/sqldash-redis-server"`
	ModulesDirectory     string        `env:"REDIS_MODULES_DIRECTORY" envDefault:"/opt/redis-runtime/modules"`
	GatewayEnabled       bool          `env:"REDIS_GATEWAY_ENABLED" envDefault:"true"`
	GatewayAddress       string        `env:"REDIS_GATEWAY_ADDRESS" envDefault:":6379"`
	PublicHost           string        `env:"REDIS_PUBLIC_HOST"`
	PublicPort           int           `env:"REDIS_PUBLIC_PORT" envDefault:"6379"`
	AllowPlaintext       bool          `env:"REDIS_ALLOW_PLAINTEXT" envDefault:"true"`
	CertificatePath      string        `env:"REDIS_CERTIFICATE_PATH"`
	PrivateKeyPath       string        `env:"REDIS_PRIVATE_KEY_PATH"`
	MemoryLimit          int64         `env:"REDIS_MEMORY_LIMIT" envDefault:"0"`
	MemoryBudget         int64         `env:"REDIS_MEMORY_BUDGET" envDefault:"0"`
	MaximumInstances     int           `env:"REDIS_MAXIMUM_INSTANCES" envDefault:"0"`
	MaximumConnections   int           `env:"REDIS_MAXIMUM_CONNECTIONS" envDefault:"0"`
	MaximumValueBytes    int64         `env:"REDIS_MAXIMUM_VALUE_BYTES" envDefault:"536870912"`
	MaximumScanBatch     int64         `env:"REDIS_MAXIMUM_SCAN_BATCH" envDefault:"1000"`
	MaximumStreamBuffer  int           `env:"REDIS_MAXIMUM_STREAM_BUFFER" envDefault:"1000"`
	StartupTimeout       time.Duration `env:"REDIS_STARTUP_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout      time.Duration `env:"REDIS_SHUTDOWN_TIMEOUT" envDefault:"45s"`
	CommandTimeout       time.Duration `env:"REDIS_COMMAND_TIMEOUT" envDefault:"0"`
	RevalidationInterval time.Duration `env:"REDIS_REVALIDATION_INTERVAL" envDefault:"2s"`
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
