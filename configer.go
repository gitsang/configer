package configer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/gitsang/defaults"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Options struct {
	template any

	envBind   bool
	envPrefix string
	envDelim  string

	command    *cobra.Command
	flagBind   bool
	flagPrefix string
	flagDelim  string
}

type Configer struct {
	Options
	viper       *viper.Viper
	conflicts   []string
}

type OptionFunc func(configer *Configer)

func WithTemplate(template any) OptionFunc {
	return func(c *Configer) {
		c.template = template
	}
}

func WithEnvBind(optfs ...OptionFunc) OptionFunc {
	return func(c *Configer) {
		c.envBind = true
		for _, apply := range optfs {
			apply(c)
		}
	}
}

func WithEnvPrefix(prefix string) OptionFunc {
	return func(c *Configer) {
		c.envPrefix = prefix
		c.viper.SetEnvPrefix(prefix)
	}
}

func WithEnvDelim(delim string) OptionFunc {
	return func(c *Configer) {
		c.envDelim = delim
		c.viper.SetEnvKeyReplacer(strings.NewReplacer(".", delim))
	}
}

func WithFlagBind(optfs ...OptionFunc) OptionFunc {
	return func(c *Configer) {
		c.flagBind = true
		for _, apply := range optfs {
			apply(c)
		}
	}
}

func WithCommand(command *cobra.Command) OptionFunc {
	return func(c *Configer) {
		c.command = command
	}
}

func WithFlagPrefix(prefix string) OptionFunc {
	return func(c *Configer) {
		c.flagPrefix = prefix
	}
}

func WithFlagDelim(delim string) OptionFunc {
	return func(c *Configer) {
		c.flagDelim = delim
	}
}

// getFieldName returns the effective field name for a struct field.
// Priority: mapstructure tag > yaml tag > snake_case(field name)
func getFieldName(f reflect.StructField) string {
	if tag := f.Tag.Get("mapstructure"); tag != "" {
		return tag
	}
	if tag := f.Tag.Get("yaml"); tag != "" {
		return tag
	}
	return strings.ToLower(f.Name)
}

// checkFieldConflict checks if a field name contains the delimiter character
// and records a warning if so.
func (p *Configer) checkFieldConflict(fieldName string, namespaces []string) {
	delim := p.envDelim
	if delim == "" {
		return
	}
	if strings.Contains(fieldName, delim) {
		path := strings.Join(namespaces[1:], ".")
		warning := fmt.Sprintf(
			"WARNING: field %q contains delimiter %q, environment variable binding may be ambiguous. "+
				"Consider using a different delimiter (e.g. \"__\") or renaming the field via mapstructure tag.",
			path, delim,
		)
		p.conflicts = append(p.conflicts, warning)
	}
}

// Warnings returns all collected warnings about potential configuration conflicts.
func (c *Configer) Warnings() []string {
	return c.conflicts
}

// getDelim returns the environment variable delimiter, defaulting to ".".
func (p *Configer) getDelim() string {
	if p.envDelim != "" {
		return p.envDelim
	}
	return "."
}

// buildEnvKey builds an environment variable key from namespaces.
// e.g., ["PREFIX", "server", "labels"] with delim "_" -> "PREFIX_SERVER_LABELS"
func (p *Configer) buildEnvKey(namespaces []string) string {
	return strings.ToUpper(strings.Join(namespaces, p.getDelim()))
}

// buildViperKey builds a viper key from namespaces (excluding prefix).
// e.g., ["PREFIX", "server", "labels"] -> "server.labels"
func buildViperKey(namespaces []string) string {
	return strings.Join(namespaces[1:], ".")
}

// preventAutoOverride binds a viper key to a non-existent env var
// to prevent AutomaticEnv from overriding the value.
func (p *Configer) preventAutoOverride(viperKey string) {
	p.viper.BindEnv(viperKey, "NONEXISTENT_ENV_VAR_"+viperKey)
}

func (p *Configer) parseFlags(i interface{}, parents []string) {
	r := reflect.TypeOf(i)

	for r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		fieldName := getFieldName(f)
		namespaces := append(parents, fieldName)

		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		if ft.Kind() == reflect.Struct {
			t := reflect.New(ft).Elem().Interface()
			p.parseFlags(t, namespaces)
			continue
		}

		if ft.Kind() == reflect.Map || ft.Kind() == reflect.Slice {
			continue
		}

		flagName := strings.TrimPrefix(strings.Join(namespaces, p.flagDelim), p.flagDelim)
		if flagTag := f.Tag.Get("flag"); flagTag != "" {
			flagName = flagTag
		}
		p.command.Flags().String(flagName, f.Tag.Get("default"), f.Tag.Get("usage"))

		viperKey := buildViperKey(namespaces)
		err := p.viper.BindPFlag(viperKey, p.command.Flags().Lookup(flagName))
		if err != nil {
			continue
		}
	}
}

func (p *Configer) parseEnv(i interface{}, parents []string) {
	t := reflect.TypeOf(i)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fieldName := getFieldName(f)
		namespaces := append(parents, fieldName)

		p.checkFieldConflict(fieldName, namespaces)

		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		// Handle map[string]Struct types (special case with nested fields)
		if ft.Kind() == reflect.Map && ft.Key().Kind() == reflect.String {
			elemType := ft.Elem()
			for elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct {
				p.parseMapEnv(f, namespaces, elemType)
				continue
			}
		}

		// Handle slice and map[string]BasicType via JSON
		if ft.Kind() == reflect.Slice || (ft.Kind() == reflect.Map && ft.Key().Kind() == reflect.String) {
			p.parseJSONEnv(f, namespaces)
			continue
		}

		if ft.Kind() == reflect.Struct {
			fi := reflect.New(ft).Elem().Interface()
			p.parseEnv(fi, namespaces)
			continue
		}

		viperKey := buildViperKey(namespaces)
		if envTag := f.Tag.Get("env"); envTag != "" {
			if err := p.viper.BindEnv(viperKey, envTag); err != nil {
				continue
			}
		}
	}
}

// parseJSONEnv handles slice and map[string]BasicType fields via JSON.
// It supports:
//   - Direct JSON:  PREFIX_FIELDNAME='["value1","value2"]' or '{"key":"value"}'
//   - Individual keys for map: PREFIX_FIELDNAME_KEY=value (only for map types)
func (p *Configer) parseJSONEnv(field reflect.StructField, namespaces []string) {
	ft := field.Type
	for ft.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	isMap := ft.Kind() == reflect.Map

	delim := p.getDelim()
	prefix := p.buildEnvKey(namespaces) + delim
	directKey := strings.TrimSuffix(prefix, delim)

	var result interface{}
	mapEntries := make(map[string]interface{})

	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envKey, envValue := parts[0], parts[1]

		// Direct JSON value
		if envKey == directKey {
			if err := json.Unmarshal([]byte(envValue), &result); err == nil {
				if !isMap {
					viperKey := buildViperKey(namespaces)
					p.viper.Set(viperKey, result)
					p.preventAutoOverride(viperKey)
					return
				}
				// For map, merge into mapEntries
				if m, ok := result.(map[string]interface{}); ok {
					for k, v := range m {
						mapEntries[strings.ToLower(k)] = v
					}
				}
			}
			continue
		}

		// Individual keys (map only)
		if isMap && strings.HasPrefix(envKey, prefix) {
			remaining := strings.TrimPrefix(envKey, prefix)
			mapKey := strings.ToLower(remaining)
			mapEntries[mapKey] = envValue
		}
	}

	if len(mapEntries) > 0 {
		viperKey := buildViperKey(namespaces)
		p.viper.Set(viperKey, mapEntries)
		p.preventAutoOverride(viperKey)
	}
}

// parseMapEnv handles map[string]Struct types with nested fields.
// It supports two patterns:
//   - JSON string:     PREFIX_FIELDNAME='{"key":{"field":"value"}}'
//   - Individual keys: PREFIX_FIELDNAME_MAPKEY_FIELDNAME=value
func (p *Configer) parseMapEnv(field reflect.StructField, namespaces []string, elemType reflect.Type) {
	delim := p.getDelim()
	prefix := p.buildEnvKey(namespaces) + delim
	directKey := strings.TrimSuffix(prefix, delim)

	fieldEnvMap := make(map[string]string)
	for i := 0; i < elemType.NumField(); i++ {
		structField := elemType.Field(i)
		subFieldName := getFieldName(structField)
		fieldEnvMap[strings.ToLower(subFieldName)] = subFieldName
	}

	mapEntries := make(map[string]map[string]interface{})

	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envKey, envValue := parts[0], parts[1]

		// Try JSON format first
		if envKey == directKey {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(envValue), &parsed); err == nil {
				for mapKey, mapValue := range parsed {
					if m, ok := mapValue.(map[string]interface{}); ok {
						mapEntries[strings.ToLower(mapKey)] = m
					}
				}
			}
			continue
		}

		// Individual keys format
		if !strings.HasPrefix(envKey, prefix) {
			continue
		}

		remaining := strings.TrimPrefix(envKey, prefix)
		remainingParts := strings.Split(remaining, delim)

		if len(remainingParts) < 2 {
			continue
		}

		mapKey := strings.ToLower(remainingParts[0])
		subFieldKey := strings.ToLower(strings.Join(remainingParts[1:], "_"))

		fieldPath := subFieldKey
		if name, exists := fieldEnvMap[subFieldKey]; exists {
			fieldPath = name
		}

		if mapEntries[mapKey] == nil {
			mapEntries[mapKey] = make(map[string]interface{})
		}

		p.setNestedValue(mapEntries[mapKey], fieldPath, envValue)
	}

	if len(mapEntries) > 0 {
		viperKey := buildViperKey(namespaces)
		p.viper.Set(viperKey, mapEntries)
		p.preventAutoOverride(viperKey)
	}
}

func (p *Configer) setNestedValue(target map[string]interface{}, path string, value string) {
	parts := strings.Split(path, ".")
	current := target

	// Navigate to the parent of the final key
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		if current[key] == nil {
			current[key] = make(map[string]interface{})
		}
		if nested, ok := current[key].(map[string]interface{}); ok {
			current = nested
		} else {
			// If the current value is not a map, we can't navigate further
			return
		}
	}

	// Set the final value
	finalKey := parts[len(parts)-1]
	current[finalKey] = value
}

func New(optfs ...OptionFunc) *Configer {
	c := &Configer{
		viper: viper.NewWithOptions(),
	}
	for _, apply := range optfs {
		apply(c)
	}

	if c.envBind {
		c.viper.AutomaticEnv()
		c.parseEnv(c.template, []string{c.envPrefix})
	}

	if c.flagBind {
		c.parseFlags(c.template, []string{c.flagPrefix})
	}

	for _, w := range c.conflicts {
		log.Println(w)
	}

	return c
}

func (c *Configer) Load(config any, files ...string) error {
	for _, file := range files {
		c.viper.SetConfigFile(file)
		if err := c.viper.ReadInConfig(); err != nil {
			fmt.Printf("failed to read file %s: %s\n", file, err.Error())
			continue
		}
	}

	if err := c.viper.Unmarshal(config); err != nil {
		return err
	}
	if err := defaults.Set(config); err != nil {
		return err
	}
	return nil
}

// Debug method to help troubleshoot configuration issues
func (c *Configer) Debug() {
	fmt.Println("=== Debug: All Viper Keys and Values ===")
	for _, key := range c.viper.AllKeys() {
		fmt.Printf("Key: %s, Value: %v, Type: %T\n", key, c.viper.Get(key), c.viper.Get(key))
	}

	// Also print all environment variables that match our prefix
	fmt.Println("\n=== Debug: Environment Variables ===")
	envVars := os.Environ()
	for _, envVar := range envVars {
		if strings.HasPrefix(envVar, c.envPrefix) {
			fmt.Println(envVar)
		}
	}
}

func (c *Configer) Store(config any, file string) error {
	configYamlBytes, _ := yaml.Marshal(config)

	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(configYamlBytes)
	if err != nil {
		return err
	}

	return nil
}
