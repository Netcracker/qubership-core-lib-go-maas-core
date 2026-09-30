[![Go build](https://github.com/Netcracker/qubership-core-lib-go-maas-core/actions/workflows/go-build.yml/badge.svg)](https://github.com/Netcracker/qubership-core-lib-go-maas-core/actions/workflows/go-build.yml)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?metric=coverage&project=Netcracker_qubership-core-lib-go-maas-core)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-maas-core)
[![duplicated_lines_density](https://sonarcloud.io/api/project_badges/measure?metric=duplicated_lines_density&project=Netcracker_qubership-core-lib-go-maas-core)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-maas-core)
[![vulnerabilities](https://sonarcloud.io/api/project_badges/measure?metric=vulnerabilities&project=Netcracker_qubership-core-lib-go-maas-core)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-maas-core)
[![bugs](https://sonarcloud.io/api/project_badges/measure?metric=bugs&project=Netcracker_qubership-core-lib-go-maas-core)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-maas-core)
[![code_smells](https://sonarcloud.io/api/project_badges/measure?metric=code_smells&project=Netcracker_qubership-core-lib-go-maas-core)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-maas-core)

# core

This lib provides methods to build maas clients with defaults required parameters: logger, namespace, maasAgentUrl and authSupplier.

<!-- TOC -->
* [core](#core)
  * [MaaS address and M2M_AUTH_MODE](#maas-address-and-m2m_auth_mode)
  * [Kafka](#kafka)
    * [Default usage:](#default-usage)
    * [You can override any of default parameters like shown in the code snippet below:](#you-can-override-any-of-default-parameters-like-shown-in-the-code-snippet-below)
    * [Watching tenant topics:](#watching-tenant-topics)
  * [Rabbit](#rabbit)
    * [Default usage:](#default-usage-1)
<!-- TOC -->


Unless `M2M_AUTH_MODE` is `k8s`, any client needs a registered security implemention - dummy or your own, the followning example shows registration of required services:

```go
import (
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
)

func init() {
  serviceloader.Register(2, &security.DummyToken{})
}
```

## MaaS address and M2M_AUTH_MODE

`NewKafkaClient` and `NewRabbitClient` pick the MaaS address by `M2M_AUTH_MODE`, described in the
[lib-go rest client README](https://github.com/Netcracker/qubership-core-lib-go/blob/main/security/rest/README.md).
The Kafka client picks the token for the tenant watch the same way.

| Mode               | MaaS address                                                   | Tenant watch token                            |
|--------------------|----------------------------------------------------------------|-----------------------------------------------|
| `legacy` (default) | maas-agent, `maas.agent.url`                                   | Legacy M2M token                              |
| `hybrid`           | `maas.internal.address`, or maas-agent when it is not set      | Legacy M2M token                              |
| `k8s`              | `maas.internal.address`, required                              | Kubernetes token with the netcracker audience |

In `k8s` mode without `maas.internal.address`, both constructors panic with
`maas.internal.address is not set: with M2M_AUTH_MODE=k8s the client sends requests directly to MaaS, set maas.internal.address to the MaaS URL`.
`WithMaaSUrl` replaces `maas.internal.address`, and `WithAuthSupplier` replaces the tenant watch token.

## Kafka

### Default usage:
~~~ go 
import (
	"context"
	"fmt"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go-maas-client/v3/classifier"
	maas "github.com/netcracker/qubership-core-lib-go-maas-core/v3"
)

func init() {
  serviceloader.Register(2, &security.DummyToken{})
}

func kafkaClientWithDefaults(ctx context.Context) error {
	maasKafkaClient := maas.NewKafkaClient()
	topicAddr, err := maasKafkaClient.GetTopic(ctx, classifier.New("demo").WithNamespace("namespace"))
	if err != nil {
		return err
	}
	fmt.Printf("topic = %s", topicAddr.TopicName)
	return nil
}
~~~

### Override default parameters
~~~ go 
import (
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go-maas-client/v3/logging"
	maas "github.com/netcracker/qubership-core-lib-go-maas-core/v3"
)

var myNamespace string
var myMaaSAgentUrl string
var myAuthSupplier func(ctx context.Context) (string, error)

func init() {
  serviceloader.Register(2, &security.DummyToken{})
}

func kafkaClientWithCustomParams() {
    maasKafkaClient := maas.NewKafkaClient(
        maas.WithNamespace(myNamespace), 
        maas.WithMaaSAgentUrl(myMaaSAgentUrl), 
        maas.WithAuthSupplier(myAuthSupplier))
}
~~~

### Supplying your own HTTP client

`WithHttpClient` replaces the default resty client, which is built with two
settings worth keeping:

* **No resty retries.** Retrying is handled by
  `qubership-core-lib-go-maas-client`; enabling both multiplies the number of
  requests per call.
* **No client-wide timeout.** The client also serves the topic watch, which
  long-polls for 60s. Bound individual calls with the context you pass instead.

### Watching tenant topics:
See example [tenant-topics-watch.go](examples/tenant-topics-watch.go)

## Rabbit

### Default usage:
~~~ go 
import (
	"context"
	"fmt"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go-maas-client/v3/classifier"
	maas "github.com/netcracker/qubership-core-lib-go-maas-core/v3"
)

func init() {
  serviceloader.Register(2, &security.DummyToken{})
}

func kafkaClientWithDefaults(ctx context.Context) error {
	maasRabbitClient := maas.NewRabbitClient()
	vhost, err := maasRabbitClient.GetVhost(ctx, classifier.New("demo").WithNamespace("namespace"))
	if err != nil {
		return err
	}
	fmt.Printf("vhost user = %s", vhost.Username)
	return nil
}
~~~

