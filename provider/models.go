/*
Package provider implements all functionality from the ess package to communicate with the Elastic Cloud API.
The package is created separately from the broker package to differentiate between the functionality defined
for a broker compared to the provider
*/
package provider

import (
	"context"

	"github.com/pivotal-cf/brokerapi/v7/domain"
)

// ServiceProvider interface is used for the Provider to implement the ServiceProvider type
// This is required by the Broker and the BrokerAPI package
type ServiceProvider interface {
	Provision(context.Context, *ProvisionData) (dashboardURL, operationData string, err error)
	Deprovision(context.Context, *DeprovisionData) (operationData string, err error)
	Bind(context.Context, *BindData) (credentials Credentials, operationData string, err error)
	Unbind(context.Context, *UnbindData) (operationData string, err error)
	Update(context.Context, *UpdateData) (operationData string, err error)
	LastOperation(context.Context, *LastOperationData) (state domain.LastOperationState, description string, err error)
}

// ProvisionData struct is the expected type used during provision operations
type ProvisionData struct {
	InstanceID string
	Details    domain.ProvisionDetails
	Service    domain.Service
	Plan       domain.ServicePlan
}

// DeprovisionData struct is the expected type used during deprovision operations
type DeprovisionData struct {
	InstanceID string
	Details    domain.DeprovisionDetails
	Service    domain.Service
	Plan       domain.ServicePlan
}

// BindData struct is the expected type used during bind operations
type BindData struct {
	InstanceID string
	BindingID  string
	Details    domain.BindDetails
}

// UnbindData struct is the expected type used during unbind operations
type UnbindData struct {
	InstanceID string
	BindingID  string
	Details    domain.UnbindDetails
}

// UpdateData struct is the expected type used during update operations
type UpdateData struct {
	InstanceID string
	Details    domain.UpdateDetails
	Service    domain.Service
	Plan       domain.ServicePlan
}

// LastOperationData struct is the expected type used during lastoperation operations
type LastOperationData struct {
	InstanceID    string
	OperationData string
}
