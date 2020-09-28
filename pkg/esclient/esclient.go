/*
Package esclient is used to create a client that communicates directly with a single Elasticsearch instance
The package itself is a wrapper around github.com/elastic/go-elasticsearch
*/
package esclient

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
)

// CreateV7Client returns a new elasticsearch client
func CreateV7Client(address string, username string, password string) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			address,
		},
		Username: username,
		Password: password,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
			TLSClientConfig: &tls.Config{
				MaxVersion:         tls.VersionTLS11,
				InsecureSkipVerify: true,
			},
		},
	}
	es, err := elasticsearch.NewClient(cfg)
	return es, err
}

// CreateBrokerCredentials is used to recreate a set of credentials that is unique to each cluster, based on a combination
// of the InstanceID and a configured seed
func CreateBrokerCredentials(id string, seed string) (username string, password string) {
	hashString := []byte(fmt.Sprintf("%s-%s", id, seed))
	sha1Bytes := sha1.Sum(hashString)
	password = hex.EncodeToString(sha1Bytes[:])
	return "pcf_broker", password
}

// CreateUserCredentials is used to recreate a set of credentials that is unique to each cluster and user, based
// on a combination of the BindID/AppGUID and a configured seed
func CreateUserCredentials(id string, seed string) (username string, password string) {
	hashString := []byte(fmt.Sprintf("%s-%s", id, seed))
	sha1Bytes := sha1.Sum(hashString)
	password = hex.EncodeToString(sha1Bytes[:])
	username = id[0:10]
	return username, password
}

// UpdateBrokerPassword is used in case the current BrokerPassword is incorrect. If a deployment is brand new or
// a user has tried to reset the password for the broker, it will update the account again to ensure correct password is set
func UpdateBrokerPassword(client *elasticsearch.Client, newpassword string) (int, error) {
	body := strings.NewReader(fmt.Sprintf(`{"password": "%s", "roles": ["superuser"]}`, newpassword))
	res, err := client.Security.PutUser("pcf_broker", body)
	if err != nil {
		return 0, err
	}
	statusCode := res.StatusCode
	return statusCode, nil
}

// CreateUserAccount is used to create the account defined in a Bind operation
func CreateUserAccount(client *elasticsearch.Client, username string, password string) (int, error) {
	body := strings.NewReader(fmt.Sprintf(`{"password": "%s", "roles": ["superuser"]}`, password))
	res, err := client.Security.PutUser(username, body)
	if err != nil {
		return 0, err
	}
	statusCode := res.StatusCode
	return statusCode, nil
}

// DeleteUserAccount is used to delete a user account defined in a Unbind operation
func DeleteUserAccount(client *elasticsearch.Client, username string) (int, error) {
	res, err := client.Security.DeleteUser(username)
	if err != nil {
		return 0, err
	}
	statusCode := res.StatusCode
	return statusCode, nil
}
