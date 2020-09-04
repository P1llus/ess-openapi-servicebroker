package broker

import (
	"context"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/P1llus/ess-openapi-servicebroker/config"
	"github.com/P1llus/ess-openapi-servicebroker/provider"
	"github.com/pivotal-cf/brokerapi/v7"
	"github.com/pivotal-cf/brokerapi/v7/domain"
)

type Broker struct {
	brokerConfig   config.Broker
	Provider       provider.ServiceProvider
	logger         lager.Logger
	brokerServices []domain.Service
}

func NewBroker(brokerConfig config.Broker, serviceProvider provider.ServiceProvider, services []domain.Service, logger lager.Logger) *Broker {
	broker := &Broker{
		brokerConfig:   brokerConfig,
		Provider:       serviceProvider,
		logger:         logger,
		brokerServices: services,
	}
	return broker
}

func (b *Broker) NewBrokerHTTPServer(broker domain.ServiceBroker) http.Handler {
	credentials := brokerapi.BrokerCredentials{
		Username: b.brokerConfig.Username,
		Password: b.brokerConfig.Password,
	}

	brokerAPI := brokerapi.New(broker, b.logger, credentials)
	mux := http.NewServeMux()
	mux.Handle("/", brokerAPI)
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func (b *Broker) GetBinding(ctx context.Context, first, second string) (domain.GetBindingSpec, error) {
	return domain.GetBindingSpec{}, nil
}

func (b *Broker) GetInstance(ctx context.Context, first string) (domain.GetInstanceDetailsSpec, error) {
	return domain.GetInstanceDetailsSpec{}, nil
}

func (b *Broker) LastBindingOperation(ctx context.Context, first, second string, pollDetails domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, nil
}

func (b *Broker) Services(ctx context.Context) ([]domain.Service, error) {
	return b.brokerServices, nil
}

func (b *Broker) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, isAsyncAllowed bool) (domain.ProvisionedServiceSpec, error) {

	if !isAsyncAllowed {
		return domain.ProvisionedServiceSpec{}, brokerapi.ErrAsyncRequired
	}
	plan, err := config.FindProvisionDetails(b.brokerServices, details.ServiceID, details.PlanID)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	provisionData := &provider.ProvisionData{
		InstanceID: instanceID,
		Details:    details,
		Plan:       plan,
	}
	dashboardURL, operationData, err := b.Provider.Provision(providerCtx, provisionData)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	return domain.ProvisionedServiceSpec{DashboardURL: dashboardURL, OperationData: operationData, IsAsync: true, AlreadyExists: false}, nil
}

func (b *Broker) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, isAsyncAllowed bool) (domain.DeprovisionServiceSpec, error) {
	if !isAsyncAllowed {
		return domain.DeprovisionServiceSpec{}, brokerapi.ErrAsyncRequired
	}
	deprovisionData := &provider.DeprovisionData{
		InstanceID: instanceID,
		Details:    details,
	}

	operationData, err := b.Provider.Deprovision(ctx, deprovisionData)
	if err != nil {
		return domain.DeprovisionServiceSpec{}, err
	}
	return domain.DeprovisionServiceSpec{IsAsync: true, OperationData: operationData}, nil
}

func (b *Broker) Bind(ctx context.Context, instanceID string, bindID string, bindDetails domain.BindDetails, isAsyncAllowed bool) (domain.Binding, error) {
	bindData := &provider.BindData{
		InstanceID: instanceID,
		BindingID:  bindID,
		Details:    bindDetails,
	}
	credentials, operationData, err := b.Provider.Bind(ctx, bindData)
	if err != nil {
		return domain.Binding{}, err
	}
	return domain.Binding{Credentials: credentials, OperationData: operationData}, nil
}

func (b *Broker) Unbind(ctx context.Context, instanceID string, bindID string, unbindDetails domain.UnbindDetails, isAsyncAllowed bool) (domain.UnbindSpec, error) {
	unBindData := &provider.UnbindData{
		InstanceID: instanceID,
		BindingID:  bindID,
		Details:    unbindDetails,
	}
	operationData, err := b.Provider.Unbind(ctx, unBindData)
	if err != nil {
		return domain.UnbindSpec{}, err
	}
	return domain.UnbindSpec{OperationData: operationData}, nil
}

func (b *Broker) Update(context.Context, string, domain.UpdateDetails, bool) (domain.UpdateServiceSpec, error) {
	return domain.UpdateServiceSpec{}, nil
}

func (b *Broker) LastOperation(ctx context.Context, instanceId string, pollDetails domain.PollDetails) (domain.LastOperation, error) {
	lastOperationData := &provider.LastOperationData{
		InstanceID:    instanceId,
		OperationData: pollDetails.OperationData,
	}
	state, description, err := b.Provider.LastOperation(ctx, lastOperationData)
	if err != nil {
		return brokerapi.LastOperation{}, err
	}
	return domain.LastOperation{State: state, Description: description}, nil
}
