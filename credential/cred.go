package credential

import (
	credentials "github.com/oras-project/oras-credentials-go"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// NewStore generates a store based on the passed-in config file paths.
func NewStore(configPaths ...string) (credentials.Store, error) {
	opts := credentials.StoreOptions{AllowPlaintextPut: true}
	if len(configPaths) == 0 {
		// use default docker config file path
		return credentials.NewStoreFromDocker(opts)
	}

	var stores []credentials.Store
	for _, config := range configPaths {
		store, err := credentials.NewStore(config, opts)
		if err != nil {
			return nil, err
		}
		stores = append(stores, store)
	}
	return credentials.NewStoreWithFallbacks(stores[0], stores[1:]...), nil
}

// Credential converts user input username and password to a credential.
func Credential(username, password string) auth.Credential {
	if username == "" {
		return auth.Credential{
			RefreshToken: password,
		}
	}
	return auth.Credential{
		Username: username,
		Password: password,
	}
}
