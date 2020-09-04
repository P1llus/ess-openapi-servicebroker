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

type ResetElasticPasswordResponse struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Create deployment
func CreateDeployment(client *api.API, data *models.DeploymentCreateRequest, requestid string) (*models.DeploymentCreateResponse, error) {
	res, err := deploymentapi.Create(deploymentapi.CreateParams{API: client, Request: data, RequestID: requestid})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	return res, nil
}

// Delete deployment
func DeleteDeployment(client *api.API, id string) (*models.DeploymentDeleteResponse, error) {
	res, err := deploymentapi.Delete(deploymentapi.DeleteParams{API: client, DeploymentID: id})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	return res, nil
}

// List all deployments
func ListDeployments(api *api.API) (*models.DeploymentsListResponse, error) {
	res, err := deploymentapi.List(deploymentapi.ListParams{API: api})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	return res, nil
}

// Get specific deployment
func GetDeployment(api *api.API, id string) (*models.DeploymentGetResponse, error) {
	res, err := deploymentapi.Get(deploymentapi.GetParams{API: api, DeploymentID: id})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	return res, nil
}

// Get specific Kibana
func GetKibana(api *api.API, id string, refid string) (*models.KibanaResourceInfo, error) {
	res, err := deploymentapi.GetKibana(deploymentapi.GetParams{API: api, DeploymentID: id, RefID: refid})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return res, nil
}

// Get specific APM
func GetApm(api *api.API, id string, refid string) (*models.ApmResourceInfo, error) {
	res, err := deploymentapi.GetApm(deploymentapi.GetParams{API: api, DeploymentID: id, RefID: refid})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return res, nil
}

// Get specific AppSearch
func GetAppSearch(api *api.API, id string, refid string) (*models.AppSearchResourceInfo, error) {
	res, err := deploymentapi.GetAppSearch(deploymentapi.GetParams{API: api, DeploymentID: id, RefID: refid})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return res, nil
}

// Get specific Elasticsearch
func GetElastisearch(api *api.API, id string, refid string) (*models.ElasticsearchResourceInfo, error) {
	res, err := deploymentapi.GetElasticsearch(deploymentapi.GetParams{API: api, DeploymentID: id, RefID: refid})
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return res, nil
}

// Get specific Elasticsearch
func ShutdownDeployment(api *api.API, id string) error {
	_, err := deploymentapi.Shutdown(deploymentapi.ShutdownParams{API: api, DeploymentID: id})
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// Returns the deploymentID of a specific deployment based on name
func SearchDeployments(api *api.API, name string) (*models.DeploymentSearchResponse, error) {
	search := CreateQuery(api, name)
	res, err := deploymentapi.Search(search)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if len(res.Deployments) == 0 {
		emptyResponse := &models.DeploymentSearchResponse{}
		return emptyResponse, fmt.Errorf("No deployment find matching the instace ID %s", name)
	}

	return res.Deployments[0], nil
}

// Prepares the search parameters to search for a specific deployment
func CreateQuery(api *api.API, name string) deploymentapi.SearchParams {
	fullQuery := fmt.Sprintf("name: %s", name)
	return deploymentapi.SearchParams{API: api, Request: &models.SearchRequest{Query: &models.QueryContainer{QueryString: &models.QueryStringQuery{Query: &fullQuery}}}}
}

// Prepares the search parameters to search for a specific deployment
func GetServiceURL(api *api.API, deployment *models.DeploymentSearchResponse) string {
	endpoint := deployment.Resources.Elasticsearch[0].Info.Metadata.Endpoint
	port := deployment.Resources.Elasticsearch[0].Info.Metadata.Ports.HTTPS
	serviceUrl := fmt.Sprintf("https://%s:%d", endpoint, *port)
	return serviceUrl
}

// Prepares the search parameters to search for a specific deployment
func ResetElasticUserPassword(endpoint string, version string, apiKey string, deploymentID string) string {
	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/%s/deployments/%s/elasticsearch/main-elasticsearch/_reset-password", endpoint, version, deploymentID), nil)
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
	return r.Password
}

// Check if a ongoing deployment is finished
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
