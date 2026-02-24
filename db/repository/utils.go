package repository

// import (
	// "fmt"
	// "reflect"
	// "regexp"
	// "strings"
	// "unicode"
// )

// structToMap преобразует структуру в map для условий
// func structToMap(obj interface{}) (map[string]interface{}, error) {
// 	result := make(map[string]interface{})
// 	v := reflect.ValueOf(obj)
// 	// Если передан указатель, получаем значение
// 	if v.Kind() == reflect.Ptr {
// 		v = v.Elem()
// 	}
// 	// Проверяем, что это структура
// 	if v.Kind() != reflect.Struct {
// 		return nil, fmt.Errorf("expected struct, got %T", obj)
// 	}
// 	t := v.Type()
// 	for i := 0; i < v.NumField(); i++ {
// 		field := t.Field(i)
// 		fieldValue := v.Field(i)
// 		// Пропускаем нулевые значения
// 		if !fieldValue.IsZero() {
// 			columnName := getColumnName(field)
// 			result[columnName] = fieldValue.Interface()
// 		}
// 	}
// 	return result, nil
// }

// getFieldIndexByName ищет поле по имени
// func getFieldIndexByName(t reflect.Type, name string) (int, bool) {
// 	for i := 0; i < t.NumField(); i++ {
// 		field := t.Field(i)
// 		if strings.EqualFold(field.Name, name) {
// 			return i, true
// 		}
// 	}
// 	return -1, false
// }

// func getColumnName(field reflect.StructField) string {
// 	// Сначала проверяем тег gorm
// 	gormTag := field.Tag.Get("gorm")
// 	if gormTag != "" {
// 		if strings.Contains(gormTag, "column:") {
// 			re := regexp.MustCompile(`column:([^;]+)`)
// 			matches := re.FindStringSubmatch(gormTag)
// 			if len(matches) > 1 {
// 				return strings.TrimSpace(matches[1])
// 			}
// 		}
// 	}
// 	// Затем проверяем тег json
// 	jsonTag := field.Tag.Get("json")
// 	if jsonTag != "" && jsonTag != "-" {
// 		parts := strings.Split(jsonTag, ",")
// 		if parts[0] != "" {
// 			return parts[0]
// 		}
// 	}
// 	// Если нет тегов, преобразуем в snake_case
// 	return toSnakeCase(field.Name)
// }

// func toSnakeCase(s string) string {
// 	var result strings.Builder
// 	for i, r := range s {
// 		if unicode.IsUpper(r) {
// 			if i > 0 {
// 				result.WriteByte('_')
// 			}
// 			result.WriteRune(unicode.ToLower(r))
// 		} else {
// 			result.WriteRune(r)
// 		}
// 	}
// 	return result.String()
// }
