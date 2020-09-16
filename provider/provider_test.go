package provider

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v7/esapi"

	"github.com/P1llus/ess-openapi-servicebroker/config"
	"github.com/P1llus/ess-openapi-servicebroker/pkg/logger"
	"github.com/elastic/cloud-sdk-go/pkg/models"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var (
	defaultLogger = logger.GetLogger()
)

var (
	defaultViper    = viper.New()
	defaultConfPath = "../config"
)

var (
	runtimeConfig   = config.LoadConfig(defaultViper, defaultLogger)
	plans, services = config.LoadCatalog(defaultConfPath, defaultLogger)
)

func init() {
	defaultViper.SetEnvPrefix("ESS")
	defaultViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defaultViper.AutomaticEnv()
	defaultViper.AddConfigPath(defaultConfPath)
	defaultViper.SetConfigName(defaultViper.GetString("config.yml"))
}

func loadResponse(response string, port int32) interface{} {
	switch r := response; r {
	case "deploymentCreateResponse":
		file, _ := ioutil.ReadFile("testdata/deploymentcreateresponse.json")
		response := models.DeploymentCreateResponse{}
		json.Unmarshal(file, &response)
		return response
	case "getKibanaResponse":
		file, _ := ioutil.ReadFile("testdata/getkibanaresponse.json")
		response := models.KibanaResourceInfo{}
		json.Unmarshal(file, &response)
		return response
	case "shutdownResponse":
		file, _ := ioutil.ReadFile("testdata/shutdownresponse.json")
		response := models.DeploymentShutdownResponse{}
		json.Unmarshal(file, &response)
		return response
	case "bindResponse":
		file, _ := ioutil.ReadFile("testdata/elasticsearchcreateuserresponse.json")
		response := esapi.SecurityPutUserRequest{}
		json.Unmarshal(file, &response)
		return response
	case "unbindResponse":
		file, _ := ioutil.ReadFile("testdata/elasticsearchdeleteuserresponse.json")
		response := esapi.SecurityDeleteUserRequest{}
		json.Unmarshal(file, &response)
		return response
	case "deploymentStatusResponse":
		file, _ := ioutil.ReadFile("testdata/getdeploymentstatus.json")
		response := models.DeploymentGetResponse{}
		json.Unmarshal(file, &response)
		return response
	case "searchResponse":
		file, _ := ioutil.ReadFile("testdata/searchresponse.json")
		response := models.DeploymentsSearchResponse{}
		json.Unmarshal(file, &response)
		if port != 0 {
			response.Deployments[0].Resources.Elasticsearch[0].Info.Metadata.Endpoint = "localhost"
			response.Deployments[0].Resources.Elasticsearch[0].Info.Metadata.Ports.HTTPS = &port
		}
		return response
	}
	return nil
}

func TestProvision(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Accept", "application/json")
		switch path := r.URL.Path; path {
		case "/api/v1/deployments":
			file := loadResponse("deploymentCreateResponse", 0)
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(file)
		case "/api/v1/deployments/063e2805388445ebbc7579fdb051014a/kibana/":
			file := loadResponse("getKibanaResponse", 0)
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(file)
		}
	}))

	runtimeConfig.Provider.URL = server.URL
	runtimeProvider := NewProvider(runtimeConfig.Provider, plans, defaultLogger)
	details := domain.ProvisionDetails{
		ServiceID: "uuid-1",
		PlanID:    "uuid-2",
	}
	plan, _ := config.FindProvisionDetails(services, details.ServiceID, details.PlanID)
	provisionData := &ProvisionData{
		InstanceID: "mock-test",
		Details:    details,
		Plan:       plan,
	}
	dashboardURL, operationData, provisionErr := runtimeProvider.Provision(context.Background(), provisionData)
	assert.Equal(t, provisionErr, nil)
	assert.Equal(t, dashboardURL, "https://063e2805388445ebbc7579fdb051014a.europe-west1.gcp.cloud.es.io:9243", "Checking Dashboard URL")
	assert.Equal(t, operationData, `{"Action":"provision","DeploymentID":"0837d2cd080743e9be080bca163c0b92"}`, "Checking returned operationData")
	t.Cleanup(server.Close)
}

func TestDeprovision(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Accept", "application/json")
		switch path := r.URL.Path; path {
		case "/api/v1/deployments/0837d2cd080743e9be080bca163c0b92/_shutdown":
			file := loadResponse("shutdownResponse", 0)
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(file)
		case "/api/v1/deployments/_search":
			file := loadResponse("searchResponse", 0)
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(file)
		}
	}))

	runtimeConfig.Provider.URL = server.URL
	runtimeProvider := NewProvider(runtimeConfig.Provider, plans, defaultLogger)
	details := domain.DeprovisionDetails{
		ServiceID: "uuid-1",
		PlanID:    "uuid-2",
	}
	deprovisionData := &DeprovisionData{
		InstanceID: "mock-test",
		Details:    details,
	}
	operationData, deprovisionErr := runtimeProvider.Deprovision(context.Background(), deprovisionData)
	assert.Equal(t, deprovisionErr, nil, "Ensure Deprovision error is nil")
	assert.Equal(t, operationData, `{"Action":"deprovision","DeploymentID":"61b9a27f0a8c47d9be1c1b2d5986bbc8"}`, "Ensure Deprovision error is nil")

	t.Cleanup(server.Close)
}

func TestBind(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Accept", "application/json")
		_, port, _ := net.SplitHostPort(r.Host)
		intPort, _ := strconv.Atoi(port)
		switch path := r.URL.Path; path {
		case "/api/v1/deployments/_search":
			file := loadResponse("searchResponse", int32(intPort))
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(file)
		case "/_security/user/bind-test1":
			file := loadResponse("bindResponse", 0)
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(file)
		}
	}))

	runtimeConfig.Provider.URL = server.URL
	runtimeProvider := NewProvider(runtimeConfig.Provider, plans, defaultLogger)
	details := domain.BindDetails{
		ServiceID: "uuid-1",
		PlanID:    "uuid-2",
	}
	bindData := &BindData{
		InstanceID: "mock-test123123123123",
		BindingID:  "bind-test123123123123123",
		Details:    details,
	}
	credentials, operationData, bindErr := runtimeProvider.Bind(context.Background(), bindData)
	assert.Equal(t, credentials, Credentials{Hostname: "", URI: "", Port: "", Username: "bind-test1", Password: "a0c37ed81aede644bd333d552bfc2fa339a94841"})
	assert.Equal(t, operationData, `{"Action":"binding","DeploymentID":"61b9a27f0a8c47d9be1c1b2d5986bbc8","UserID":"bind-test1"}`)
	assert.Equal(t, bindErr, nil)

	t.Cleanup(server.Close)
}

func TestUnbind(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Accept", "application/json")
		_, port, _ := net.SplitHostPort(r.Host)
		intPort, _ := strconv.Atoi(port)
		switch path := r.URL.Path; path {
		case "/api/v1/deployments/_search":
			file := loadResponse("searchResponse", int32(intPort))
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(file)
		case "/_security/user/bind-test1":
			file := loadResponse("unbindResponse", 0)
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(file)
		}
	}))

	runtimeConfig.Provider.URL = server.URL
	runtimeProvider := NewProvider(runtimeConfig.Provider, plans, defaultLogger)
	details := domain.UnbindDetails{
		ServiceID: "uuid-1",
		PlanID:    "uuid-2",
	}
	unbindData := &UnbindData{
		InstanceID: "mock-test123123123123",
		BindingID:  "bind-test123123123123123",
		Details:    details,
	}
	operationData, unbindErr := runtimeProvider.Unbind(context.Background(), unbindData)

	assert.Equal(t, operationData, `{"Action":"unbind","DeploymentID":"61b9a27f0a8c47d9be1c1b2d5986bbc8","UserID":"bind-test1"}`, "Testing Unbind")
	assert.Equal(t, unbindErr, nil)

	t.Cleanup(server.Close)
}

func TestLastOperation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Accept", "application/json")
		switch path := r.URL.Path; path {
		case "/api/v1/deployments/61b9a27f0a8c47d9be1c1b2d5986bbc8":
			file := loadResponse("deploymentStatusResponse", 0)
			if file == nil {
				t.Error("Response file not found")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(file)
		}
	}))

	runtimeConfig.Provider.URL = server.URL
	runtimeProvider := NewProvider(runtimeConfig.Provider, plans, defaultLogger)

	unbindContext := &OperationData{
		Action:       "provision",
		DeploymentID: "61b9a27f0a8c47d9be1c1b2d5986bbc8",
		UserID:       "elastic-123123",
	}
	var unbindContextJSON []byte
	unbindContextJSON, _ = json.Marshal(unbindContext)

	details := domain.PollDetails{
		ServiceID:     "uuid-1",
		PlanID:        "uuid-2",
		OperationData: string(unbindContextJSON),
	}
	lastOperationData := &LastOperationData{
		InstanceID:    "mock-test123123123123",
		OperationData: details.OperationData,
	}
	state, operationData, lastOperationErr := runtimeProvider.LastOperation(context.Background(), lastOperationData)
	assert.Equal(t, state, domain.Succeeded)
	assert.Equal(t, operationData, "last operation succeeded")
	assert.Equal(t, lastOperationErr, nil)

	t.Cleanup(server.Close)
}
