// Copyright 2025-2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"slices"

	"github.com/thunder-id/thunderid/pkg/thunderidengine/providers"

	"github.com/thunder-id/thunderid/internal/entitytype"
	"github.com/thunder-id/thunderid/internal/flow/core"
	"github.com/thunder-id/thunderid/internal/system/log"
)

// userTypeResolver is a registration-flow executor that resolves the user type at flow start.
type userTypeResolver struct {
	providers.Executor
	*entityTypeResolution
	logger *log.Logger
}

var _ providers.Executor = (*userTypeResolver)(nil)

// newUserTypeResolver creates a new instance of the UserTypeResolver executor.
func newUserTypeResolver(
	flowFactory core.FlowFactoryInterface,
	entityTypeService entitytype.EntityTypeServiceInterface,
	ouService providers.OrganizationUnitProvider,
) *userTypeResolver {
	logger := log.GetLogger().With(
		log.String(log.LoggerKeyComponentName, "UserTypeResolver"),
		log.String(log.LoggerKeyExecutorName, ExecutorNameUserTypeResolver))

	defaultInputs := []providers.Input{
		{
			Ref:        "usertype_input",
			Identifier: userTypeKey,
			Type:       providers.InputTypeSelect,
			Required:   true,
		},
	}

	base := flowFactory.CreateExecutor(ExecutorNameUserTypeResolver, providers.ExecutorTypeRegistration,
		defaultInputs, []providers.Input{}, &providers.ExecutorMeta{
			SupportedProperties: []providers.ExecutorSupportedProperties{
				{Property: propertyKeyAllowedUserTypes},
			},
		})

	return &userTypeResolver{
		Executor: base,
		entityTypeResolution: &entityTypeResolution{
			entityTypeService: entityTypeService,
			ouService:         ouService,
		},
		logger: logger,
	}
}

// Execute resolves the user type from inputs or prompts the user to select one.
func (u *userTypeResolver) Execute(ctx *providers.NodeContext) (*providers.ExecutorResponse, error) {
	logger := u.logger.With(log.String(log.LoggerKeyExecutionID, ctx.ExecutionID))
	logger.Debug(ctx.Context, "Executing user type resolver")

	execResp := &providers.ExecutorResponse{
		AdditionalData: make(map[string]string),
		RuntimeData:    make(map[string]string),
		ForwardedData:  make(map[string]interface{}),
	}

	switch ctx.FlowType {
	case providers.FlowTypeAuthentication:
		return checkAuthenticationAllowed(ctx.Application.AllowedUserTypes, execResp), nil
	case providers.FlowTypeRegistration:
		return u.handleRegistrationFlows(ctx, execResp, logger)
	case providers.FlowTypeUserOnboarding:
		return u.resolve(ctx, entitytype.TypeCategoryUser,
			allowedTypesFromProperties(ctx, propertyKeyAllowedUserTypes), u.GetDefaultInputs(), execResp, logger)
	default:
		logger.Debug(ctx.Context, "User type resolver is not applicable for the flow type",
			log.String("flowType", string(ctx.FlowType)))
		execResp.Status = providers.ExecComplete
		return execResp, nil
	}
}

// handleRegistrationFlows resolves the user type for self-registration. The candidates are the
// application's allowed user types, narrowed by the node's allowedUserTypes property, and only a type
// open to self-registration is accepted.
func (u *userTypeResolver) handleRegistrationFlows(ctx *providers.NodeContext,
	execResp *providers.ExecutorResponse, logger *log.Logger) (*providers.ExecutorResponse, error) {
	allowed := ctx.Application.AllowedUserTypes
	if len(allowed) == 0 {
		// TODO: This should be improved to fallback to the application's ou when the support is available.
		//  userType has an attached ou. Need to find userType from the application's ou.
		//  Also should check if self registration is enabled for the user type when the support is available.
		execResp.Status = providers.ExecFailure
		execResp.Error = &ErrSelfRegNotAvailableForApp
		return execResp, nil
	}

	if nodeAllowed := allowedTypesFromProperties(ctx, propertyKeyAllowedUserTypes); len(nodeAllowed) > 0 {
		allowed = slices.DeleteFunc(slices.Clone(allowed), func(userType string) bool {
			return !slices.Contains(nodeAllowed, userType)
		})
		if len(allowed) == 0 {
			execResp.Status = providers.ExecFailure
			execResp.Error = errForEntityCategory(ErrNoValidUserTypes, entitytype.TypeCategoryUser)
			return execResp, nil
		}
	}

	if u.HasRequiredInputs(ctx, execResp) {
		selected := ctx.UserInputs[userTypeKey]
		if !slices.Contains(allowed, selected) {
			execResp.Status = providers.ExecFailure
			execResp.Error = errForEntityCategory(ErrUserTypeNotAllowed, entitytype.TypeCategoryUser)
			return execResp, nil
		}
		allowed = []string{selected}
	}

	if len(allowed) == 1 {
		entityType, ouID, err := u.entityTypeAndOU(ctx.Context, entitytype.TypeCategoryUser, allowed[0])
		if err != nil {
			return execResp, err
		}
		if !entityType.AllowSelfRegistration {
			execResp.Status = providers.ExecFailure
			execResp.Error = &ErrSelfRegDisabledForUserType
			return execResp, nil
		}
		return u.accept(ctx, allowed[0], ouID, execResp, logger), nil
	}

	return u.resolveFromSelfRegEnabled(ctx, allowed, execResp, logger)
}

// resolveFromSelfRegEnabled narrows several allowed user types to those open to self-registration,
// taking the only one left or offering them as a choice.
func (u *userTypeResolver) resolveFromSelfRegEnabled(ctx *providers.NodeContext, allowed []string,
	execResp *providers.ExecutorResponse, logger *log.Logger) (*providers.ExecutorResponse, error) {
	selfRegEnabled := make([]*entitytype.EntityType, 0, len(allowed))
	for _, userType := range allowed {
		entityType, _, err := u.entityTypeAndOU(ctx.Context, entitytype.TypeCategoryUser, userType)
		if err != nil {
			return execResp, err
		}
		if entityType.AllowSelfRegistration {
			selfRegEnabled = append(selfRegEnabled, entityType)
		}
	}

	switch len(selfRegEnabled) {
	case 0:
		execResp.Status = providers.ExecFailure
		execResp.Error = &ErrSelfRegNotAvailableForApp
		return execResp, nil
	case 1:
		return u.accept(ctx, selfRegEnabled[0].Handle, selfRegEnabled[0].OUID, execResp, logger), nil
	}

	options := make([]string, 0, len(selfRegEnabled))
	for _, entityType := range selfRegEnabled {
		options = append(options, entityType.Handle)
	}
	u.prompt(ctx, entitytype.TypeCategoryUser, options, u.GetDefaultInputs(), execResp, logger)

	return execResp, nil
}
