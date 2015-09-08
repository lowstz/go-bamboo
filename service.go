package bamboo

type Service struct {
	Id  string `json:"id"`
	Acl string `json:"acl"`
}

func (client *Client) HasService(name string) (bool, error) {

}

func (client *Client) AllServices() (map[string]*Service, error) {

}

func (client *Client) CreateService(service *Service) (*Service, error) {

}

func (client *Client) UpdateService(service *Service) (*Service, error) {

}

func (client *Client) DeleteService(name string) (string, error) {

}
