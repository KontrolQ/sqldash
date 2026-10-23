package database

import (
	"fmt"
	"strings"

	"sqldash/models"

	"gorm.io/gorm"
)

type oidColumnAlias struct {
	Model       any
	Previous    string
	Current     string
	Table       string
	CurrentSeen bool
}

type oidColumnIndex struct {
	Name string
	SQL  string
}

func migrateOIDColumns() error {
	return DB.Transaction(func(transaction *gorm.DB) error {
		aliases := []oidColumnAlias{
			{Model: &models.Database{}, Previous: "backend_o_id", Current: "backend_oid"},
			{Model: &models.PostgresStatistics{}, Previous: "database_o_id", Current: "database_oid"},
			{Model: &models.PostgresStatementStatistics{}, Previous: "database_o_id", Current: "database_oid"},
			{Model: &models.PostgresStatementStatistics{}, Previous: "native_o_id", Current: "native_oid"},
		}
		var pending []oidColumnAlias
		for _, alias := range aliases {
			statement := &gorm.Statement{DB: transaction}
			if err := statement.Parse(alias.Model); err != nil {
				return err
			}
			alias.Table = statement.Schema.Table
			var columns []string
			if err := transaction.Raw("SELECT name FROM pragma_table_xinfo(?)", alias.Table).Scan(&columns).Error; err != nil {
				return err
			}
			previousSeen := false
			for _, column := range columns {
				previousSeen = previousSeen || strings.EqualFold(column, alias.Previous)
				alias.CurrentSeen = alias.CurrentSeen || strings.EqualFold(column, alias.Current)
			}
			if !previousSeen {
				continue
			}
			if alias.CurrentSeen {
				var conflicts int64
				query := fmt.Sprintf("SELECT count(*) FROM %s WHERE COALESCE(%s, 0) <> 0 AND COALESCE(%s, 0) <> 0 AND %s <> %s",
					quoteOIDIdentifier(alias.Table), quoteOIDIdentifier(alias.Previous), quoteOIDIdentifier(alias.Current),
					quoteOIDIdentifier(alias.Previous), quoteOIDIdentifier(alias.Current))
				if err := transaction.Raw(query).Scan(&conflicts).Error; err != nil {
					return err
				}
				if conflicts != 0 {
					return fmt.Errorf("cannot repair %s.%s: %d rows disagree with %s", alias.Table, alias.Previous, conflicts, alias.Current)
				}
			}
			pending = append(pending, alias)
		}
		for _, alias := range pending {
			if err := migrateOIDColumn(transaction, alias); err != nil {
				return fmt.Errorf("repair %s.%s: %w", alias.Table, alias.Previous, err)
			}
		}
		return nil
	})
}

func migrateOIDColumn(transaction *gorm.DB, alias oidColumnAlias) error {
	table := quoteOIDIdentifier(alias.Table)
	previous := quoteOIDIdentifier(alias.Previous)
	current := quoteOIDIdentifier(alias.Current)
	if !alias.CurrentSeen {
		return transaction.Exec(fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", table, previous, current)).Error
	}
	query := fmt.Sprintf("UPDATE %s SET %s = %s WHERE (%s IS NULL OR %s = 0) AND %s IS NOT NULL",
		table, current, previous, current, current, previous)
	if err := transaction.Exec(query).Error; err != nil {
		return err
	}
	var indexes []oidColumnIndex
	if err := transaction.Raw("SELECT name, sql FROM sqlite_schema WHERE type = 'index' AND tbl_name = ? AND sql IS NOT NULL", alias.Table).Scan(&indexes).Error; err != nil {
		return err
	}
	var replaced []oidColumnIndex
	for _, index := range indexes {
		rewritten, err := rewriteOIDIndex(index.SQL, alias.Previous, alias.Current)
		if err != nil {
			return err
		}
		if rewritten == index.SQL {
			continue
		}
		if err := transaction.Exec("DROP INDEX " + quoteOIDIdentifier(index.Name)).Error; err != nil {
			return err
		}
		index.SQL = rewritten
		replaced = append(replaced, index)
	}
	if err := transaction.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, previous)).Error; err != nil {
		return err
	}
	for _, index := range replaced {
		if err := transaction.Exec(index.SQL).Error; err != nil {
			return err
		}
	}
	return nil
}

func quoteOIDIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func rewriteOIDIndex(statement, previous, current string) (string, error) {
	var rewritten strings.Builder
	inBody := false
	for position := 0; position < len(statement); {
		start := position
		character := statement[position]
		switch {
		case strings.HasPrefix(statement[position:], "--"):
			for position < len(statement) && statement[position] != '\n' {
				position++
			}
		case strings.HasPrefix(statement[position:], "/*"):
			end := strings.Index(statement[position+2:], "*/")
			if end < 0 {
				return "", fmt.Errorf("unterminated comment in OID index definition")
			}
			position += end + 4
		case character == '\'' || character == '"' || character == '`' || character == '[':
			closing := character
			if character == '[' {
				closing = ']'
			}
			position++
			var identifier strings.Builder
			closed := false
			for position < len(statement) {
				if statement[position] == closing {
					position++
					if position < len(statement) && statement[position] == closing {
						identifier.WriteByte(closing)
						position++
						continue
					}
					closed = true
					break
				}
				identifier.WriteByte(statement[position])
				position++
			}
			if !closed {
				return "", fmt.Errorf("unterminated quote in OID index definition")
			}
			if inBody && character != '\'' && strings.EqualFold(identifier.String(), previous) {
				rewritten.WriteString(quoteOIDIdentifier(current))
				continue
			}
		case oidIdentifierCharacter(character):
			for position < len(statement) && oidIdentifierCharacter(statement[position]) {
				position++
			}
			if inBody && strings.EqualFold(statement[start:position], previous) {
				rewritten.WriteString(quoteOIDIdentifier(current))
				continue
			}
		default:
			inBody = inBody || character == '('
			position++
		}
		rewritten.WriteString(statement[start:position])
	}
	return rewritten.String(), nil
}

func oidIdentifierCharacter(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' || character == '_' || character == '$' || character >= 128
}
