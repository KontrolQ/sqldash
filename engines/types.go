package engines

import (
	"context"
	"sqldash/models"
	"sqldash/sqld"
	"time"
)

type Mode string

const (
	ReadOnly  Mode = "read-only"
	ReadWrite Mode = "read-write"
	Objects   Mode = "objects"
)

type Statement = sqld.Statement
type Result = sqld.Result
type StatisticsRecord = sqld.StatisticsRecord
type ConfigurationRecord = sqld.ConfigurationRecord

type Features struct {
	Schemas           bool
	TypedValues       bool
	CompositeKeys     bool
	SQLiteFiles       bool
	CustomArchives    bool
	HistoricalFork    bool
	Attach            bool
	ReadWriteBlocks   bool
	NativeConnections bool
	Activity          bool
	Extensions        bool
	MeasuredQueries   bool
}

type Backend interface {
	Features() Features
	Create(context.Context, *models.Database) error
	Delete(context.Context, *models.Database) error
	Run(context.Context, *models.Database, Mode, ...Statement) ([]*Result, error)
	Statistics(context.Context, *models.Database) (*StatisticsRecord, error)
	Configuration(context.Context, *models.Database) (*ConfigurationRecord, error)
	SetConfiguration(context.Context, *models.Database, ConfigurationRecord) error
	Fork(context.Context, *models.Database, *models.Database, *time.Time) error
}

type Relation struct {
	Schema string
	Name   string
	Kind   string
	Rows   int64
}

type RelationColumns struct {
	Schema  string
	Name    string
	Kind    string
	Columns []string
}

type Column struct {
	Name             string
	Kind             string
	NotNull          bool
	PrimaryKey       bool
	PrimaryPosition  int
	Default          string
	Generated        bool
	Identity         bool
	References       string
	ReferenceSchema  string
	ReferenceColumns []string
}

type Index struct {
	Name       string
	Unique     bool
	Primary    bool
	Columns    []string
	Definition string
}

type Constraint struct {
	Name             string
	Kind             string
	Columns          []string
	ReferenceSchema  string
	ReferenceTable   string
	ReferenceColumns []string
	Definition       string
}

type Definition struct {
	Relation    Relation
	Columns     []Column
	Indexes     []Index
	Constraints []Constraint
	Triggers    []NamedDefinition
	Functions   []NamedDefinition
	Sequences   []NamedDefinition
	SQL         string
}

type NamedDefinition struct {
	Name       string
	Definition string
}

type Catalog interface {
	Relations(context.Context, *models.Database, string) ([]Relation, error)
	Columns(context.Context, *models.Database, string, string) ([]Column, error)
	Structure(context.Context, *models.Database, string, string) (*Definition, error)
}
