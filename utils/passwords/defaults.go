package passwords

const (
	LogPrefix = "Passwords"

	SaltLength = 16
	KeyLength  = 32
	TimeCost   = 3
	MemoryCost = 64 * 1024
	Threads    = 2

	Algorithm     = "argon2id"
	Separator     = "$"
	Format        = "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
	VersionFormat = "v=%d"
	CostFormat    = "m=%d,t=%d,p=%d"

	FieldCount     = 6
	CostFieldCount = 3
	AlgorithmField = 1
	VersionField   = 2
	CostField      = 3
	SaltField      = 4
	KeyField       = 5

	MinimumSize = 8
)
