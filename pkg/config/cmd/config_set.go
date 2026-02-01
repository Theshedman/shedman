package cmd

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	appconfig "github.com/theshedman/shedman/internal/config"
)

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.LoadDefault()
			if err != nil {
				return err
			}

			val, err := getConfigValue(cfg, args[0])
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.LoadDefault()
			if err != nil {
				return err
			}

			if err := setConfigValue(cfg, args[0], args[1]); err != nil {
				return err
			}

			if err := appconfig.Save(appconfig.DefaultConfigPath(), cfg); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s updated.\n", args[0])
			return nil
		},
	}
}

func setConfigValue(cfg *appconfig.Config, key, value string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if key == "" {
		return fmt.Errorf("config key is required")
	}
	segments := strings.Split(key, ".")
	root := reflect.ValueOf(cfg)
	if root.Kind() != reflect.Pointer {
		return fmt.Errorf("config root must be a pointer")
	}
	return setValueByPath(root.Elem(), segments, value)
}

func getConfigValue(cfg *appconfig.Config, key string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}
	if key == "" {
		return "", fmt.Errorf("config key is required")
	}
	segments := strings.Split(key, ".")
	root := reflect.ValueOf(cfg)
	if root.Kind() != reflect.Pointer {
		return "", fmt.Errorf("config root must be a pointer")
	}
	val, err := getValueByPath(root.Elem(), segments)
	if err != nil {
		return "", err
	}
	return formatConfigValue(val)
}

func setValueByPath(current reflect.Value, segments []string, value string) error {
	if len(segments) == 0 {
		return fmt.Errorf("config key is required")
	}
	if current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	if current.Kind() != reflect.Struct {
		return fmt.Errorf("config path is not a struct")
	}

	field, ok := findConfigField(current, segments[0])
	if !ok {
		return fmt.Errorf("unknown config key: %s", segments[0])
	}

	if len(segments) == 1 {
		return assignConfigValue(field, value)
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}
	if field.Kind() != reflect.Struct {
		return fmt.Errorf("config key %s is not a nested section", segments[0])
	}

	return setValueByPath(field, segments[1:], value)
}

func getValueByPath(current reflect.Value, segments []string) (reflect.Value, error) {
	if len(segments) == 0 {
		return reflect.Value{}, fmt.Errorf("config key is required")
	}
	if current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	if current.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("config path is not a struct")
	}

	field, ok := findConfigField(current, segments[0])
	if !ok {
		return reflect.Value{}, fmt.Errorf("unknown config key: %s", segments[0])
	}

	if len(segments) == 1 {
		return field, nil
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return reflect.Value{}, fmt.Errorf("config key %s is not set", segments[0])
		}
		field = field.Elem()
	}
	if field.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("config key %s is not a nested section", segments[0])
	}
	return getValueByPath(field, segments[1:])
}

func assignConfigValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return fmt.Errorf("config value is not settable")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
		return nil
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid bool value: %s", value)
		}
		field.SetBool(parsed)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid int value: %s", value)
		}
		field.SetInt(int64(parsed))
		return nil
	case reflect.Slice:
		return assignSliceValue(field, value)
	default:
		return fmt.Errorf("unsupported config value type: %s", field.Kind())
	}
}

func assignSliceValue(field reflect.Value, value string) error {
	if field.Type().Elem().Kind() == reflect.String {
		var items []string
		for _, part := range strings.Split(value, ",") {
			item := strings.TrimSpace(part)
			if item == "" {
				continue
			}
			items = append(items, item)
		}
		field.Set(reflect.ValueOf(items))
		return nil
	}

	if field.Type().Elem().Kind() == reflect.Int {
		var items []int
		for _, part := range strings.Split(value, ",") {
			item := strings.TrimSpace(part)
			if item == "" {
				continue
			}
			parsed, err := strconv.Atoi(item)
			if err != nil {
				return fmt.Errorf("invalid int value: %s", item)
			}
			items = append(items, parsed)
		}
		field.Set(reflect.ValueOf(items))
		return nil
	}

	return fmt.Errorf("unsupported slice element type: %s", field.Type().Elem().Kind())
}

func formatConfigValue(val reflect.Value) (string, error) {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return "", nil
		}
		val = val.Elem()
	}
	switch val.Kind() {
	case reflect.String:
		return val.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil
	case reflect.Slice:
		if val.Type().Elem().Kind() != reflect.String {
			return "", fmt.Errorf("unsupported slice element type: %s", val.Type().Elem().Kind())
		}
		var items []string
		for i := 0; i < val.Len(); i++ {
			items = append(items, val.Index(i).String())
		}
		return strings.Join(items, ","), nil
	default:
		return "", fmt.Errorf("unsupported config value type: %s", val.Kind())
	}
}

func findConfigField(current reflect.Value, segment string) (reflect.Value, bool) {
	segment = normalizeConfigSegment(segment)
	t := current.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := strings.Split(field.Tag.Get("toml"), ",")[0]
		tag = normalizeConfigSegment(tag)
		name := normalizeConfigSegment(field.Name)
		if segment == tag || segment == name {
			return current.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func normalizeConfigSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	segment = strings.ToLower(segment)
	segment = strings.ReplaceAll(segment, "-", "_")
	return segment
}
