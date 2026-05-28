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
  * [Kafka](#kafka)
    * [Default usage:](#default-usage)
    * [You can override any of default parameters like shown in the code snippet below:](#you-can-override-any-of-default-parameters-like-shown-in-the-code-snippet-below)
    * [Watching tenant topics:](#watching-tenant-topics)
  * [Rabbit](#rabbit)
    * [Default usage:](#default-usage-1)
<!-- TOC -->


To use any client it's necessary to register security implemention - dummy or your own, the followning example shows registration of required services:

```go
import (
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
)

func init() {
  serviceloader.Register(2, &security.DummyToken{})
}
```

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

