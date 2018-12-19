package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/david972/kubectl-login/auth"
	"github.com/david972/kubectl-login/kubeconfig"
	flags "github.com/jessevdk/go-flags"
	homedir "github.com/mitchellh/go-homedir"
)

// Parse parses command line arguments and returns a CLI instance.
func Parse(osArgs []string, version string) (*CLI, error) {
	var cli CLI
	parser := flags.NewParser(&cli, flags.HelpFlag)
	parser.LongDescription = fmt.Sprintf(`Version %s
		This updates the kubeconfig for Kubernetes OpenID Connect (OIDC) authentication.`,
		version)
	args, err := parser.ParseArgs(osArgs[1:])
	if err != nil {
		return nil, err
	}
	if len(args) > 0 {
		return nil, fmt.Errorf("Too many argument")
	}
	return &cli, nil
}

// CLI represents an interface of this command.
type CLI struct {
	KubeConfig      string `long:"kubeconfig" default:"~/.kube/config" env:"KUBECONFIG" description:"Path to the kubeconfig file"`
	ListenPort      int    `long:"listen-port" default:"8000" env:"KUBELOGIN_LISTEN_PORT" description:"Port used by kubelogin to bind its webserver"`
	SkipTLSVerify   bool   `long:"insecure-skip-tls-verify" env:"KUBELOGIN_INSECURE_SKIP_TLS_VERIFY" description:"If set, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure"`
	SkipOpenBrowser bool   `long:"skip-open-browser" env:"KUBELOGIN_SKIP_OPEN_BROWSER" description:"If set, it does not open the browser on authentication."`
}

// ExpandKubeConfig returns an expanded KubeConfig path.
func (c *CLI) ExpandKubeConfig() (string, error) {
	d, err := homedir.Expand(c.KubeConfig)
	if err != nil {
		return "", fmt.Errorf("Could not expand %s: %s", c.KubeConfig, err)
	}
	return d, nil
}

// Run performs this command.
func (c *CLI) Run(ctx context.Context) error {
	log.Printf("Reading %s", c.KubeConfig)
	path, err := c.ExpandKubeConfig()
	if err != nil {
		return err
	}
	cfg, err := kubeconfig.Read(path)
	if err != nil {
		return fmt.Errorf("Could not read kubeconfig: %s", err)
	}
	log.Printf("Using current-context: %s", cfg.CurrentContext)
	authProvider, err := kubeconfig.FindOIDCAuthProvider(cfg)
	if err != nil {
		return fmt.Errorf(`Could not find OIDC configuration in kubeconfig: %s
			Did you setup kubectl for OIDC authentication?
				kubectl config set-credentials %s \
					--auth-provider oidc \
					--auth-provider-arg idp-issuer-url=https://issuer.example.com \
					--auth-provider-arg client-id=YOUR_CLIENT_ID \
					--auth-provider-arg client-secret=YOUR_CLIENT_SECRET`,
			err, cfg.CurrentContext)
	}
	tlsConfig := c.tlsConfig(authProvider)
	authConfig := &auth.Config{
		Issuer:          authProvider.IDPIssuerURL(),
		ClientID:        authProvider.ClientID(),
		ClientSecret:    authProvider.ClientSecret(),
		ExtraScopes:     authProvider.ExtraScopes(),
		Client:          &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}},
		LocalServerPort: c.ListenPort,
		SkipOpenBrowser: c.SkipOpenBrowser,
	}
	fmt.Printf("authConfig = %v\n", authConfig)
	token, err := authConfig.GetTokenSet(ctx)
	if err != nil {
		return fmt.Errorf("Could not get token from OIDC provider: %s", err)
	}

	authProvider.SetIDToken(token.IDToken)
	authProvider.SetRefreshToken(token.RefreshToken)
	kubeconfig.Write(cfg, path)
	log.Printf("Updated %s", c.KubeConfig)
	return nil
}
