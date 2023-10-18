package option

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	credentials "github.com/oras-project/oras-credentials-go"
	"golang.org/x/exp/slog"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/koolay/oras-sdk/credential"
	onet "github.com/koolay/oras-sdk/net"
)

// Remote options struct.
type Remote struct {
	DistributionSpec
	CACertFilePath    string
	Insecure          bool
	Configs           []string
	Username          string
	PasswordFromStdin bool
	Password          string

	resolveFlag           []string
	applyDistributionSpec bool
	headerFlags           []string
	headers               http.Header
	warned                map[string]*sync.Map
	plainHTTP             func() (plainHTTP bool, enforced bool)
}

func NewRemote(plainHTTP bool, username, password string) Remote {
	return Remote{
		Username: username,
		Password: password,
		plainHTTP: func() (bool, bool) {
			return plainHTTP, plainHTTP
		},
	}
}

// EnableDistributionSpecFlag set distribution specification flag as applicable.
func (opts *Remote) EnableDistributionSpecFlag() {
	opts.applyDistributionSpec = true
}

func applyPrefix(prefix, description string) (flagPrefix, notePrefix string) {
	if prefix == "" {
		return "", ""
	}
	return prefix + "-", description + " "
}

// Parse tries to read password with optional cmd prompt.
func (opts *Remote) Parse() error {
	if err := opts.parseCustomHeaders(); err != nil {
		return err
	}
	if err := opts.readPassword(); err != nil {
		return err
	}
	return opts.DistributionSpec.Parse()
}

// readPassword tries to read password with optional cmd prompt.
func (opts *Remote) readPassword() (err error) {
	if opts.Password != "" {
		fmt.Fprintln(
			os.Stderr,
			"WARNING! Using --password via the CLI is insecure. Use --password-stdin.",
		)
	} else if opts.PasswordFromStdin {
		// Prompt for credential
		password, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		opts.Password = strings.TrimSuffix(string(password), "\n")
		opts.Password = strings.TrimSuffix(opts.Password, "\r")
	}
	return nil
}

// parseResolve parses resolve flag.
func (opts *Remote) parseResolve(baseDial onet.DialFunc) (onet.DialFunc, error) {
	if len(opts.resolveFlag) == 0 {
		return baseDial, nil
	}

	formatError := func(param, message string) error {
		return fmt.Errorf("failed to parse resolve flag %q: %s", param, message)
	}
	var dialer onet.Dialer
	for _, r := range opts.resolveFlag {
		parts := strings.SplitN(r, ":", 4)
		length := len(parts)
		if length < 3 {
			return nil, formatError(r, "expecting host:port:address[:address_port]")
		}
		host := parts[0]
		hostPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, formatError(r, "expecting uint64 host port")
		}
		// ipv6 zone is not parsed
		address := net.ParseIP(parts[2])
		if address == nil {
			return nil, formatError(r, "invalid IP address")
		}
		addressPort := hostPort
		if length > 3 {
			addressPort, err = strconv.Atoi(parts[3])
			if err != nil {
				return nil, formatError(r, "expecting uint64 address port")
			}
		}
		dialer.Add(host, hostPort, address, addressPort)
	}
	dialer.BaseDialContext = baseDial
	return dialer.DialContext, nil
}

// tlsConfig assembles the tls config.
func (opts *Remote) tlsConfig() (*tls.Config, error) {
	config := &tls.Config{
		InsecureSkipVerify: opts.Insecure,
	}
	return config, nil
}

// authClient assembles a oras auth client.
func (opts *Remote) authClient(registry string, debug bool) (client *auth.Client, err error) {
	config, err := opts.tlsConfig()
	if err != nil {
		return nil, err
	}
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.TLSClientConfig = config
	dialContext, err := opts.parseResolve(baseTransport.DialContext)
	if err != nil {
		return nil, err
	}
	baseTransport.DialContext = dialContext
	client = &auth.Client{
		Client: &http.Client{
			// http.RoundTripper with a retry using the DefaultPolicy
			// see: https://pkg.go.dev/oras.land/oras-go/v2/registry/remote/retry#Policy
			Transport: retry.NewTransport(baseTransport),
		},
		Cache:  auth.NewCache(),
		Header: opts.headers,
	}
	if debug {
		client.Client.Transport = newTransport(client.Client.Transport)
	}

	cred := opts.Credential()
	if cred != auth.EmptyCredential {
		client.Credential = func(ctx context.Context, s string) (auth.Credential, error) {
			return cred, nil
		}
	} else {
		store, err := credential.NewStore(opts.Configs...)
		if err != nil {
			return nil, err
		}
		client.Credential = credentials.Credential(store)
	}
	return
}

func (opts *Remote) parseCustomHeaders() error {
	if len(opts.headerFlags) != 0 {
		headers := map[string][]string{}
		for _, h := range opts.headerFlags {
			name, value, found := strings.Cut(h, ":")
			if !found || strings.TrimSpace(name) == "" {
				// In conformance to the RFC 2616 specification
				// Reference: https://www.rfc-editor.org/rfc/rfc2616#section-4.2
				return fmt.Errorf("invalid header: %q", h)
			}
			headers[name] = append(headers[name], value)
		}
		opts.headers = headers
	}
	return nil
}

// Credential returns a credential based on the remote options.
func (opts *Remote) Credential() auth.Credential {
	return credential.Credential(opts.Username, opts.Password)
}

func (opts *Remote) handleWarning(
	registry string,
	logger *slog.Logger,
) func(warning remote.Warning) {
	if opts.warned == nil {
		opts.warned = make(map[string]*sync.Map)
	}
	warned := opts.warned[registry]
	if warned == nil {
		warned = &sync.Map{}
		opts.warned[registry] = warned
	}
	logger = logger.With("registry", registry)
	return func(warning remote.Warning) {
		if _, loaded := warned.LoadOrStore(warning.WarningValue, struct{}{}); !loaded {
			logger.Warn(warning.Text)
		}
	}
}

// NewRegistry assembles a oras remote registry.
func (opts *Remote) NewRegistry(
	registry string,
	common Common,
	logger *slog.Logger,
) (reg *remote.Registry, err error) {
	reg, err = remote.NewRegistry(registry)
	if err != nil {
		return nil, err
	}
	registry = reg.Reference.Registry
	reg.PlainHTTP = opts.isPlainHttp(registry)
	reg.HandleWarning = opts.handleWarning(registry, logger)
	if reg.Client, err = opts.authClient(registry, common.Debug); err != nil {
		return nil, err
	}
	return
}

// NewRepository assembles a oras remote repository.
func (opts *Remote) NewRepository(
	reference string,
	common Common,
	logger *slog.Logger,
) (repo *remote.Repository, err error) {
	repo, err = remote.NewRepository(reference)
	if err != nil {
		return nil, err
	}
	registry := repo.Reference.Registry
	repo.PlainHTTP = opts.isPlainHttp(registry)
	repo.HandleWarning = opts.handleWarning(registry, logger)
	if repo.Client, err = opts.authClient(registry, common.Debug); err != nil {
		return nil, err
	}
	repo.SkipReferrersGC = true
	if opts.ReferrersAPI != nil {
		if err := repo.SetReferrersCapability(*opts.ReferrersAPI); err != nil {
			return nil, err
		}
	}
	return
}

// isPlainHttp returns the plain http flag for a given registry.
func (opts *Remote) isPlainHttp(registry string) bool {
	plainHTTP, enforced := opts.plainHTTP()
	if enforced {
		return plainHTTP
	}
	host, _, _ := net.SplitHostPort(registry)
	if host == "localhost" || registry == "localhost" {
		// not specified, defaults to plain http for localhost
		return true
	}
	return plainHTTP
}
