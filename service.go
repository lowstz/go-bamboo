package bamboo

const (
	BAMBOO_SERVICE_API_URI = "api/services"
)

type Service struct {
	Id  string `json:"id"`
	Acl string `json:"acl"`
}

func NewService(id, acl string) *Service {
	return &Service{
		Id:  id,
		Acl: acl,
	}
}

func (client *Client) HasService(name string) (bool, error) {
	allServices, err := client.AllServices()
	if err != nil {
		return false, err
	}
	for k, v := range allServices {
		if k == name {
			if v.Id == name {
				return true, nil
			}
		}
	}
	return false, nil
}

func (client *Client) AllServices() (map[string]*Service, error) {
	allServices := make(map[string]*Service)
	if err := client.apiGet(BAMBOO_SERVICE_API_URI, nil, &allServices); err != nil {
		return allServices, err
	}
	return allServices, nil
}

func (client *Client) CreateService(service *Service) (*Service, error) {
	var respService *Service
	if err := client.apiPost(BAMBOO_SERVICE_API_URI, service, &respService); err != nil {
		return respService, err
	}
	return respService, nil
}

func (client *Client) UpdateService(service *Service) (*Service, error) {
	var respService *Service
	if err := client.apiPut(BAMBOO_SERVICE_API_URI+service.Id, service, &respService); err != nil {
		return respService, err
	}
	return nil, nil
}

func (client *Client) DeleteService(name string) (*Service, error) {
	var respService *Service
	if err := client.apiDelete(BAMBOO_SERVICE_API_URI+name, nil, &respService); err != nil {
		return respService, err
	}
	return respService, nil
}
