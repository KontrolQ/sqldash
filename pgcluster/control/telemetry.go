package control

import (
	"regexp"
	"strings"
)

const SensitiveStatisticsPattern = `\y(password|do|call|function|procedure|sqd_control)\y|\y(alter|create|drop)\s+(role|user)\y`
const CredentialStatement = "<private statement>"

var sensitiveActivityPattern = regexp.MustCompile(`(?i)\b(password|do|call|function|procedure|sqd_control)\b|\b(alter|create|drop)\s+(role|user)\b`)

func telemetryRoleNames(roles Roles, primaryUsername string) []string {
	return []string{roles.Owner, roles.Reader, roles.Writer, primaryUsername}
}

func applicationRolePrefix(roles Roles) string {
	return strings.TrimSuffix(roles.Custodian, "custodian") + "app_"
}

func activityQuery(query string) string {
	if sensitiveActivityPattern.MatchString(query) {
		return CredentialStatement
	}
	return redactSQLValues(query)
}
