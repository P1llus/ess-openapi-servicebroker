/*
Package cmd is the starting point of the application when starting up
*/
package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/P1llus/ess-openapi-servicebroker/broker"
	"github.com/P1llus/ess-openapi-servicebroker/config"
	"github.com/P1llus/ess-openapi-servicebroker/pkg/logger"
	"github.com/P1llus/ess-openapi-servicebroker/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// General defaults
var (
	defaultLogger = logger.GetLogger()
)

// Viper defaults
var (
	defaultViper    = viper.New()
	defaultConfPath = "./config"
)

// General variable flags for Cobra
var (
	cfgFile string
	Verbose bool
)

// Broker variable flags for Cobra
var (
	username      string
	password      string
	listenAddress string
	listenPort    string
	urlPrefix     string
	sslEnabled    bool
	certificate   string
	privateKey    string
)

// Provider variable flags for Cobra
var (
	providerURL     string
	providerVersion string
	apiKey          string
	userAgent       string
	seed            string
)

var rootCmd = &cobra.Command{
	Use:   "ess-servicebroker",
	Short: "OpenAPI Servicebroker for Elastic Cloud",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

// Execute will be executed by main.go in the root directory and takes care of initializing
// the application and starting the HTTP listener.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config", "Name of the config file (default is config.yaml)")
	defaultViper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	rootCmd.PersistentFlags().BoolVar(&Verbose, "verbose", false, "verbose output")
	defaultViper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Broker config flags
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Basic Auth username for the Servicebroker HTTP listener")
	defaultViper.BindPFlag("broker.username", rootCmd.PersistentFlags().Lookup("username"))
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "Basic Auth password for the Servicebroker HTTP listener")
	defaultViper.BindPFlag("broker.password", rootCmd.PersistentFlags().Lookup("password"))
	rootCmd.PersistentFlags().StringVar(&listenAddress, "address", "localhost", "The IP or hostname to bind the HTTP listener to")
	defaultViper.BindPFlag("broker.address", rootCmd.PersistentFlags().Lookup("address"))
	rootCmd.PersistentFlags().StringVar(&listenPort, "port", "8000", "The port to bind the HTTP listener to")
	defaultViper.BindPFlag("broker.port", rootCmd.PersistentFlags().Lookup("port"))
	rootCmd.PersistentFlags().StringVar(&urlPrefix, "urlprefix", "/", "The URL prefix for the HTTP Endpoint")
	defaultViper.BindPFlag("broker.urlprefix", rootCmd.PersistentFlags().Lookup("urlprefix"))
	rootCmd.PersistentFlags().StringVar(&certificate, "cert", "server.crt", "Path to the certificate if HTTP is enabled")
	defaultViper.BindPFlag("broker.ssl.cert", rootCmd.PersistentFlags().Lookup("cert"))
	rootCmd.PersistentFlags().StringVar(&privateKey, "privatekey", "server.key", "Path to the certificate private key if HTTP is enabled")
	defaultViper.BindPFlag("broker.ssl.key", rootCmd.PersistentFlags().Lookup("privatekey"))
	rootCmd.PersistentFlags().BoolVar(&sslEnabled, "ssl", false, "Enable the use of HTTPS")
	defaultViper.BindPFlag("broker.ssl.enabled", rootCmd.PersistentFlags().Lookup("ssl"))

	// Provider config flags
	rootCmd.PersistentFlags().StringVar(&providerURL, "providerurl", "https://api.elastic-cloud.com", "The API Endpoint for Elastic Cloud API, defaults to https://api.elastic-cloud.com")
	defaultViper.BindPFlag("provider.url", rootCmd.PersistentFlags().Lookup("providerurl"))
	rootCmd.PersistentFlags().StringVar(&providerVersion, "providerversion", "v1", "The version of the Elastic Cloud API to use, defaults to v1")
	defaultViper.BindPFlag("provider.version", rootCmd.PersistentFlags().Lookup("providerversion"))
	rootCmd.PersistentFlags().StringVar(&apiKey, "apikey", "", "API key to authenticate to the Elastic Cloud API")
	defaultViper.BindPFlag("provider.apikey", rootCmd.PersistentFlags().Lookup("apikey"))
	rootCmd.PersistentFlags().StringVar(&userAgent, "useragent", "cloud-sdk-go", "User agent used when communicating with Elastic Cloud API, defaults to cloud-sdk-go")
	defaultViper.BindPFlag("provider.useragent", rootCmd.PersistentFlags().Lookup("useragent"))
	rootCmd.PersistentFlags().StringVar(&seed, "seed", "cloud-sdk-go", "User agent used when communicating with Elastic Cloud API, defaults to cloud-sdk-go")
	defaultViper.BindPFlag("provider.seed", rootCmd.PersistentFlags().Lookup("seed"))
}

func initConfig() error {
	defaultViper.SetEnvPrefix("ESS")
	defaultViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defaultViper.AutomaticEnv()
	defaultViper.AddConfigPath(defaultConfPath)
	defaultViper.SetConfigName(defaultViper.GetString("config"))
	if err := defaultViper.ReadInConfig(); err == nil && defaultViper.GetBool("verbose") {
		defaultLogger.Info(fmt.Sprintf("Using config file: %s", defaultViper.ConfigFileUsed()))
		return err
	}
	return nil
}

func run() error {
	runtimeConfig := config.LoadConfig(defaultViper, defaultLogger)
	plans, services := config.LoadCatalog(defaultLogger)

	runtimeProvider := provider.NewProvider(runtimeConfig.Provider, plans, defaultLogger)
	runtimeBroker := broker.NewBroker(runtimeConfig.Broker, runtimeProvider, services, defaultLogger)

	mux := runtimeBroker.NewBrokerHTTPServer(runtimeBroker)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", runtimeConfig.Address, runtimeConfig.Port),
		Handler: mux,
	}
	if runtimeConfig.Broker.SSLConfig.Enabled {
		defaultLogger.Info(fmt.Sprintf("Starting new ServiceBroker HTTPS listener on port %s", runtimeConfig.Broker.Port), lager.Data{
			"ssl":     "true",
			"port":    runtimeConfig.Broker.Port,
			"address": runtimeConfig.Address,
		})
		if err := httpServer.ListenAndServeTLS(runtimeConfig.Broker.SSLConfig.Certificate, runtimeConfig.Broker.SSLConfig.Key); err != http.ErrServerClosed {
			return fmt.Errorf("HTTPS Server shutdown with error: %s", err)
		}
	} else {
		defaultLogger.Info(fmt.Sprintf("Starting new ServiceBroker HTTP listener on port %s", runtimeConfig.Broker.Port), lager.Data{
			"ssl":     "false",
			"port":    runtimeConfig.Broker.Port,
			"address": runtimeConfig.Address,
		})
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			return fmt.Errorf("HTTP Server shutdown with error: %s", err)
		}
	}

	return nil
}
