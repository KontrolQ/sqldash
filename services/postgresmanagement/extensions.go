package postgresmanagement

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetExtensions(ctx context.Context, name string) (*ExtensionsContext, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, permissions.PostgresExtensionsManage)
	if err != nil {
		return nil, err
	}
	bounded, cancel := context.WithTimeout(fresh, 10*time.Second)
	defer cancel()
	extensions, readError := control.Extensions(bounded, record)
	if readError != nil {
		return nil, shortcuts.ServiceError(http.StatusServiceUnavailable, ExtensionsUnavailable)
	}
	shown := &ExtensionsContext{Title: "Extensions", Name: record.Name}
	for _, extension := range extensions {
		if !extension.Available || !extension.Supported && !extension.Protected {
			continue
		}
		label := extension.Label
		if label == "" {
			label = extension.Name
		}
		shown.Rows = append(shown.Rows, ExtensionView{Label: label, Name: extension.Name, DefaultVersion: extension.DefaultVersion,
			InstalledVersion: extension.InstalledVersion, Description: extension.Comment, Supported: extension.Supported, Protected: extension.Protected,
			Available: extension.Available, Enabled: extension.Enabled, CanChange: extension.Supported && !extension.Protected && extension.Available,
			Source: extensionSourceLabel(extension.SourceKey), NativeUsage: extensionUsageLabel(extension.NativeUsage), Reason: extension.Reason, Type: extension.Type})
	}
	return shown, nil
}

func ChangeExtension(ctx context.Context, name string, extensionName string, install bool) (*models.Database, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, permissions.PostgresExtensionsManage)
	if err != nil {
		return nil, err
	}
	bounded, cancel := context.WithTimeout(fresh, 15*time.Second)
	defer cancel()
	extensions, readError := control.Extensions(bounded, record)
	if readError != nil {
		return nil, shortcuts.ServiceError(http.StatusServiceUnavailable, ExtensionsUnavailable)
	}
	permitted := false
	for _, extension := range extensions {
		if extension.Name == extensionName && extension.Supported && !extension.Protected && extension.Available && extension.Enabled != install {
			permitted = true
			break
		}
	}
	if !permitted {
		return nil, shortcuts.ServiceError(http.StatusBadRequest, ExtensionRefused)
	}
	if err := control.ManageExtension(bounded, record, extensionName, install); err != nil {
		var dependency *control.ExtensionDependencyError
		if errors.As(err, &dependency) && slices.Contains(control.SupportedExtensions(), dependency.Name) {
			return nil, shortcuts.ServiceError(http.StatusConflict, fmt.Sprintf(ExtensionDependencyRequired, dependency.Name))
		}
		return nil, shortcuts.ServiceError(http.StatusConflict, ExtensionFailed)
	}
	return record, nil
}

func extensionSourceLabel(sourceKey string) string {
	switch sourceKey {
	case "contrib", control.ContribSource:
		return "PostgreSQL"
	case "core":
		return "PostgreSQL"
	case "bundled":
		return "Bundled"
	case "thirdparty":
		return "Bundled extension"
	default:
		return sourceKey
	}
}

func extensionUsageLabel(usage string) string {
	switch usage {
	case control.ApplicationExtension:
		return "Application use"
	case control.AdministrativeExtension:
		return "Administration only"
	default:
		return usage
	}
}
