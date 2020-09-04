package provider

import (
	"context"

	"github.com/pivotal-cf/brokerapi/v7/domain"
)

type ServiceProvider interface {
	Provision(context.Context, *ProvisionData) (dashboardURL, operationData string, err error)
	Deprovision(context.Context, *DeprovisionData) (operationData string, err error)
	Bind(context.Context, *BindData) (credentials Credentials, operationData string, err error)
	Unbind(context.Context, *UnbindData) (operationData string, err error)
	Update(context.Context, *UpdateData) (operationData string, err error)
	LastOperation(context.Context, *LastOperationData) (state domain.LastOperationState, description string, err error)
}

type ProvisionData struct {
	InstanceID string
	Details    domain.ProvisionDetails
	Service    domain.Service
	Plan       domain.ServicePlan
}

type DeprovisionData struct {
	InstanceID string
	Details    domain.DeprovisionDetails
	Service    domain.Service
	Plan       domain.ServicePlan
}

type BindData struct {
	InstanceID string
	BindingID  string
	Details    domain.BindDetails
}

type UnbindData struct {
	InstanceID string
	BindingID  string
	Details    domain.UnbindDetails
}

type UpdateData struct {
	InstanceID string
	Details    domain.UpdateDetails
	Service    domain.Service
	Plan       domain.ServicePlan
}

type LastOperationData struct {
	InstanceID    string
	OperationData string
}
