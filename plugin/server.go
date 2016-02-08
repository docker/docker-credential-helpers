package plugin

import "github.com/calavera/docker-credential-helpers/credentials"

// CredentialsGetResponse holds the information sent to docker after
// a request for credentials.
type CredentialsGetResponse struct {
	Error    string
	Username string
	Password string
}

func (p *credentialsPlugin) Get(c *credentials.Credentials, resp *CredentialsGetResponse) error {
	username, password, err := p.helper.Get(c.ServerURL)
	if err != nil {
		*resp = CredentialsGetResponse{
			Error: err.Error(),
		}
		return nil
	}

	*resp = CredentialsGetResponse{
		Username: username,
		Password: password,
	}
	return nil
}

func (p *credentialsPlugin) Add(c *credentials.Credentials, resp *string) error {
	err := p.helper.Add(c)
	if err != nil {
		*resp = err.Error()
	}
	return nil
}

func (p *credentialsPlugin) Delete(c *credentials.Credentials, resp *string) error {
	err := p.helper.Delete(c.ServerURL)
	if err != nil {
		*resp = err.Error()
	}
	return nil
}
