package v1

import (
	"context"
	"sync"

	"github.com/rancher/norman/controller"
	"github.com/rancher/norman/objectclient"
	"github.com/rancher/norman/objectclient/dynamic"
	"github.com/rancher/norman/restwatch"
	"k8s.io/client-go/rest"
)

type (
	contextKeyType        struct{}
	contextClientsKeyType struct{}
)

type Interface interface {
	RESTClient() rest.Interface
	controller.Starter

	ModulesGetter
	ExecutionsGetter
	ExecutionRunsGetter
}

type Clients struct {
	Module       ModuleClient
	Execution    ExecutionClient
	ExecutionRun ExecutionRunClient
}

type Client struct {
	sync.Mutex
	restClient rest.Interface
	starters   []controller.Starter

	moduleControllers       map[string]ModuleController
	executionControllers    map[string]ExecutionController
	executionRunControllers map[string]ExecutionRunController
}

func Factory(ctx context.Context, config rest.Config) (context.Context, controller.Starter, error) {
	c, err := NewForConfig(config)
	if err != nil {
		return ctx, nil, err
	}

	cs := NewClientsFromInterface(c)

	ctx = context.WithValue(ctx, contextKeyType{}, c)
	ctx = context.WithValue(ctx, contextClientsKeyType{}, cs)
	return ctx, c, nil
}

func ClientsFrom(ctx context.Context) *Clients {
	return ctx.Value(contextClientsKeyType{}).(*Clients)
}

func From(ctx context.Context) Interface {
	return ctx.Value(contextKeyType{}).(Interface)
}

func NewClients(config rest.Config) (*Clients, error) {
	iface, err := NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return NewClientsFromInterface(iface), nil
}

func NewClientsFromInterface(iface Interface) *Clients {
	return &Clients{

		Module: &moduleClient2{
			iface: iface.Modules(""),
		},
		Execution: &executionClient2{
			iface: iface.Executions(""),
		},
		ExecutionRun: &executionRunClient2{
			iface: iface.ExecutionRuns(""),
		},
	}
}

func NewForConfig(config rest.Config) (Interface, error) {
	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = dynamic.NegotiatedSerializer
	}

	restClient, err := restwatch.UnversionedRESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &Client{
		restClient: restClient,

		moduleControllers:       map[string]ModuleController{},
		executionControllers:    map[string]ExecutionController{},
		executionRunControllers: map[string]ExecutionRunController{},
	}, nil
}

func (c *Client) RESTClient() rest.Interface {
	return c.restClient
}

func (c *Client) Sync(ctx context.Context) error {
	return controller.Sync(ctx, c.starters...)
}

func (c *Client) Start(ctx context.Context, threadiness int) error {
	return controller.Start(ctx, threadiness, c.starters...)
}

type ModulesGetter interface {
	Modules(namespace string) ModuleInterface
}

func (c *Client) Modules(namespace string) ModuleInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &ModuleResource, ModuleGroupVersionKind, moduleFactory{})
	return &moduleClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type ExecutionsGetter interface {
	Executions(namespace string) ExecutionInterface
}

func (c *Client) Executions(namespace string) ExecutionInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &ExecutionResource, ExecutionGroupVersionKind, executionFactory{})
	return &executionClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type ExecutionRunsGetter interface {
	ExecutionRuns(namespace string) ExecutionRunInterface
}

func (c *Client) ExecutionRuns(namespace string) ExecutionRunInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &ExecutionRunResource, ExecutionRunGroupVersionKind, executionRunFactory{})
	return &executionRunClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}
