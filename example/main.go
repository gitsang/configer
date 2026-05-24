package main

import (
	"fmt"
	"os"

	"github.com/gitsang/configer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Host string `yaml:"host" mapstructure:"host" env:"HOST" default:"localhost" usage:"service host"`
	Port int    `yaml:"port" mapstructure:"port" env:"PORT" default:"8080" usage:"service port"`
}

type Config struct {
	Log struct {
		Enable bool   `yaml:"enable" mapstructure:"enable" default:"true" usage:"enable log"`
		Level  string `yaml:"level" mapstructure:"level" default:"warn" usage:"log level"`
	} `yaml:"log" mapstructure:"log"`

	Server struct {
		Hosts   []string `yaml:"hosts" mapstructure:"hosts" usage:"server hosts"`
		Ports   []int    `yaml:"ports" mapstructure:"ports" usage:"server ports"`
		Enabled []bool   `yaml:"enabled" mapstructure:"enabled" usage:"server enabled flags"`

		Endpoints []ServiceConfig `yaml:"endpoints" mapstructure:"endpoints" usage:"server endpoints"`

		Labels map[string]string `yaml:"labels" mapstructure:"labels" usage:"server labels"`

		PortMap map[string]int `yaml:"port_map" mapstructure:"port_map" usage:"server port map"`

		Services map[string]ServiceConfig `yaml:"services" mapstructure:"services" usage:"server services"`

		Advanced struct {
			Tags     []string          `yaml:"tags" mapstructure:"tags" usage:"advanced tags"`
			Config   map[string]string `yaml:"config" mapstructure:"config" usage:"advanced config"`
			Backends []ServiceConfig   `yaml:"backends" mapstructure:"backends" usage:"advanced backends"`
		} `yaml:"advanced" mapstructure:"advanced"`
	} `yaml:"server" mapstructure:"server"`
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
