package control

type extensionSupport struct {
	SourceKey   string
	NativeUsage string
	Reason      string
	Supported   bool
}

const (
	ApplicationExtension    = "application"
	AdministrativeExtension = "administrative"
	ContribSource           = "postgresql-18"
)

var extensionCatalog = makeExtensionCatalog()

func makeExtensionCatalog() map[string]extensionSupport {
	catalog := map[string]extensionSupport{}
	for _, name := range []string{
		"autoinc", "bloom", "btree_gin", "btree_gist", "citext", "cube", "dict_int", "dict_xsyn",
		"earthdistance", "fuzzystrmatch", "hstore", "insert_username", "intagg", "intarray", "isn", "lo", "ltree",
		"moddatetime", "pg_trgm", "pgcrypto", "refint", "seg", "tablefunc", "tcn", "tsm_system_rows", "tsm_system_time", "unaccent", "uuid-ossp",
	} {
		catalog[name] = extensionSupport{SourceKey: ContribSource, NativeUsage: ApplicationExtension, Supported: true}
	}
	for _, name := range []string{
		"amcheck", "dblink", "pageinspect", "pg_buffercache", "pg_freespacemap", "pg_logicalinspect", "pg_prewarm",
		"pg_surgery", "pg_visibility", "pg_walinspect", "pgrowlocks", "pgstattuple", "sslinfo", "xml2",
	} {
		catalog[name] = extensionSupport{SourceKey: ContribSource, NativeUsage: AdministrativeExtension, Supported: true,
			Reason: "Installed functions remain private; application connections receive no diagnostic, file, network or server privileges."}
	}
	for name, source := range map[string]string{
		"vector":           "postgresql-18-pgvector",
		"postgis":          "postgresql-18-postgis-3",
		"postgis_topology": "postgresql-18-postgis-3",
		"postgis_sfcgal":   "postgresql-18-postgis-3",
		"pgrouting":        "postgresql-18-pgrouting",
		"hll":              "postgresql-18-hll",
		"ip4r":             "postgresql-18-ip4r",
		"rum":              "postgresql-18-rum",
		"pg_bigm":          "postgresql-18-pg-bigm",
		"pg_similarity":    "postgresql-18-similarity",
		"roaringbitmap":    "postgresql-18-roaringbitmap",
		"pg_uuidv7":        "postgresql-18-pg-uuidv7",
	} {
		catalog[name] = extensionSupport{SourceKey: source, NativeUsage: ApplicationExtension, Supported: true}
	}
	catalog["postgis_raster"] = extensionSupport{SourceKey: "postgresql-18-postgis-3", NativeUsage: AdministrativeExtension, Supported: true,
		Reason: "Raster file access is not granted to application connections."}
	for name, source := range map[string]string{
		"pg_cron":         "postgresql-18-cron",
		"timescaledb":     "timescaledb-2-postgresql-18",
		"citus":           "postgresql-18-citus",
		"pgaudit":         "postgresql-18-pgaudit",
		"pg_stat_kcache":  "postgresql-18-pg-stat-kcache",
		"pg_stat_monitor": "postgresql-18-pg-stat-monitor",
		"pg_qualstats":    "postgresql-18-pg-qualstats",
		"pg_tle":          "pg_tle",
		"plpython3u":      "postgresql-plpython3-18",
		"plperl":          "postgresql-plperl-18",
		"plperlu":         "postgresql-plperl-18",
		"plv8":            "postgresql-18-plv8",
		"plrust":          "plrust",
	} {
		catalog[name] = extensionSupport{SourceKey: source, NativeUsage: AdministrativeExtension,
			Reason: "This module needs additional runtime or privilege integration before activation."}
	}
	for name, source := range map[string]string{
		"postgis_tiger_geocoder": "postgresql-18-postgis-3",
		"pg_cron":                "postgresql-18-cron",
		"pg_repack":              "postgresql-18-repack",
		"pg_partman":             "postgresql-18-partman",
		"pg_squeeze":             "postgresql-18-squeeze",
		"pg_dirtyread":           "postgresql-18-dirtyread",
		"pg_stat_kcache":         "postgresql-18-pg-stat-kcache",
		"pg_qualstats":           "postgresql-18-pg-qualstats",
		"pg_hint_plan":           "postgresql-18-pg-hint-plan",
		"pgaudit":                "postgresql-18-pgaudit",
		"timescaledb":            "timescaledb-2-postgresql-18",
	} {
		catalog[name] = extensionSupport{SourceKey: source, NativeUsage: AdministrativeExtension, Supported: true,
			Reason: "Application roles receive no administrative functions, job queues, connection mappings or file access. Runtime prerequisites must already be configured."}
	}
	catalog["semver"] = extensionSupport{SourceKey: "postgresql-18-semver", NativeUsage: ApplicationExtension, Supported: true}
	catalog["auto_explain"] = extensionSupport{SourceKey: ContribSource, NativeUsage: AdministrativeExtension,
		Reason: "This server module requires verified per-database activation and SQL-log privacy integration."}
	catalog["pg_stat_statements"] = extensionSupport{SourceKey: ContribSource, NativeUsage: AdministrativeExtension, Supported: true,
		Reason: "Statistics are collected through private database-scoped management queries."}
	return catalog
}

func supportedExtension(name string) bool {
	return extensionCatalog[name].Supported
}
