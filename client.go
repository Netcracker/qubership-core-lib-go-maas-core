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
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/utils"
)

var logger = logging.GetLogger("maas-client")

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
	maasUrl := config.maasAgentUrl()
	if configloader.GetKoanf().Bool("security.m2m.kubernetes.enabled") {
		maasUrl = config.maasUrl()
	}
	return kafka.NewClient(config.namespace(), maasUrl, config.tenantManagerUrl(), config.httpClient(),
		config.stompDialer(), config.authSupplier())
}

func NewRabbitClient(opts ...Option) rabbit.MaasClient {
	config := configure(opts...)
	maasUrl := config.maasAgentUrl()
	if configloader.GetKoanf().Bool("security.m2m.kubernetes.enabled") {
		maasUrl = config.maasUrl()
	}
	return rabbit.NewClient(config.namespace(), maasUrl, config.httpClient())
}

func configure(opts ...Option) *options {
	config := &options{
		namespace:        getNamespace,
		maasAgentUrl:     getMaaSAgentUrl,
		maasUrl:          getMaaSUrl(getMaaSAgentUrl),
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

func getMaaSUrl(fallbackUrl func() string) func() string {
	return func() string {
		maasUrl := configloader.GetOrDefaultString("maas.internal.address", "")
		if maasUrl == "" {
			logger.Warn("MaaS address is not available, falling back to maas-agent. Specify 'maas.internal.address' property to MaaS url")
			return fallbackUrl()
		}
		return maasUrl
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

func getHttpClient() *resty.Client {
	return resty.New().SetTransport(&m2mRoundTripper{rest.NewM2MRestClient()}).SetRetryCount(10)
}

func getStompDialer() *websocket.Dialer {
	return &websocket.Dialer{TLSClientConfig: utils.GetTlsConfig()}
}

func getAuthSupplier() func(ctx context.Context) (string, error) {
	tokenProvider := serviceloader.MustLoad[security.TokenProvider]()
	return tokenProvider.GetToken
}
