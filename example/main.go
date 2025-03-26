package main

import (
	"fmt"

	"github.com/gitsang/configer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Log struct {
		Enable bool   `json:"enable" yaml:"enable" default:"true" usage:"enable log"`
		Level  string `json:"level" yaml:"level" default:"warn" usage:"log level"`
	} `json:"log" yaml:"log"`
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
		panic(err)
	}

	yamlBytes, _ := yaml.Marshal(c)
	fmt.Println(string(yamlBytes))
}

func main() {
	rootCmd.Execute()
}
