package main

import (
	"fmt"
	"os"

	"github.com/gitsang/configer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Host string `yaml:"host" env:"HOST" default:"localhost" usage:"service host"`
	Port int    `yaml:"port" env:"PORT" default:"8080" usage:"service port"`
}

type Config struct {
	Log struct {
		Enable bool   `yaml:"enable" default:"true" usage:"enable log"`
		Level  string `yaml:"level" default:"warn" usage:"log level"`
	} `yaml:"log"`

	Server struct {
		// map[string]string - basic type map
		Labels map[string]string `yaml:"labels" usage:"server labels"`

		// map[string]int - int type map
		Ports map[string]int `yaml:"ports" usage:"server ports"`

		// map[string]bool - bool type map
		Features map[string]bool `yaml:"features" usage:"server features"`

		// map[string]Struct - struct type map
		Services map[string]ServiceConfig `yaml:"services" usage:"server services"`

		// Nested struct with map
		Advanced struct {
			Tags   map[string]string   `yaml:"tags" usage:"advanced tags"`
			Routes map[string][]string `yaml:"routes" usage:"advanced routes"`
		} `yaml:"advanced"`
	} `yaml:"server"`
}

var rootCmd = &cobra.Command{
	Use: "example",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

var rootFlags = struct {
	ConfigPaths []string
}{}

var cfger *configer.Configer

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&rootFlags.ConfigPaths, "config", "c", nil, "config file path")

	cfger = configer.New(
		configer.WithTemplate(new(Config)),
		configer.WithEnvBind(
			configer.WithEnvPrefix("EXAMPLE"),
			configer.WithEnvDelim("_"),
		),
		configer.WithFlagBind(
			configer.WithCommand(rootCmd),
			configer.WithFlagPrefix("example"),
			configer.WithFlagDelim("."),
		),
	)
}

func run() {
	var c Config
	err := cfger.Load(&c, rootFlags.ConfigPaths...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	yamlBytes, _ := yaml.Marshal(c)
	fmt.Println(string(yamlBytes))
}

func main() {
	rootCmd.Execute()
}
