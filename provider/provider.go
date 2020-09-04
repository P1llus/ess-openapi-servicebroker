/*
Package provider implements all functionality from the ess package to communicate with the Elastic Cloud API.
The package is created separately from the broker package to differentiate between the functionality defined
for a broker compared to the provider
*/
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/P1llus/ess-openapi-servicebroker/config"
	"github.com/P1llus/ess-openapi-servicebroker/pkg/esclient"
	"github.com/P1llus/ess-openapi-servicebroker/pkg/ess"
	"github.com/elastic/cloud-sdk-go/pkg/api"
	"github.com/elastic/cloud-sdk-go/pkg/auth"
	"github.com/elastic/cloud-sdk-go/pkg/models"
	"github.com/pivotal-cf/brokerapi/v7/domain"
)

// Provider struct describes the structure of a complete Provider object
type Provider struct {
	Client   *api.API
	Config   config.Provider
	Logger   lager.Logger
	Services []domain.Service
	Plans    []models.DeploymentCreateRequest
}

// OperationData struct builds a body of metadata about the current action, that is sent back to the broker and can be retrieved
// as a reference for asynchronous calls
type OperationData struct {
	Action       string
	DeploymentID string
	UserID       string `json:",omitempty"`
}

// Credentials struct used when sending credentials back to the broker
type Credentials struct {
	URI      string `json:"uri,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     string `json:"port,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewProvider returns a new Provider struct that includes the related Logger, Config and Plans objects
func NewProvider(providerConfig config.Provider, plans []models.DeploymentCreateRequest, logger lager.Logger) *Provider {
	essconfig, err := api.NewAPI(api.Config{
		Client:     new(http.Client),
		AuthWriter: auth.APIKey(providerConfig.APIKey),
		Host:       fmt.Sprintf("%s/api/%s", providerConfig.URL, providerConfig.Version),
		UserAgent:  fmt.Sprintf("%s/%s", providerConfig.UserAgent, providerConfig.Version),
	})
	if err != nil {
		logger.Fatal("Failed to create Provider:", err)
	}

	provider := &Provider{
		Client: essconfig,
		Config: providerConfig,
		Logger: logger,
		Plans:  plans,
	}
	logger.Info("Provider Initiated Successfully")

	return provider
}

// Provision compares the choosen PlanID to the local services files to find a match.
// When a match is found it will trigger the creation of a new cluster, using the InstanceID as the name
func (p *Provider) Provision(ctx context.Context, provision *ProvisionData) (string, string, error) {
	deploymentTemplate, err := config.FindDeploymentTemplateFromPlan(p.Plans, provision.Plan)
	deploymentTemplate.Name = provision.InstanceID
	res, err := ess.CreateDeployment(p.Client, &deploymentTemplate, provision.InstanceID)
	if err != nil {
		p.Logger.Error("Unable to Create a new deployment:", err, lager.Data{
			"instance-id": provision.InstanceID,
		})
		return "", "", err
	}
	deploymentID := *res.ID

	newKibana, _ := ess.GetKibana(p.Client, "main-kibana")
	dashboardURL := fmt.Sprintf("https://%s:%d", newKibana.Info.Metadata.Endpoint, *newKibana.Info.Metadata.Ports.HTTPS)

	provisionContext := &OperationData{
		Action:       "provision",
		DeploymentID: deploymentID,
	}
	var provisionContextJSON []byte
	provisionContextJSON, err = json.Marshal(provisionContext)
	if err != nil {
		p.Logger.Error("Unable to create OperationData for Provision task", err, lager.Data{
			"instance-id": provision.InstanceID,
		})
		return "", "", err
	}
	p.Logger.Info("New provision initiated successfully", lager.Data{
		"instance-id":   provision.InstanceID,
		"deployment-id": deploymentID,
	})

	operationData := string(provisionContextJSON)
	return dashboardURL, operationData, nil
}

// Deprovision deletes the cluster related to the instanceID used in the request
func (p *Provider) Deprovision(ctx context.Context, deprovisionData *DeprovisionData) (string, error) {
	deployment, err := ess.SearchDeployments(p.Client, deprovisionData.InstanceID)
	if err != nil {
		p.Logger.Error("Unable to find the related cluster to deprovision", err, lager.Data{
			"instance-id":   deprovisionData.InstanceID,
			"deployment-id": *deployment.ID,
		})
		return "", err
	}

	err = ess.ShutdownDeployment(p.Client, *deployment.ID)
	if err != nil {
		p.Logger.Error("Unable to delete the related cluster", err, lager.Data{
			"instance-id":   deprovisionData.InstanceID,
			"deployment-id": *deployment.ID,
		})
		return "", err
	}

	deprovisionContext := &OperationData{
		Action:       "deprovision",
		DeploymentID: *deployment.ID,
	}
	var deprovisionContextJSON []byte
	deprovisionContextJSON, err = json.Marshal(deprovisionContext)
	if err != nil {
		p.Logger.Error("Unable to create operationData context for Deprovision task", err, lager.Data{
			"instance-id":   deprovisionData.InstanceID,
			"deployment-id": *deployment.ID,
		})
		return "", err
	}
	p.Logger.Info("Deprovision has successfully been initiated", lager.Data{
		"instance-id":   deprovisionData.InstanceID,
		"deployment-id": *deployment.ID,
	})

	operationData := string(deprovisionContextJSON)

	return operationData, nil
}

// Bind operations creates a new user related to the BindID on the cluster related to the InstanceID in the request
func (p *Provider) Bind(ctx context.Context, bindData *BindData) (Credentials, string, error) {
	deployment, err := ess.SearchDeployments(p.Client, bindData.InstanceID)
	if err != nil {
		p.Logger.Error("Unable to find cluster for bind operation", err, lager.Data{
			"instance-id":   bindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       bindData.BindingID,
		})
		return Credentials{}, "", err
	}
	serviceURL := ess.GetServiceURL(p.Client, deployment)
	deploymentUsername, deploymentPassword := esclient.CreateBrokerCredentials(bindData.InstanceID, p.Config.Seed)
	deploymentClient, err := esclient.CreateV7Client(serviceURL, deploymentUsername, deploymentPassword)
	if err != nil {
		p.Logger.Error("Unable to create client connection to cluster", err, lager.Data{
			"instance-id":   bindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       bindData.BindingID,
			"service-url":   serviceURL,
		})
		return Credentials{}, "", err
	}
	ping, err := deploymentClient.Ping()
	if err != nil {
		p.Logger.Error("Authentication test towards cluster returned an error for bind operation", err, lager.Data{
			"instance-id":   bindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       bindData.BindingID,
			"service-url":   serviceURL,
		})
		return Credentials{}, "", err
	}
	if ping.StatusCode == 401 {
		p.Logger.Info("Authentication denied first try, resetting master password for cluster for bind operation", lager.Data{
			"instance-id":   bindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       bindData.BindingID,
			"service-url":   serviceURL,
		})
		newDeploymentPassword := ess.ResetElasticUserPassword(
			p.Config.URL,
			p.Config.Version,
			p.Config.APIKey,
			*deployment.ID)
		deploymentClient, err = esclient.CreateV7Client(serviceURL, "elastic", newDeploymentPassword)
		if err != nil {
			p.Logger.Error("Authentication denied second try after resetting password, cancelling bind operation", err, lager.Data{
				"instance-id":   bindData.InstanceID,
				"deployment-id": *deployment.ID,
				"bind-id":       bindData.BindingID,
				"service-url":   serviceURL,
			})
			return Credentials{}, "", err
		}
		time.Sleep(5 * time.Second)
		updateElasticPasswordOutcome, _ := esclient.UpdateBrokerPassword(deploymentClient, deploymentPassword)
		if updateElasticPasswordOutcome != 200 {
			p.Logger.Error("Update password for servicebroker account on cluster failed during bind operation", err, lager.Data{
				"instance-id":   bindData.InstanceID,
				"deployment-id": *deployment.ID,
				"bind-id":       bindData.BindingID,
				"service-url":   serviceURL,
			})
			return Credentials{}, "", err
		}
	}
	p.Logger.Info("Servicebroker authentication to cluster successful during bind operation", lager.Data{
		"instance-id":   bindData.InstanceID,
		"deployment-id": *deployment.ID,
		"bind-id":       bindData.BindingID,
		"service-url":   serviceURL,
	})

	bindUsername, bindPassword := esclient.CreateUserCredentials(bindData.BindingID, p.Config.Seed)
	bindOutcome, _ := esclient.CreateUserAccount(deploymentClient, bindUsername, bindPassword)
	if bindOutcome != 200 {
		p.Logger.Error("Unable to create new user account for bind operation", err, lager.Data{
			"instance-id":   bindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       bindData.BindingID,
			"service-url":   serviceURL,
		})
		return Credentials{}, "", fmt.Errorf("Unable to create new account for bind operation, statuscode: %d", bindOutcome)
	}
	credentials := Credentials{Username: bindUsername, Password: bindPassword}

	bindContext := &OperationData{
		Action:       "binding",
		DeploymentID: *deployment.ID,
		UserID:       bindUsername,
	}
	var bindContextJSON []byte
	bindContextJSON, err = json.Marshal(bindContext)
	if err != nil {
		p.Logger.Error("Unable to create operationData context for bind operation", err, lager.Data{
			"instance-id":   bindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       bindData.BindingID,
			"service-url":   serviceURL,
		})
		return Credentials{}, "", err
	}

	p.Logger.Info("New account created successfully during bind operation", lager.Data{
		"instance-id":   bindData.InstanceID,
		"deployment-id": *deployment.ID,
		"bind-id":       bindData.BindingID,
		"service-url":   serviceURL,
	})
	operationData := string(bindContextJSON)
	return credentials, operationData, nil
}

// Unbind operations deletes the user related to the BindID, on the cluster related to the InstanceID in the request
func (p *Provider) Unbind(ctx context.Context, unbindData *UnbindData) (string, error) {
	deployment, err := ess.SearchDeployments(p.Client, unbindData.InstanceID)
	if err != nil {
		p.Logger.Error("Unable to find cluster for unbind operation", err, lager.Data{
			"instance-id":   unbindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       unbindData.BindingID,
		})
		return "", err
	}
	serviceURL := ess.GetServiceURL(p.Client, deployment)
	deploymentUsername, deploymentPassword := esclient.CreateBrokerCredentials(unbindData.InstanceID, p.Config.Seed)
	deploymentClient, err := esclient.CreateV7Client(serviceURL, deploymentUsername, deploymentPassword)
	if err != nil {
		p.Logger.Error("Unable to create client connection to cluster during unbind operation", err, lager.Data{
			"instance-id":   unbindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       unbindData.BindingID,
			"service-url":   serviceURL,
		})
		return "", err
	}
	ping, err := deploymentClient.Ping()
	if err != nil {
		p.Logger.Info("Authentication denied first try, resetting master password for cluster for unbind operation", lager.Data{
			"instance-id":   unbindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       unbindData.BindingID,
			"service-url":   serviceURL,
		})
		return "", err
	}
	if ping.StatusCode == 401 {
		p.Logger.Info("Authentication denied first try, resetting master password for cluster for unbind operation", lager.Data{
			"instance-id":   unbindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       unbindData.BindingID,
			"service-url":   serviceURL,
		})
		newDeploymentPassword := ess.ResetElasticUserPassword(
			p.Config.URL,
			p.Config.Version,
			p.Config.APIKey,
			*deployment.ID)
		deploymentClient, err = esclient.CreateV7Client(serviceURL, "elastic", newDeploymentPassword)
		if err != nil {
			p.Logger.Error("Authentication denied second try after resetting password, cancelling unbind operation", err, lager.Data{
				"instance-id":   unbindData.InstanceID,
				"deployment-id": *deployment.ID,
				"bind-id":       unbindData.BindingID,
				"service-url":   serviceURL,
			})
			return "", err
		}
		time.Sleep(5 * time.Second)
		updateElasticPasswordOutcome, err := esclient.UpdateBrokerPassword(deploymentClient, deploymentPassword)
		if updateElasticPasswordOutcome != 200 {
			p.Logger.Error("Update password for servicebroker account on cluster failed during unbind operation", err, lager.Data{
				"instance-id":   unbindData.InstanceID,
				"deployment-id": *deployment.ID,
				"bind-id":       unbindData.BindingID,
				"service-url":   serviceURL,
			})
			return "", err
		}
	}
	p.Logger.Info("Servicebroker authentication to cluster successful during unbind operation", lager.Data{
		"instance-id":   unbindData.InstanceID,
		"deployment-id": *deployment.ID,
		"bind-id":       unbindData.BindingID,
		"service-url":   serviceURL,
	})
	unbindUsername, _ := esclient.CreateUserCredentials(unbindData.BindingID, p.Config.Seed)
	unbindOutcome, _ := esclient.DeleteUserAccount(deploymentClient, unbindUsername)
	if unbindOutcome != 200 {
		p.Logger.Error("Unable to delete user account during bind operation", fmt.Errorf("Unable to delete account, statuscode: %d", unbindOutcome), lager.Data{
			"instance-id":   unbindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       unbindData.BindingID,
			"service-url":   serviceURL,
		})
		return "", fmt.Errorf("Unable to delete account, statuscode: %d", unbindOutcome)
	}

	unbindContext := &OperationData{
		Action:       "unbind",
		DeploymentID: *deployment.ID,
		UserID:       unbindUsername,
	}
	var unbindContextJSON []byte
	unbindContextJSON, err = json.Marshal(unbindContext)
	if err != nil {
		p.Logger.Error("Unable to create operationData context for unbind operation", err, lager.Data{
			"instance-id":   unbindData.InstanceID,
			"deployment-id": *deployment.ID,
			"bind-id":       unbindData.BindingID,
			"service-url":   serviceURL,
		})
	}
	p.Logger.Info("Account deleted successfully", lager.Data{
		"instance-id":   unbindData.InstanceID,
		"deployment-id": *deployment.ID,
		"bind-id":       unbindData.BindingID,
		"service-url":   serviceURL,
	})

	operationData := string(unbindContextJSON)
	return operationData, nil
}

// Update changes the size of an existing cluster related to the InstanceID in the request
func (p *Provider) Update(context.Context, *UpdateData) (string, error) {
	return "", nil
}

// LastOperation is used to get the latest status on any Provision, Deprovision, Bind, Unbind or Update actions, since they are all asynchronous the
// initial function call to any of the mentioned functions does not expect it to finish, but rather uses LastOperation to confirm the current status
func (p *Provider) LastOperation(ctx context.Context, lastOperationData *LastOperationData) (state domain.LastOperationState, description string, err error) {
	var operationData OperationData
	err = json.Unmarshal([]byte(lastOperationData.OperationData), &operationData)
	p.Logger.Info(fmt.Sprintf("LastOperation check started for operation: %s", operationData.Action), lager.Data{
		"instance-id":   lastOperationData.InstanceID,
		"deployment-id": operationData.DeploymentID,
	})
	if operationData.Action == "provision" {
		deployment, _ := ess.GetDeployment(p.Client, operationData.DeploymentID)
		status := ess.DeploymentStatus(deployment, "started")
		if !status {
			return domain.InProgress, "Provision in progress", nil
		}
	}
	if operationData.Action == "deprovision" {
		deployment, _ := ess.GetDeployment(p.Client, operationData.DeploymentID)
		status := ess.DeploymentStatus(deployment, "stopped")
		if !status {
			return domain.InProgress, "Deprovision in progress", nil
		}
	}
	if operationData.Action == "bind" {
		deployment, err := ess.SearchDeployments(p.Client, lastOperationData.InstanceID)
		if err != nil {
			p.Logger.Error("LastOperation check failed for bind operation, cluster not found", err, lager.Data{
				"instance-id":   lastOperationData.InstanceID,
				"deployment-id": operationData.DeploymentID,
			})
			return domain.Failed, "Bind failed, cluster not found", nil
		}
		serviceURL := ess.GetServiceURL(p.Client, deployment)
		bindUsername, bindPassword := esclient.CreateUserCredentials(operationData.UserID, p.Config.Seed)
		deploymentClient, err := esclient.CreateV7Client(serviceURL, bindUsername, bindPassword)
		ping, err := deploymentClient.Ping()
		if ping.StatusCode != 200 {
			return domain.InProgress, "Bind in progress", nil
		}
	}
	if operationData.Action == "unbind" {
		deployment, err := ess.SearchDeployments(p.Client, lastOperationData.InstanceID)
		if err != nil {
			p.Logger.Error("LastOperation check failed for unbind operation, cluster not found", err, lager.Data{
				"instance-id":   lastOperationData.InstanceID,
				"deployment-id": operationData.DeploymentID,
			})
			return domain.Failed, "Unbind failed, cluster not found", nil
		}
		serviceURL := ess.GetServiceURL(p.Client, deployment)
		unbindUsername, unbindPassword := esclient.CreateUserCredentials(operationData.UserID, p.Config.Seed)
		deploymentClient, err := esclient.CreateV7Client(serviceURL, unbindUsername, unbindPassword)
		ping, err := deploymentClient.Ping()
		if ping.StatusCode == 200 {
			return domain.InProgress, "Unbind in progress", nil
		}
	}
	p.Logger.Info(fmt.Sprintf("LastOperation check finished for action: %s", operationData.Action), lager.Data{
		"instance-id":   lastOperationData.InstanceID,
		"deployment-id": operationData.DeploymentID,
	})
	return domain.Succeeded, "Last operation succeeded", nil
}
