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

	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7"
)

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

func CreateBrokerCredentials(id string, seed string) (username string, password string) {
	hashString := []byte(fmt.Sprintf("%s-%s", id, seed))
	sha1Bytes := sha1.Sum(hashString)
	password = hex.EncodeToString(sha1Bytes[:])
	return "pcf_broker", password
}

func CreateUserCredentials(id string, seed string) (username string, password string) {
	hashString := []byte(fmt.Sprintf("%s-%s", id, seed))
	sha1Bytes := sha1.Sum(hashString)
	password = hex.EncodeToString(sha1Bytes[:])
	username = id[0:10]
	return username, password
}

func UpdateBrokerPassword(client *elasticsearch.Client, newpassword string) (int, error) {
	body := strings.NewReader(fmt.Sprintf(`{"password": "%s", "roles": ["superuser"]}`, newpassword))
	res, err := client.Security.PutUser("pcf_broker", body)
	if err != nil {
		return 0, err
	}
	spew.Dump(res)
	statusCode := res.StatusCode
	return statusCode, nil
}

func CreateUserAccount(client *elasticsearch.Client, username string, password string) (int, error) {
	body := strings.NewReader(fmt.Sprintf(`{"password": "%s", "roles": ["superuser"]}`, password))
	res, err := client.Security.PutUser(username, body)
	if err != nil {
		return 0, err
	}
	statusCode := res.StatusCode
	return statusCode, nil
}

func DeleteUserAccount(client *elasticsearch.Client, username string) (int, error) {
	res, err := client.Security.DeleteUser(username)
	if err != nil {
		return 0, err
	}
	statusCode := res.StatusCode
	return statusCode, nil
}
