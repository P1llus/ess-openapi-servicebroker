/*
Package config is used to define all configuration items used throughout the application
*/
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"code.cloudfoundry.org/lager"
	"github.com/elastic/cloud-sdk-go/pkg/models"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/spf13/viper"
)

// Config struct is a collection of all configuration items supported
type Config struct {
	Provider `mapstructure:"provider"`
	Broker   `mapstructure:"broker"`
}

// Provider struct includes all settings supported for the Provider
type Provider struct {
	Version   string `mapstructure:"version"`
	URL       string `mapstructure:"url"`
	APIKey    string `mapstructure:"apikey"`
	UserAgent string `mapstructure:"useragent"`
	Seed      string `mapstructure:"seed"`
}

// Broker struct includes all settings supported for the Broker
type Broker struct {
	Address   string `mapstructure:"address"`
	Port      string `mapstructure:"port"`
	URLPrefix string `mapstructure:"urlprefix"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	SSLConfig SSL    `mapstructure:"ssl"`
}

// SSL struct to be nested under Broker configuration for the HTTP Server
type SSL struct {
	Enabled     bool   `mapstructure:"enabled"`
	Certificate string `mapstructure:"certificate"`
	Key         string `mapstructure:"key"`
}

// LoadConfig tries to read the defined config file and return a Config struct upon success
func LoadConfig(v *viper.Viper, logger lager.Logger) *Config {
	var C Config
	err := v.Unmarshal(&C)
	if err != nil {
		fmt.Println(err)
	}
	return &C
}

// LoadCatalog returns a collection of DeploymentRequests for the Provider
// when Provisioning new cluster. It will also return a collection of all
// the available Service Catalog items that will be presented to any consumer
// utilizing the API. The ID of the Service Catalog item will need to match the name
// DeploymentRequest that should be used
func LoadCatalog(path string, logger lager.Logger) ([]models.DeploymentCreateRequest, []domain.Service) {
	planfile, err := ioutil.ReadFile(fmt.Sprintf("%s/plans.json", path))
	if err != nil {
		logger.Fatal("Error loading plans:", err, lager.Data{
			"plan-path": fmt.Sprintf("%s/plans.json", path),
		})
	}
	var plans []models.DeploymentCreateRequest

	err = json.Unmarshal([]byte(planfile), &plans)
	if err != nil {
		logger.Fatal("Unable to import plans file, Unmarshal failure:", err, lager.Data{
			"plan-path": fmt.Sprintf("%s/plans.json", path),
		})
		return []models.DeploymentCreateRequest{}, []domain.Service{}
	}
	logger.Info("Plans file loaded", lager.Data{
		"plan-path": fmt.Sprintf("%s/plans.json", path),
	})

	servicefile, err := ioutil.ReadFile(fmt.Sprintf("%s/services.json", path))
	if err != nil {
		fmt.Println(err)
	}
	var services []domain.Service

	err = json.Unmarshal([]byte(servicefile), &services)
	if err != nil {
		logger.Fatal("Unable to import services file, Unmarshal failure:", err, lager.Data{
			"service-path": fmt.Sprintf("%s/services.json", path),
		})
		return []models.DeploymentCreateRequest{}, []domain.Service{}
	}
	logger.Info("Services file loaded", lager.Data{
		"service-path": fmt.Sprintf("%s/services.json", path),
	})

	return plans, services
}

// FindProvisionDetails will iterate over the Service Catalog and return the correct plan
// related to the planId parameter
func FindProvisionDetails(services []domain.Service, serviceID string, planID string) (domain.ServicePlan, error) {
	for _, service := range services {
		if service.ID == serviceID {
			for _, plan := range service.Plans {
				if plan.ID == planID {
					return plan, nil
				}
			}
		}
	}
	return domain.ServicePlan{}, fmt.Errorf("could not find service with ID:%s and plan with ID: %s", serviceID, planID)
}

// FindDeploymentTemplateFromPlan will return the correct DeploymentRequest matching the plan parameter
func FindDeploymentTemplateFromPlan(deployments []models.DeploymentCreateRequest, plan domain.ServicePlan) (models.DeploymentCreateRequest, error) {
	for _, deployment := range deployments {
		if deployment.Name == plan.Name {
			return deployment, nil
		}
	}
	return models.DeploymentCreateRequest{}, fmt.Errorf("could not find a Elasticsearch deployment template that matches Plan ID: %s", plan.ID)
}
