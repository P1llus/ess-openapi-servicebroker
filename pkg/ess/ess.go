/*
Package ess is used to communicate with the Elastic Cloud API: https://www.elastic.co/guide/en/cloud/current/ec-restful-api.html
The package itself is a wrapper around github.com/elastic/cloud-sdk-go
*/
package ess

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/elastic/cloud-sdk-go/pkg/api"
	"github.com/elastic/cloud-sdk-go/pkg/api/deploymentapi"
	"github.com/elastic/cloud-sdk-go/pkg/models"
)

// ResetElasticPasswordResponse struct is used to Marshal password reset responses from the Elastic Cloud API
type ResetElasticPasswordResponse struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateDeployment is a wrapper around deploymentapi.Create to work with the servicebroker
// It will try to create a new cluster defined by the data body
func CreateDeployment(client *api.API, data *models.DeploymentCreateRequest, requestid string) (*models.DeploymentCreateResponse, error) {
	res, err := deploymentapi.Create(deploymentapi.CreateParams{API: client, Request: data, RequestID: requestid})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	return res, nil
}

// DeleteDeployment is a wrapper around deploymentapi.Delete to work with the servicebroker
// It will try to delete an existing cluster defined by the id parameter
func DeleteDeployment(client *api.API, id string) (*models.DeploymentDeleteResponse, error) {
	res, err := deploymentapi.Delete(deploymentapi.DeleteParams{API: client, DeploymentID: id})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	return res, nil
}

// ListDeployments is a wrapper around deploymentapi.List to work with the servicebroker
// This function will return a list of all active deployments for the authenticated account
func ListDeployments(api *api.API) (*models.DeploymentsListResponse, error) {
	res, err := deploymentapi.List(deploymentapi.ListParams{API: api})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	return res, nil
}

// GetDeployment is a wrapper around deploymentapi.Get to work with the servicebroker
// This function returns a single deployment specified by the id parameter
func GetDeployment(api *api.API, id string) (*models.DeploymentGetResponse, error) {
	res, err := deploymentapi.Get(deploymentapi.GetParams{API: api, DeploymentID: id})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	return res, nil
}

// GetKibana is a wrapper around deploymentapi.GetKibana to work with the servicebroker
// This function returns a single Kibana instance specified by the id parameter
func GetKibana(api *api.API, id string) (*models.KibanaResourceInfo, error) {
	res, err := deploymentapi.GetKibana(deploymentapi.GetParams{API: api, DeploymentID: id})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return res, nil
}

// GetApm is a wrapper around deploymentapi.GetApm to work with the servicebroker
// This function returns a single APM instance specified by the id parameter
func GetApm(api *api.API, id string) (*models.ApmResourceInfo, error) {
	res, err := deploymentapi.GetApm(deploymentapi.GetParams{API: api, DeploymentID: id})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return res, nil
}

// GetAppSearch is a wrapper around deploymentapi.GetAppSearch to work with the servicebroker
// This function returns a single AppSearch instance specified by the id parameter
func GetAppSearch(api *api.API, id string, refid string) (*models.AppSearchResourceInfo, error) {
	res, err := deploymentapi.GetAppSearch(deploymentapi.GetParams{API: api, DeploymentID: id, RefID: refid})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return res, nil
}

// GetElasticsearch is a wrapper around deploymentapi.GetElasticsearch to work with the servicebroker
// This function returns a single Elasticsearch instance specified by the id parameter
func GetElasticsearch(api *api.API, id string, refid string) (*models.ElasticsearchResourceInfo, error) {
	res, err := deploymentapi.GetElasticsearch(deploymentapi.GetParams{API: api, DeploymentID: id, RefID: refid})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return res, nil
}

// ShutdownDeployment is a wrapper around deploymentapi.Shutdown to work with the servicebroker
// This function shuts down the whole deployment instance specified by the id parameter
func ShutdownDeployment(api *api.API, id string) error {
	_, err := deploymentapi.Shutdown(deploymentapi.ShutdownParams{API: api, DeploymentID: id})
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// SearchDeployments is a wrapper around deploymentapi.Search to work with the servicebroker
// This functions searches all available deployments for a cluster with the name specified by the name parameter
func SearchDeployments(api *api.API, name string) (*models.DeploymentSearchResponse, error) {
	search := createQuery(api, name)
	res, err := deploymentapi.Search(search)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if len(res.Deployments) == 0 {
		emptyResponse := &models.DeploymentSearchResponse{}
		return emptyResponse, fmt.Errorf("no deployment find matching the instace ID %s", name)
	}

	return res.Deployments[0], nil
}

func createQuery(api *api.API, name string) deploymentapi.SearchParams {
	fullQuery := fmt.Sprintf("name: %s", name)
	return deploymentapi.SearchParams{API: api, Request: &models.SearchRequest{Query: &models.QueryContainer{QueryString: &models.QueryStringQuery{Query: &fullQuery}}}}
}

// GetServiceURL returns the full Endpoint URL for the Elasticsearch instance from the
// related deployment specified by the deployment parameter
func GetServiceURL(api *api.API, deployment *models.DeploymentSearchResponse) string {
	endpoint := deployment.Resources.Elasticsearch[0].Info.Metadata.Endpoint
	port := deployment.Resources.Elasticsearch[0].Info.Metadata.Ports.HTTPS
	serviceURL := fmt.Sprintf("https://%s:%d", endpoint, *port)
	return serviceURL
}

// ResetElasticUserPassword tries to reset the password for the "elastic" user for the related deploymentID
// Will return the new password upon success
func ResetElasticUserPassword(endpoint string, version string, apiKey string, deploymentID string) string {
	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/%s/deployments/%s/elasticsearch/main-elasticsearch/_reset-password", endpoint, version, deploymentID), nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("ApiKey %s", apiKey))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}
	var r ResetElasticPasswordResponse
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal([]byte(body), &r)
	if err != nil {
		fmt.Println(err)
	}
	return r.Password
}

// DeploymentStatus iterates over all products and services in a single deployment and returns true if
// all components have the status defined by the status parameter
func DeploymentStatus(deployment *models.DeploymentGetResponse, status string) bool {
	for _, es := range deployment.Resources.Elasticsearch {
		if *es.Info.Status != status {
			return false
		}
	}
	for _, kib := range deployment.Resources.Kibana {
		if *kib.Info.Status != status {
			return false
		}
	}
	for _, ents := range deployment.Resources.EnterpriseSearch {
		if *ents.Info.Status != status {
			return false
		}
	}
	for _, app := range deployment.Resources.Appsearch {
		if *app.Info.Status != status {
			return false
		}
	}
	for _, apm := range deployment.Resources.Apm {
		if *apm.Info.Status != status {
			return false
		}
	}
	return true
}
