/*
Package cmd is the starting point of the application when starting up
*/
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

// Run will be executed by main.go in the root directory and takes care of initializing
// the application and starting the HTTP listener.
func Run() error {
	logger := logger.GetLogger()
	runtimeConfig := config.LoadConfig(logger)
	plans, services := config.LoadCatalog(logger)

	provider := provider.NewProvider(runtimeConfig.Provider, plans, logger)

	runtimeBroker := broker.NewBroker(runtimeConfig.Broker, provider, services, logger)
	server := runtimeBroker.NewBrokerHTTPServer(runtimeBroker)

	logger.Info(fmt.Sprintf("Starting new ServiceBroker listener on port %s", runtimeConfig.Broker.Port))
	listener, err := net.Listen(runtimeConfig.Broker.Protocol, fmt.Sprintf(":%s", runtimeConfig.Broker.Port))
	http.Serve(listener, server)

	return err
}
