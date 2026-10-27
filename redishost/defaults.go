package redishost

import "time"

const (
	manifestVersion        = 1
	systemProfile          = Profile("control")
	instanceUserBase       = 200000
	maximumDatabaseID      = 2000000000
	instanceOverhead       = 64 * 1024 * 1024
	InstanceMemoryOverhead = instanceOverhead
	privateClientReserve   = 32
	controlTimeout         = 5 * time.Second
	maximumControlBytes    = 4096
	maximumManifestBytes   = 1024 * 1024
)

var systemProfiles = []Profile{systemProfile, ReadOnly, ReadWrite, Inspect, Monitor, Subscribe, Publish}

var moduleFiles = []string{"redisbloom.so", "redisearch.so", "redistimeseries.so", "rejson.so"}

var vectorReadCommands = []string{"vcard", "vdim", "vemb", "vgetattr", "vinfo", "vismember", "vlinks", "vrandmember", "vrange", "vsim"}

var vectorWriteCommands = []string{"vadd", "vrem", "vsetattr"}
