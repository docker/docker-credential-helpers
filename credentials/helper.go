package credentials

// Credentials holds the information shared between docker and the credentials store.
type Credentials struct {
	ServerURL string
	Username  string
	Password  string
}

// Helper is the interface a credentials store helper must implement.
type Helper interface {
	Add(*Credentials) error
	Delete(serverURL string) error
	Get(serverURL string) (string, string, error)
}
