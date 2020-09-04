package cmd

import (
	"fmt"
	"net"
	"net/http"

	"github.com/P1llus/ess-openapi-servicebroker/broker"
	"github.com/P1llus/ess-openapi-servicebroker/config"
	"github.com/P1llus/ess-openapi-servicebroker/pkg/logger"
	"github.com/P1llus/ess-openapi-servicebroker/provider"
)

func Run() error {
	// Load Provider
	logger := logger.GetLogger()
	runtimeConfig := config.LoadConfig(logger)
	plans, services := config.LoadCatalogue(logger)
	provider := provider.NewProvider(runtimeConfig.Provider, plans, logger)

	// Load Broker
	runtimeBroker := broker.NewBroker(runtimeConfig.Broker, provider, services, logger)
	server := runtimeBroker.NewBrokerHTTPServer(runtimeBroker)

	// Create listener from config and serve the Broker API
	logger.Info(fmt.Sprintf("Starting new ServiceBroker listener on port %s", runtimeConfig.Broker.Port))
	listener, err := net.Listen(runtimeConfig.Broker.Protocol, fmt.Sprintf(":%s", runtimeConfig.Broker.Port))
	http.Serve(listener, server)

	return err
}
