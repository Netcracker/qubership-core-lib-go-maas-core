package core

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/jarcoal/httpmock"
	"github.com/netcracker/qubership-core-lib-go-maas-client/v3/classifier"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	serviceloader.Register(2, &security.DummyToken{})
}

func TestGetMaaSAgentUrl(t *testing.T) {
	testYamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "./testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(testYamlParams))

	assertions := require.New(t)
	maasAgentUrl := getMaaSAgentUrl()
	assertions.Equal("http://maas-agent:8080", maasAgentUrl)
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

func TestNewKafkaClient_WithAthSupplier(t *testing.T) {
	client := resty.New()
	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", "/api/v1/kafka/topic/get-by-classifier",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	isAuthSupplierCalled := false
	kafkaClient := NewKafkaClient(WithAuthSupplier(func(ctx context.Context) (string, error) {
		isAuthSupplierCalled = true
		return "test-token", nil
	}),
		WithHttpClient(client),
		WithNamespace("test-namespace"),
	)
	topic, err := kafkaClient.GetTopic(context.Background(), classifier.Keys{classifier.Namespace: "test-namespace"})
	assert.Nil(t, topic)
	assert.NoError(t, err)
	assertions := require.New(t)
	assertions.True(isAuthSupplierCalled)
}
