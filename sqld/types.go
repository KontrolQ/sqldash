package sqld

type StatisticsRecord struct {
	Identifier       string  `json:"id"`
	RowsRead         int64   `json:"rows_read_count"`
	RowsWritten      int64   `json:"rows_written_count"`
	StorageBytesUsed int64   `json:"storage_bytes_used"`
	ReplicationIndex int64   `json:"replication_index"`
	QueryCount       int64   `json:"query_count"`
	ElapsedMs        float64 `json:"elapsed_ms"`
}

type ConfigurationRecord struct {
	BlockReads     bool    `json:"block_reads"`
	BlockWrites    bool    `json:"block_writes"`
	BlockReason    *string `json:"block_reason"`
	MaximumSize    string  `json:"max_db_size,omitempty"`
	HeartbeatURL   *string `json:"heartbeat_url"`
	JWTKey         *string `json:"jwt_key"`
	AllowAttach    bool    `json:"allow_attach"`
	TxnTimeoutS    int     `json:"txn_timeout_s,omitempty"`
	DurabilityMode string  `json:"durability_mode,omitempty"`
}
