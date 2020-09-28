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

// Config defaults
var (
	defaultViper       = viper.New()
	defaultConfPath    = "./config"
	defaultConfName    = "config.yml"
	defaultProviderURL = "https://api.elastic-cloud.com"
)

// General variable flags for Cobra
var (
	cfgFile string
	cfgPath string
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

func init() {
	setCobraFlags(rootCmd)
	bindViperFlags(defaultViper, rootCmd)
}

func setCobraFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&cfgFile, "configfile", defaultConfName, "Name of the config file (default is config.yaml)")
	cmd.PersistentFlags().StringVar(&cfgPath, "configpath", defaultConfPath, "Path to the config file")
	cmd.PersistentFlags().BoolVar(&Verbose, "verbose", false, "verbose output")

	// Broker config flags
	cmd.PersistentFlags().StringVar(&username, "username", "", "Basic Auth username for the Servicebroker HTTP listener")
	cmd.PersistentFlags().StringVar(&password, "password", "", "Basic Auth password for the Servicebroker HTTP listener")
	cmd.PersistentFlags().StringVar(&listenAddress, "address", "localhost", "The IP or hostname to bind the HTTP listener to")
	cmd.PersistentFlags().StringVar(&listenPort, "port", "8000", "The port to bind the HTTP listener to")
	cmd.PersistentFlags().StringVar(&urlPrefix, "urlprefix", "/", "The URL prefix for the HTTP Endpoint")
	cmd.PersistentFlags().StringVar(&certificate, "cert", "server.crt", "Path to the certificate if HTTP is enabled")
	cmd.PersistentFlags().StringVar(&privateKey, "privatekey", "server.key", "Path to the certificate private key if HTTP is enabled")
	cmd.PersistentFlags().BoolVar(&sslEnabled, "ssl", false, "Enable the use of HTTPS")

	// Provider config flags
	cmd.PersistentFlags().StringVar(&providerURL, "providerurl", defaultProviderURL, "The API Endpoint for Elastic Cloud API, defaults to https://api.elastic-cloud.com")
	cmd.PersistentFlags().StringVar(&providerVersion, "providerversion", "v1", "The version of the Elastic Cloud API to use, defaults to v1")
	cmd.PersistentFlags().StringVar(&apiKey, "apikey", "", "API key to authenticate to the Elastic Cloud API")
	cmd.PersistentFlags().StringVar(&userAgent, "useragent", "cloud-sdk-go", "User agent used when communicating with Elastic Cloud API, defaults to cloud-sdk-go")
	cmd.PersistentFlags().StringVar(&seed, "seed", "cloud-sdk-go", "User agent used when communicating with Elastic Cloud API, defaults to cloud-sdk-go")
}

func bindViperFlags(v *viper.Viper, cmd *cobra.Command) {
	v.BindPFlag("configfile", cmd.PersistentFlags().Lookup("configfile"))
	v.BindPFlag("configpath", cmd.PersistentFlags().Lookup("configpath"))
	v.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))
	v.BindPFlag("broker.username", cmd.PersistentFlags().Lookup("username"))
	v.BindPFlag("broker.password", cmd.PersistentFlags().Lookup("password"))
	v.BindPFlag("broker.address", cmd.PersistentFlags().Lookup("address"))
	v.BindPFlag("broker.port", cmd.PersistentFlags().Lookup("port"))
	v.BindPFlag("broker.urlprefix", cmd.PersistentFlags().Lookup("urlprefix"))
	v.BindPFlag("broker.ssl.cert", cmd.PersistentFlags().Lookup("cert"))
	v.BindPFlag("broker.ssl.key", cmd.PersistentFlags().Lookup("privatekey"))
	v.BindPFlag("broker.ssl.enabled", cmd.PersistentFlags().Lookup("ssl"))
	v.BindPFlag("provider.url", cmd.PersistentFlags().Lookup("providerurl"))
	v.BindPFlag("provider.version", cmd.PersistentFlags().Lookup("providerversion"))
	v.BindPFlag("provider.apikey", cmd.PersistentFlags().Lookup("apikey"))
	v.BindPFlag("provider.useragent", cmd.PersistentFlags().Lookup("useragent"))
	v.BindPFlag("provider.seed", cmd.PersistentFlags().Lookup("seed"))
}

// Execute will be executed by main.go in the root directory and takes care of initializing
// the application and starting the HTTP listener.
func Execute() error {
	return rootCmd.Execute()
}

func initConfig() error {
	defaultViper.SetEnvPrefix("ESS")
	defaultViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defaultViper.AutomaticEnv()
	defaultViper.AddConfigPath(defaultViper.GetString("configpath"))
	defaultViper.SetConfigName(defaultViper.GetString("configfile"))
	if err := defaultViper.ReadInConfig(); err == nil && defaultViper.GetBool("verbose") {
		defaultLogger.Info(fmt.Sprintf("Using config file: %s", defaultViper.ConfigFileUsed()))
		return err
	}
	return nil
}

func run() error {
	runtimeConfig := config.LoadConfig(defaultViper, defaultLogger)
	plans, services := config.LoadCatalog(defaultViper.GetString("configpath"), defaultLogger)

	runtimeProvider := provider.NewProvider(runtimeConfig.Provider, plans, defaultLogger)
	runtimeBroker := broker.NewBroker(runtimeConfig.Broker, runtimeProvider, services, defaultLogger)

	mux := runtimeBroker.NewBrokerHTTPServer(runtimeBroker)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", runtimeConfig.Broker.Address, runtimeConfig.Broker.Port),
		Handler: mux,
	}
	if runtimeConfig.Broker.SSLConfig.Enabled {
		defaultLogger.Info(fmt.Sprintf("Starting new ServiceBroker HTTPS listener on port %s", runtimeConfig.Broker.Port), lager.Data{
			"ssl":     "true",
			"port":    runtimeConfig.Broker.Port,
			"address": runtimeConfig.Broker.Address,
		})
		if err := httpServer.ListenAndServeTLS(runtimeConfig.Broker.SSLConfig.Certificate, runtimeConfig.Broker.SSLConfig.Key); err != http.ErrServerClosed {
			return fmt.Errorf("HTTPS Server shutdown with error: %s", err)
		}
	} else {
		defaultLogger.Info(fmt.Sprintf("Starting new ServiceBroker HTTP listener on port %s", runtimeConfig.Broker.Port), lager.Data{
			"ssl":     "false",
			"port":    runtimeConfig.Broker.Port,
			"address": runtimeConfig.Broker.Address,
		})
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			return fmt.Errorf("HTTP Server shutdown with error: %s", err)
		}
	}

	return nil
}
