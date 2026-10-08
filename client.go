package core

import (
	"context"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/netcracker/qubership-core-lib-go-maas-client/v3/kafka"
	"github.com/netcracker/qubership-core-lib-go-maas-client/v3/rabbit"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	constants "github.com/netcracker/qubership-core-lib-go/v3/const"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go/v3/security/rest"
	"github.com/netcracker/qubership-core-lib-go/v3/security/tokensource"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/utils"
)

var logger = logging.GetLogger("maas-client")

const maasAddressProperty = "maas.internal.address"

type options struct {
	namespace        func() string
	maasAgentUrl     func() string
	maasUrl          func() string
	tenantManagerUrl func() string
	httpClient       func() *resty.Client
	stompDialer      func() *websocket.Dialer
	authSupplier     func() func(ctx context.Context) (string, error)
}

type Option func(options *options)

func NewKafkaClient(opts ...Option) kafka.MaasClient {
	config := configure(opts...)
	maasUrl := selectMaaSUrl(security.MustReadM2MAuthMode(), config)
	return kafka.NewClient(config.namespace(), maasUrl, config.tenantManagerUrl(), config.httpClient(),
		config.stompDialer(), config.authSupplier())
}

func NewRabbitClient(opts ...Option) rabbit.MaasClient {
	config := configure(opts...)
	maasUrl := selectMaaSUrl(security.MustReadM2MAuthMode(), config)
	return rabbit.NewClient(config.namespace(), maasUrl, config.httpClient())
}

func configure(opts ...Option) *options {
	config := &options{
		namespace:        getNamespace,
		maasAgentUrl:     getMaaSAgentUrl,
		maasUrl:          getMaaSUrl,
		tenantManagerUrl: getTenantManagerUrl,
		httpClient:       getHttpClient,
		stompDialer:      getStompDialer,
		authSupplier:     getAuthSupplier,
	}
	for _, option := range opts {
		option(config)
	}
	return config
}

func WithNamespace(namespace string) Option {
	return func(options *options) { options.namespace = func() string { return namespace } }
}

func WithMaaSAgentUrl(url string) Option {
	return func(options *options) { options.maasAgentUrl = func() string { return url } }
}

func WithMaaSUrl(url string) Option {
	return func(options *options) { options.maasUrl = func() string { return url } }
}

func WithHttpClient(client *resty.Client) Option {
	return func(options *options) { options.httpClient = func() *resty.Client { return client } }
}

func WithStompDialer(stompDialer *websocket.Dialer) Option {
	return func(options *options) { options.stompDialer = func() *websocket.Dialer { return stompDialer } }
}

func WithAuthSupplier(authSupplier func(ctx context.Context) (string, error)) Option {
	return func(options *options) {
		options.authSupplier = func() func(ctx context.Context) (string, error) { return authSupplier }
	}
}

func getMaaSAgentUrl() string {
	defaultUrl := constants.SelectUrl("http://maas-agent:8080", "https://maas-agent:8443")
	return configloader.GetOrDefaultString("maas.agent.url", defaultUrl)
}

func getMaaSUrl() string {
	return configloader.GetOrDefaultString(maasAddressProperty, "")
}

func selectMaaSUrl(mode security.M2MAuthMode, config *options) string {
	switch mode {
	case security.M2MAuthModeK8s:
		maasUrl := config.maasUrl()
		if maasUrl == "" {
			logger.Panic("%[1]s is not set: with M2M_AUTH_MODE=k8s the client sends requests directly to MaaS, set %[1]s to the MaaS URL", maasAddressProperty)
		}
		return maasUrl
	case security.M2MAuthModeHybrid:
		if maasUrl := config.maasUrl(); maasUrl != "" {
			return maasUrl
		}
		logger.Warn("MaaS address is not available, falling back to maas-agent. Specify '%s' property to MaaS url", maasAddressProperty)
		return config.maasAgentUrl()
	default:
		return config.maasAgentUrl()
	}
}

func getTenantManagerUrl() string {
	defaultUrl := constants.SelectUrl("ws://tenant-manager:8080", "wss://tenant-manager:8443")
	return configloader.GetOrDefaultString("tenant.manager.url", defaultUrl)
}

func getNamespace() string {
	return configloader.GetKoanf().MustString("microservice.namespace")
}

type m2mRoundTripper struct {
	client *rest.M2MRestClient
}

func (m *m2mRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.client.DoRequest(
		req.Context(),
		req.Method,
		req.URL.String(),
		req.Header,
		req.Body,
	)
}

// getHttpClient builds the resty client used by the maas clients.
//
// No retries: they belong to the maas client, and enabling both multiplies the
// attempts. No client-wide timeout: this client also serves the 60s topic watch
// long poll, so callers bound their calls via context.
func getHttpClient() *resty.Client {
	return resty.New().
		SetTransport(&m2mRoundTripper{rest.NewMaasRestClient()}).
		SetRetryCount(0)
}

func getStompDialer() *websocket.Dialer {
	return &websocket.Dialer{TLSClientConfig: utils.GetTlsConfig()}
}

func getAuthSupplier() func(ctx context.Context) (string, error) {
	switch security.MustReadM2MAuthMode() {
	case security.M2MAuthModeK8s:
		return func(ctx context.Context) (string, error) {
			return tokensource.GetAudienceToken(ctx, tokensource.AudienceNetcracker)
		}
	default:
		return serviceloader.MustLoad[security.TokenProvider]().GetToken
	}
}
