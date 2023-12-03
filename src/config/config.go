package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Certs  certsConfig
	Server serverConfig
	Client clientConfig
}

type certsConfig struct {
	CAFile               string
	ServerCertFile       string
	ServerKeyFile        string
	ClientCertFile       string
	ClientKeyFile        string
	RootClientCertFile   string
	RootClientKeyFile    string
	NobodyClientCertFile string
	NobodyClientKeyFile  string
	ACLModelFile         string
	ACLPolicyFile        string
}

type serverConfig struct {
	Address       string
	MaxStoreBytes uint64
	MaxIndexBytes uint64
	LogDirectory  string
}

type clientConfig struct {
	GatewayPort int
}

var (
	configFile string
	config     Config
)

// InitializeConfig initializes Viper and Cobra configurations.
func InitializeConfig(rootCmd *cobra.Command) error {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file (default is $HOME/.config.yaml)")

	// Bind Viper to Cobra
	if err := viper.BindPFlag("config", rootCmd.Flags().Lookup("config")); err != nil {
		return err
	}
	return nil
}

// LoadConfig loads the configuration from the specified file or the default file in the home directory.
func LoadConfig() (*Config, error) {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Search config in home directory with name ".config" (without extension).
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}
	viper.SetConfigType("yaml")

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Error reading config file: %s", err)
	}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("Error unmarshaling config: %s", err)
	}

	return &config, nil
}
