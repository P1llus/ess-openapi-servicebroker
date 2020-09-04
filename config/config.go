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
	Provider `yaml:"provider"`
	Broker   `yaml:"broker"`
}

// Provider struct includes all settings supported for the Provider
type Provider struct {
	Version   string `yaml:"version"`
	URL       string `yaml:"url"`
	APIKey    string `yaml:"apikey"`
	UserAgent string `yaml:"useragent"`
	Seed      string `yaml:"seed"`
}

// Broker struct includes all settings supported for the Broker
type Broker struct {
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// LoadConfig tries to read the defined config file and return a Config struct upon success
func LoadConfig(logger lager.Logger) *Config {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	err := v.ReadInConfig()
	if err != nil {
		logger.Fatal("Error loading config file:", err, lager.Data{
			"config-path": "./config/config.yml",
		})
		return nil
	}
	var C Config
	err = v.Unmarshal(&C)
	if err != nil {
		logger.Fatal("Unable to import config file, Unmarshal failure:", err, lager.Data{
			"config-path": "./config/config.yml",
		})
	}
	logger.Info("Config file loaded", lager.Data{
		"config-path": "./config/config.yml",
	})
	return &C
}

// LoadCatalog returns a collection of DeploymentRequests for the Provider
// When Provisioning new cluster. It will also return a collection of all
// the available Service Catalog that will be presented to any consumer
// utilizing the API
func LoadCatalog(logger lager.Logger) ([]models.DeploymentCreateRequest, []domain.Service) {
	planfile, err := ioutil.ReadFile("./config/plans.json")
	if err != nil {
		logger.Fatal("Error loading plans:", err, lager.Data{
			"plan-path": "./config/plans.json",
		})
	}
	var plans []models.DeploymentCreateRequest

	err = json.Unmarshal([]byte(planfile), &plans)
	if err != nil {
		logger.Fatal("Unable to import plans file, Unmarshal failure:", err, lager.Data{
			"plan-path": "./config/plans.json",
		})
		return []models.DeploymentCreateRequest{}, []domain.Service{}
	}
	logger.Info("Plans file loaded", lager.Data{
		"plan-path": "./config/plans.json",
	})

	servicefile, err := ioutil.ReadFile("./config/services.json")
	if err != nil {
		fmt.Println(err)
	}
	var services []domain.Service

	err = json.Unmarshal([]byte(servicefile), &services)
	if err != nil {
		logger.Fatal("Unable to import services file, Unmarshal failure:", err, lager.Data{
			"service-path": "./config/services.json",
		})
		return []models.DeploymentCreateRequest{}, []domain.Service{}
	}
	logger.Info("Services file loaded", lager.Data{
		"service-path": "./config/services.json",
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
