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

func (p *Configer) parseFlags(i interface{}, parents []string) {
	r := reflect.TypeOf(i)

	for r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		namespaces := append(parents, strings.ToLower(f.Name))

		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		if ft.Kind() == reflect.Struct {
			t := reflect.New(ft).Elem().Interface()
			p.parseFlags(t, namespaces)
			continue
		}

		if ft.Kind() == reflect.Map {
			continue
		}

		// trim delim prefix to avoid empty prefix
		flagName := strings.TrimPrefix(strings.Join(namespaces, p.flagDelim), p.flagDelim)
		if flagTag := f.Tag.Get("flag"); flagTag != "" {
			flagName = flagTag
		}
		p.command.Flags().String(flagName, f.Tag.Get("default"), f.Tag.Get("usage"))

		// viperKey use dot to addressing (mapstructure default) and should exclude the prefix
		viperKey := strings.Join(namespaces[1:], ".")
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

		// Handle slice types
		if ft.Kind() == reflect.Slice {
			p.parseSliceEnv(f, namespaces)
			continue
		}

		// Handle map[string]T types
		if ft.Kind() == reflect.Map && ft.Key().Kind() == reflect.String {
			elemType := ft.Elem()
			for elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct {
				p.parseMapEnv(f, namespaces, elemType)
				continue
			}
			// Handle map[string]BasicType (string, int, bool, etc.)
			p.parseBasicMapEnv(f, namespaces)
			continue
		}

		if ft.Kind() == reflect.Struct {
			fi := reflect.New(ft).Elem().Interface()
			p.parseEnv(fi, namespaces)
			continue
		}

		viperKey := strings.Join(namespaces[1:], ".")
		if envTag := f.Tag.Get("env"); envTag != "" {
			if err := p.viper.BindEnv(viperKey, envTag); err != nil {
				continue
			}
		}
	}
}

// parseMapEnv handles parsing environment variables for map[string]Struct types
// It looks for environment variables with pattern: PREFIX_MAPKEY_FIELDNAME
// and converts them to map entries like map[MAPKEY]{FIELDNAME: value}
func (p *Configer) parseMapEnv(field reflect.StructField, namespaces []string, elemType reflect.Type) {
	envVars := os.Environ()

	delim := p.envDelim
	if delim == "" {
		delim = "."
	}

	fieldName := namespaces[len(namespaces)-1]

	// Build prefix with the determined field name
	// e.g., if namespaces is ["CONFIGER", "logs"], prefix will be "CONFIGER_LOGS_"
	prefixNamespaces := make([]string, len(namespaces)-1)
	copy(prefixNamespaces, namespaces[:len(namespaces)-1])
	prefixNamespaces = append(prefixNamespaces, strings.ToUpper(fieldName))
	prefix := strings.ToUpper(strings.Join(prefixNamespaces, delim)) + delim

	// Create a map to store env tag to field name mappings
	fieldEnvMap := make(map[string]string)
	for i := 0; i < elemType.NumField(); i++ {
		structField := elemType.Field(i)
		fieldName := strings.ToLower(structField.Name)
		envTag := structField.Tag.Get("env")
		if envTag != "" {
			fieldEnvMap[strings.ToLower(envTag)] = fieldName
		} else {
			fieldEnvMap[fieldName] = fieldName
		}
	}

	// Track found map keys and their field mappings
	mapEntries := make(map[string]map[string]interface{})

	// Parse environment variables to find matches
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envKey := parts[0]
		envValue := parts[1]

		// Check if this env var matches our prefix pattern
		if !strings.HasPrefix(envKey, prefix) {
			continue
		}

		// Remove prefix to get the remaining part: MAPKEY_FIELDNAME
		remaining := strings.TrimPrefix(envKey, prefix)
		remainingParts := strings.Split(remaining, delim)

		if len(remainingParts) < 2 {
			continue
		}

		// First part is the map key, rest is the field path
		mapKey := strings.ToLower(remainingParts[0])
		fieldEnvTag := strings.ToLower(strings.Join(remainingParts[1:], "_"))

		// Check if the field has an env tag and use the actual field name instead
		fieldPath := fieldEnvTag
		if fieldName, exists := fieldEnvMap[fieldEnvTag]; exists {
			fieldPath = fieldName
		}

		// Initialize map entry if not exists
		if mapEntries[mapKey] == nil {
			mapEntries[mapKey] = make(map[string]interface{})
		}

		// Store the field mapping - handle nested structures properly
		p.setNestedValue(mapEntries[mapKey], fieldPath, envValue)
	}

	// Set the entire map structure in viper and prevent AutomaticEnv from overriding it
	if len(mapEntries) > 0 {
		mapFieldKey := strings.Join(namespaces[1:], ".")
		p.viper.Set(mapFieldKey, mapEntries)

		// Explicitly bind the map field to prevent AutomaticEnv from overriding
		// We bind it to a non-existent env var to prevent automatic binding
		p.viper.BindEnv(mapFieldKey, "NONEXISTENT_ENV_VAR_"+mapFieldKey)
	}
}

// parseBasicMapEnv handles map[string]BasicType fields (e.g. map[string]string).
// It supports two patterns:
//   - Individual keys: PREFIX_FIELDNAME_KEY=value
//   - JSON string:     PREFIX_FIELDNAME='{"key":"value"}'
//
// If entries are found, they are set in viper and the field is bound to a
// non-existent env var to prevent AutomaticEnv from overriding with a raw string.
func (p *Configer) parseBasicMapEnv(field reflect.StructField, namespaces []string) {
	delim := p.envDelim
	if delim == "" {
		delim = "."
	}

	fieldName := namespaces[len(namespaces)-1]

	prefixNamespaces := make([]string, len(namespaces)-1)
	copy(prefixNamespaces, namespaces[:len(namespaces)-1])
	prefixNamespaces = append(prefixNamespaces, strings.ToUpper(fieldName))
	prefix := strings.ToUpper(strings.Join(prefixNamespaces, delim)) + delim

	mapEntries := make(map[string]interface{})

	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envKey := parts[0]
		envValue := parts[1]

		if envKey == strings.TrimSuffix(prefix, delim) {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(envValue), &parsed); err == nil {
				for k, v := range parsed {
					mapEntries[strings.ToLower(k)] = v
				}
			}
			continue
		}

		if !strings.HasPrefix(envKey, prefix) {
			continue
		}

		remaining := strings.TrimPrefix(envKey, prefix)
		mapKey := strings.ToLower(remaining)
		mapEntries[mapKey] = envValue
	}

	mapFieldKey := strings.Join(namespaces[1:], ".")
	if len(mapEntries) > 0 {
		p.viper.Set(mapFieldKey, mapEntries)
		p.viper.BindEnv(mapFieldKey, "NONEXISTENT_ENV_VAR_"+mapFieldKey)
	}
}

// parseSliceEnv handles slice types (e.g. []string, []int, []Struct).
// It supports JSON string format: PREFIX_FIELDNAME='["value1","value2"]'
//
// If a value is found, it is set in viper and the field is bound to a
// non-existent env var to prevent AutomaticEnv from overriding with a raw string.
func (p *Configer) parseSliceEnv(field reflect.StructField, namespaces []string) {
	delim := p.envDelim
	if delim == "" {
		delim = "."
	}

	fieldName := namespaces[len(namespaces)-1]

	prefixNamespaces := make([]string, len(namespaces)-1)
	copy(prefixNamespaces, namespaces[:len(namespaces)-1])
	prefixNamespaces = append(prefixNamespaces, strings.ToUpper(fieldName))
	envKey := strings.ToUpper(strings.Join(prefixNamespaces, delim))

	envValue := os.Getenv(envKey)
	if envValue == "" {
		return
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(envValue), &parsed); err != nil {
		return
	}

	sliceFieldKey := strings.Join(namespaces[1:], ".")
	p.viper.Set(sliceFieldKey, parsed)
	p.viper.BindEnv(sliceFieldKey, "NONEXISTENT_ENV_VAR_"+sliceFieldKey)
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
