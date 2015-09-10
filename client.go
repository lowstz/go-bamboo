package bamboo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lowstz/httpcluster"
)

const (
	HTTP_GET    = "GET"
	HTTP_PUT    = "PUT"
	HTTP_DELETE = "DELETE"
	HTTP_POST   = "POST"
)

const (
	BAMBOO_HEALTH_CHECK_URI     = ""
	HTTP_CLIENT_REQUEST_TIMEOUT = 20
)

type Bamboo interface {
	/* -- Service -- */

	// check it see if a application exists
	HasService(name string) (bool, error)
	// get a list of service from bamboo
	AllServices() (map[string]*Service, error)
	// create a service in bamboo
	CreateService(service *Service) (*Service, error)
	// update a service
	UpdateService(service *Service) (*Service, error)
	// delete a service
	DeleteService(name string) (*Service, error)
}

var (
	// the url specified was invalid
	ErrInvalidEndpoint = errors.New("Invalid Marathon endpoint specified")
	// invalid or error response from bamboo
	ErrInvalidResponse = errors.New("Invalid response from bamboo")
	// some resource does not exists
	ErrDoesNotExist = errors.New("The resource does not exist")
	// all the bamboo endpoints are down
	ErrBambooDown = errors.New("All the Marathon hosts are presently down")
	// unable to decode the response
	ErrInvalidResult = errors.New("Unable to decode the response from bamboo")
	// invalid argument
	ErrInvalidArgument = errors.New("The argument passed is invalid")
	// error return by marathon
	ErrBambooError = errors.New("bamboo error")
	// the operation has timed out
	ErrTimeoutError = errors.New("The operation has timed out")
)

type Client struct {
	sync.RWMutex
	// bamboo url
	url string
	// the http client
	http *http.Client
	// the output for the logger
	logger *log.Logger
	// the bamboo http cluster
	cluster httpcluster.Cluster
}

type Message struct {
	Message string `json:"message"`
}

func NewClient(config Config) (Bamboo, error) {
	if cluster, err := httpcluster.NewHttpCluster(config.URL, BAMBOO_HEALTH_CHECK_URI); err != nil {
		return nil, err
	} else {
		service := new(Client)
		service.url = config.URL
		if config.LogOutput == nil {
			config.LogOutput = ioutil.Discard
		}
		service.logger = log.New(config.LogOutput, "[debug] ", log.LstdFlags|log.Lshortfile)
		service.cluster = cluster
		service.http = &http.Client{
			Timeout: (time.Duration(HTTP_CLIENT_REQUEST_TIMEOUT) * time.Second),
		}
		return service, nil
	}
}
func (client *Client) marshallJSON(data interface{}) (string, error) {
	if response, err := json.Marshal(data); err != nil {
		return "", err
	} else {
		return string(response), err
	}
}

func (client *Client) unMarshallDataToJson(stream io.Reader, result interface{}) error {
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(result); err != nil {
		return err
	}
	return nil
}

func (client *Client) unmarshallJsonArray(stream io.Reader, results []interface{}) error {
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(results); err != nil {
		return err
	}
	return nil
}

func (client *Client) apiPostData(data interface{}) (string, error) {
	if data == nil {
		return "", nil
	}
	content, err := client.marshallJSON(data)
	if err != nil {
		return "", err
	}
	return content, nil
}

func (client *Client) apiGet(uri string, post, result interface{}) error {
	if content, err := client.apiPostData(post); err != nil {
		return err
	} else {
		_, _, error := client.apiCall(HTTP_GET, uri, content, result)
		return error
	}
}

func (client *Client) apiPut(uri string, post, result interface{}) error {
	if content, err := client.apiPostData(post); err != nil {
		return err
	} else {
		_, _, error := client.apiCall(HTTP_PUT, uri, content, result)
		return error
	}
}

func (client *Client) apiPost(uri string, post, result interface{}) error {
	if content, err := client.apiPostData(post); err != nil {
		return err
	} else {
		_, _, error := client.apiCall(HTTP_POST, uri, content, result)
		return error
	}
}

func (client *Client) apiDelete(uri string, post, result interface{}) error {
	if content, err := client.apiPostData(post); err != nil {
		return err
	} else {
		_, _, error := client.apiCall(HTTP_DELETE, uri, content, result)
		return error
	}
}

func (client *Client) apiCall(method, uri, body string, result interface{}) (int, string, error) {
	client.log("apiCall() method: %s, uri: %s, body: %s", method, uri, body)
	if status, content, _, err := client.httpCall(method, uri, body); err != nil {
		return 0, "", err
	} else {
		client.log("apiCall() status: %d, content: %s\n", status, content)
		if status >= 200 && status <= 299 {
			if result != nil {
				if err := client.unMarshallDataToJson(strings.NewReader(content), result); err != nil {
					client.log("apiCall(): failed to unmarshall the response from bamboo, error: %s", err)
					return status, content, ErrInvalidResponse
				}
			}
			client.log("apiCall() result: %+v", result)
			return status, content, nil
		}
		switch status {
		case 500:
			return status, "", ErrInvalidResponse
		case 404:
			return status, "", ErrDoesNotExist
		}

		/* step: lets decode into a error message */
		var message Message
		if err := client.unMarshallDataToJson(strings.NewReader(content), &message); err != nil {
			return status, content, ErrInvalidResponse
		} else {
			errorMessage := "unknown error"
			if message.Message != "" {
				errorMessage = message.Message
			}
			return status, message.Message, errors.New(errorMessage)
		}
	}
}

func (client *Client) httpCall(method, uri, body string) (int, string, *http.Response, error) {
	/* step: get a member from the cluster */
	if member, err := client.cluster.GetMember(); err != nil {
		return 0, "", nil, err
	} else {
		url := fmt.Sprintf("%s/%s", member, uri)
		client.log("httpCall(): %s, uri: %s, url: %s", method, uri, url)

		if request, err := http.NewRequest(method, url, strings.NewReader(body)); err != nil {
			return 0, "", nil, err
		} else {
			request.Header.Add("Content-Type", "application/json")
			request.Header.Add("Accept", "application/json")
			var content string
			/* step: perform the request */
			if response, err := client.http.Do(request); err != nil {
				/* step: mark the endpoint as down */
				client.cluster.MarkDown()
				/* step: retry the request with another endpoint */
				return client.httpCall(method, uri, body)
			} else {
				/* step: lets read in any content */
				client.log("httpCall: %s, uri: %s, url: %s\n", method, uri, url)
				if response.ContentLength != 0 {
					/* step: read in the content from the request */
					responseContent, err := ioutil.ReadAll(response.Body)
					if err != nil {
						return response.StatusCode, "", response, err
					}
					content = string(responseContent)
				}
				/* step: return the request */
				return response.StatusCode, content, response, nil
			}
		}
	}
}

func (client *Client) log(message string, args ...interface{}) {
	client.logger.Printf(message+"\n", args...)
}
