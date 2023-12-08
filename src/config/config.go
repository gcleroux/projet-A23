package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Certs   certsConfig    `mapstructure:"certs"`
	Servers []serverConfig `mapstructure:"servers"`
}

type certsConfig struct {
	CAFile         string `mapstructure:"ca_file"`
	ServerCertFile string `mapstructure:"server_cert_file"`
	ServerKeyFile  string `mapstructure:"server_key_file"`
	ClientCertFile string `mapstructure:"client_cert_file"`
	ClientKeyFile  string `mapstructure:"client_key_file"`
	UserCertFile   string `mapstructure:"user_cert_file"`
	UserKeyFile    string `mapstructure:"user_key_file"`
	NobodyCertFile string `mapstructure:"nobody_cert_file"`
	NobodyKeyFile  string `mapstructure:"nobody_key_file"`
	ACLModelFile   string `mapstructure:"acl_model_file"`
	ACLPolicyFile  string `mapstructure:"acl_policy_file"`
}

type serverConfig struct {
	NodeName     string   `mapstructure:"node_name"`
	Bootstrap    bool     `mapstructure:"bootstrap"`
	JoinAddr     []string `mapstructure:"join_addr"`
	Address      string   `mapstructure:"address"`
	LogDirectory string   `mapstructure:"log_directory"`
	SerfPort     int      `mapstructure:"serf_port"`
	RPCPort      int      `mapstructure:"rpc_port"`
	GatewayPort  int      `mapstructure:"gateway_port"`
	Latitude     float64  `mapstructure:"latitude"`
	Longitude    float64  `mapstructure:"longitude"`
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
