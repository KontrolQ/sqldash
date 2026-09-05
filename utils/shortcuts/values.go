package shortcuts

import (
	"errors"
	"maps"
	"reflect"
	"strings"

	"sqldash/utils/collections"

	"github.com/gofiber/fiber/v3"
)

func mergeContextValues(context fiber.Ctx, target collections.Record[string, any]) {
	context.RequestCtx().VisitUserValuesAll(func(key any, value any) {
		switch typedKey := key.(type) {
		case string:
			if typedKey != "" {
				target[typedKey] = value
			}
		case []byte:
			if len(typedKey) > 0 {
				target[string(typedKey)] = value
			}
		}
	})
}

func mergeBindData(target collections.Record[string, any], data any) error {
	normalized, normalizeError := normalizeToMap(data)
	if normalizeError != nil {
		return normalizeError
	}

	maps.Copy(target, normalized)

	return nil
}

func normalizeToMap(data any) (collections.Record[string, any], error) {
	switch typedData := data.(type) {
	case collections.Record[string, any]:
		return typedData, nil
	default:
		return convertStructToMap(data)
	}
}

func convertStructToMap(data any) (collections.Record[string, any], error) {
	structValue := reflect.ValueOf(data)

	switch structValue.Kind() {
	case reflect.Pointer:
		structValue = structValue.Elem()
	}

	switch structValue.Kind() {
	case reflect.Struct:
		return extractStructFields(structValue), nil
	default:
		return nil, errors.New(UnsupportedBindType)
	}
}

func extractStructFields(structValue reflect.Value) collections.Record[string, any] {
	structType := structValue.Type()
	fieldMap := make(collections.Record[string, any], structType.NumField())

	for fieldIndex := range structType.NumField() {
		fieldDescriptor := structType.Field(fieldIndex)

		if !fieldDescriptor.IsExported() {
			continue
		}

		fieldValue := structValue.Field(fieldIndex).Interface()
		fieldMap[fieldDescriptor.Name] = fieldValue

		fieldKey := resolveFieldKey(fieldDescriptor)
		if fieldKey != fieldDescriptor.Name {
			fieldMap[fieldKey] = fieldValue
		}
	}

	return fieldMap
}

func resolveFieldKey(fieldDescriptor reflect.StructField) string {
	jsonTag := fieldDescriptor.Tag.Get("json")

	switch jsonTag {
	case "", "-":
		return fieldDescriptor.Name
	default:
		return extractTagName(jsonTag, fieldDescriptor.Name)
	}
}

func extractTagName(jsonTag string, fallbackName string) string {
	separatorIndex := strings.IndexByte(jsonTag, ',')

	switch {
	case separatorIndex < 0:
		return jsonTag
	case separatorIndex > 0:
		return jsonTag[:separatorIndex]
	default:
		return fallbackName
	}
}
