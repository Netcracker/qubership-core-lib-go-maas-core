package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/netcracker/qubership-core-lib-go-maas-client/v3/classifier"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTokenProvider struct {
	token string
}

func (m *mockTokenProvider) GetToken(_ context.Context) (string, error) {
	return m.token, nil
}

func (m *mockTokenProvider) ValidateToken(_ context.Context, _ string) (*jwt.Token, error) {
	return nil, nil
}

func (m *mockTokenProvider) GetClaimValue(_ *jwt.Token, _ string) (interface{}, error) {
	return nil, nil
}

func (m *mockTokenProvider) GetTokenAttribute(_ context.Context, _ string) (string, error) {
	return "", nil
}

func init() {
	serviceloader.Register(2, &security.DummyToken{})
}

// Test_GetHttpClient_HasNoOwnRetries pins that the shared client neither retries
// nor times out on its own - both would fight the maas client's retry loop.
func Test_GetHttpClient_HasNoOwnRetries(t *testing.T) {
	assertions := require.New(t)
	client := getHttpClient()

	assertions.Equal(0, client.RetryCount,
		"resty must not retry on its own, it multiplies the attempts of the maas client")
	assertions.Zero(client.GetClient().Timeout,
		"a client-wide timeout would cut the 60s topic watch long poll short; bounding a call is the caller's job via context")
}

func TestGetMaaSAgentUrl(t *testing.T) {
	testYamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "./testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(testYamlParams))

	assertions := require.New(t)
	maasAgentUrl := getMaaSAgentUrl()
	assertions.Equal("http://maas-agent:8080", maasAgentUrl)
}

func TestGetMaaSUrl(t *testing.T) {
	assertions := require.New(t)

	testYamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "./testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(testYamlParams))

	maasUrl := getMaaSUrl(getMaaSAgentUrl)
	assertions.Equal("http://maas-agent:8080", maasUrl())

	testConf := confmap.Provider(map[string]interface{}{
		"maas.internal.address": "http://maas-service:8080",
	}, ".")
	configloader.Init(configloader.YamlPropertySource(testYamlParams), &configloader.PropertySource{
		Provider: configloader.AsPropertyProvider(testConf),
	})
	assertions.Equal("http://maas-service:8080", maasUrl())

}
func TestGetTenantManagerUrl(t *testing.T) {
	testYamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "./testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(testYamlParams))

	assertions := require.New(t)
	tenantManagerUrl := getTenantManagerUrl()
	assertions.Equal("ws://tenant-manager:8080", tenantManagerUrl)
}

func TestGetNamespace(t *testing.T) {
	testYamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "./testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(testYamlParams))

	assertions := require.New(t)
	tenantManagerUrl := getNamespace()
	assertions.Equal("test-namespace", tenantManagerUrl)
}

func TestGetStompDialer(t *testing.T) {
	assertions := require.New(t)
	stompDialer := getStompDialer()
	assertions.NotNil(stompDialer)
	assertions.NotNil(stompDialer.TLSClientConfig)
}

func TestNewKafkaClient(t *testing.T) {
	testYamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "./testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(testYamlParams))

	assertions := require.New(t)
	client := NewKafkaClient()
	assertions.NotNil(client)
}

func TestNewRabbitClient(t *testing.T) {
	testYamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "./testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(testYamlParams))

	assertions := require.New(t)
	client := NewRabbitClient()
	assertions.NotNil(client)
}

func TestConfigure(t *testing.T) {
	testYamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "./testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(testYamlParams))

	assertions := require.New(t)
	testHttpClient := &resty.Client{}
	testNamespace := "custom-namespace"
	testMaaSUrl := "test.url"
	testDialer := &websocket.Dialer{}

	config := configure(
		WithHttpClient(testHttpClient),
		WithNamespace(testNamespace),
		WithMaaSAgentUrl(testMaaSUrl),
		WithStompDialer(testDialer),
	)
	assertions.NotNil(config)
	assertions.Equal(testHttpClient, config.httpClient())
	assertions.Equal(testNamespace, config.namespace())
	assertions.Equal(testMaaSUrl, config.maasAgentUrl())
	assertions.Equal(testDialer, config.stompDialer())
}

func TestNewKafkaClient_AuthIsInjectedByRestClient(t *testing.T) {
	const testToken = "test-m2m-token"
	serviceloader.Register(3, &mockTokenProvider{token: testToken})

	var receivedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	configloader.Init(&configloader.PropertySource{
		Provider: configloader.AsPropertyProvider(confmap.Provider(map[string]interface{}{
			"maas.agent.url":         server.URL,
			"microservice.namespace": "test-namespace",
		}, ".")),
	})

	kafkaClient := NewKafkaClient(WithNamespace("test-namespace"))
	topic, err := kafkaClient.GetTopic(context.Background(), classifier.Keys{classifier.Namespace: "test-namespace"})
	assert.Nil(t, topic)
	assert.NoError(t, err)
	assert.Equal(t, "Bearer "+testToken, receivedAuthHeader)
}

func TestIsK8sM2mEnabled(t *testing.T) {
	err := os.Setenv("KUBERNETES_M2M_ENABLED", "true")
	require.NoError(t, err)
	assert.True(t, isK8sM2mEnabled())

	err = os.Setenv("KUBERNETES_M2M_ENABLED", "false")
	require.NoError(t, err)
	assert.False(t, isK8sM2mEnabled())

	err = os.Setenv("KUBERNETES_M2M_ENABLED", "")
	require.NoError(t, err)
	assert.False(t, isK8sM2mEnabled())

	err = os.Setenv("KUBERNETES_M2M_ENABLED", "1")
	require.NoError(t, err)
	assert.True(t, isK8sM2mEnabled())

	err = os.Setenv("KUBERNETES_M2M_ENABLED", "0")
	require.NoError(t, err)
	assert.False(t, isK8sM2mEnabled())

	err = os.Unsetenv("KUBERNETES_M2M_ENABLED")
	require.NoError(t, err)
}

var maasAddressCases = []struct {
	name              string
	env               map[string]string
	withDirectAddress bool
	want              string
}{
	{name: "unset flag uses the agent", withDirectAddress: true, want: "agent"},
	{name: "false uses the agent", env: map[string]string{"KUBERNETES_M2M_ENABLED": "false"}, withDirectAddress: true, want: "agent"},
	{name: "true uses the direct address", env: map[string]string{"KUBERNETES_M2M_ENABLED": "true"}, withDirectAddress: true, want: "direct"},
	{name: "true without the direct address uses the agent", env: map[string]string{"KUBERNETES_M2M_ENABLED": "true"}, want: "agent"},
}

func TestNewKafkaClient_SelectsMaaSAddress(t *testing.T) {
	for _, tt := range maasAddressCases {
		t.Run(tt.name, func(t *testing.T) {
			requested := requestedMaaSServers(t, tt.env, tt.withDirectAddress, func() {
				_, _ = NewKafkaClient(WithHttpClient(resty.New())).GetTopic(context.Background(), classifier.Keys{classifier.Namespace: "test-namespace"})
			})
			assert.Equal(t, tt.want, requested)
		})
	}
}

func TestNewRabbitClient_SelectsMaaSAddress(t *testing.T) {
	for _, tt := range maasAddressCases {
		t.Run(tt.name, func(t *testing.T) {
			requested := requestedMaaSServers(t, tt.env, tt.withDirectAddress, func() {
				_, _ = NewRabbitClient(WithHttpClient(resty.New())).GetVhost(context.Background(), classifier.Keys{classifier.Namespace: "test-namespace"})
			})
			assert.Equal(t, tt.want, requested)
		})
	}
}

// requestedMaaSServers starts a maas-agent server and, with withDirectAddress, a MaaS server, sets env, runs call, and
// returns the names of the servers that received a request, joined by commas.
func requestedMaaSServers(t *testing.T, env map[string]string, withDirectAddress bool, call func()) string {
	var requested []string
	newServer := func(name string) *httptest.Server {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requested = append(requested, name)
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(server.Close)
		return server
	}
	config := map[string]interface{}{
		"maas.agent.url":         newServer("agent").URL,
		"microservice.namespace": "test-namespace",
	}
	if withDirectAddress {
		config["maas.internal.address"] = newServer("direct").URL
	}
	for name, value := range env {
		t.Setenv(name, value)
	}
	configloader.Init(&configloader.PropertySource{
		Provider: configloader.AsPropertyProvider(confmap.Provider(config, ".")),
	})
	call()
	return strings.Join(requested, ",")
}
