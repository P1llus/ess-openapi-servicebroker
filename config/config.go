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

type Config struct {
	Provider `yaml:"provider"`
	Broker   `yaml:"broker"`
}

type Provider struct {
	Version   string `yaml:"version"`
	URL       string `yaml:"url"`
	APIKey    string `yaml:"apikey"`
	UserAgent string `yaml:"useragent"`
	Seed      string `yaml:"seed"`
}

type Broker struct {
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

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

func LoadCatalogue(logger lager.Logger) ([]models.DeploymentCreateRequest, []domain.Service) {
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

func FindProvisionDetails(services []domain.Service, serviceId string, planId string) (domain.ServicePlan, error) {
	for _, service := range services {
		if service.ID == serviceId {
			for _, plan := range service.Plans {
				if plan.ID == planId {
					return plan, nil
				}
			}
		}
	}
	return domain.ServicePlan{}, fmt.Errorf("Could not find service with ID:%s and plan with ID: %s", serviceId, planId)
}

func FindDeploymentTemplateFromPlan(deployments []models.DeploymentCreateRequest, plan domain.ServicePlan) (models.DeploymentCreateRequest, error) {
	for _, deployment := range deployments {
		if deployment.Name == plan.Name {
			return deployment, nil
		}
	}
	return models.DeploymentCreateRequest{}, fmt.Errorf("Could not find a Elasticsearch deployment template that matches Plan ID: %s", plan.ID)
}
