/*
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package executor

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/asgardeo/thunder/internal/flow/common"
	"github.com/asgardeo/thunder/internal/flow/core"
	"github.com/asgardeo/thunder/internal/system/error/serviceerror"
	"github.com/asgardeo/thunder/internal/system/template"
	"github.com/asgardeo/thunder/tests/mocks/flow/coremock"
	"github.com/asgardeo/thunder/tests/mocks/notification/notificationmock"
	"github.com/asgardeo/thunder/tests/mocks/templatemock"
)

type SMSExecutorTestSuite struct {
	suite.Suite
	mockFlowFactory     *coremock.FlowFactoryInterfaceMock
	mockSMSSenderSvc    *notificationmock.SMSSenderServiceInterfaceMock
	mockTemplateService *templatemock.TemplateServiceInterfaceMock
	executor            *smsExecutor
}

func (suite *SMSExecutorTestSuite) SetupTest() {
	suite.mockFlowFactory = coremock.NewFlowFactoryInterfaceMock(suite.T())
	mockBaseExecutor := coremock.NewExecutorInterfaceMock(suite.T())
	suite.mockSMSSenderSvc = notificationmock.NewSMSSenderServiceInterfaceMock(suite.T())
	suite.mockTemplateService = templatemock.NewTemplateServiceInterfaceMock(suite.T())

	suite.mockFlowFactory.On("CreateExecutor",
		ExecutorNameSMSExecutor,
		common.ExecutorTypeUtility,
		[]common.Input{},
		[]common.Input{
			{Identifier: userAttributeMobileNumber, Type: common.InputTypePhone, Required: true},
		},
	).Return(mockBaseExecutor)

	suite.executor = newSMSExecutor(suite.mockFlowFactory, suite.mockSMSSenderSvc, suite.mockTemplateService)
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_Success() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: map[string]string{
			common.RuntimeKeyInviteLink: "https://example.com/invite",
		},
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
			propertyKeySMSTemplate: "SMS_INVITE",
		},
	}

	suite.mockTemplateService.On("Render",
		mock.Anything,
		template.ScenarioSMSInvite,
		template.TemplateData{
			"inviteLink": "https://example.com/invite",
			"appName":    "",
		},
	).Return(&template.RenderedTemplate{
		Body: "You have a pending notification. Visit: https://example.com/invite",
	}, nil)

	suite.mockSMSSenderSvc.On("SendSMS",
		mock.Anything, "sender-uuid-001", "+94714627887",
		"You have a pending notification. Visit: https://example.com/invite",
	).Return(nil)

	resp, err := suite.executor.Execute(ctx)

	suite.NoError(err)
	suite.Equal(common.ExecComplete, resp.Status)
	suite.Equal(dataValueTrue, resp.AdditionalData[common.DataSMSSent])
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_DefaultScenario_WhenNoTemplateProperty() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: make(map[string]string),
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
		},
	}

	suite.mockTemplateService.On("Render",
		mock.Anything,
		template.ScenarioSMSInvite,
		mock.Anything,
	).Return(&template.RenderedTemplate{
		Body: "You have a pending notification.",
	}, nil)

	suite.mockSMSSenderSvc.On("SendSMS",
		mock.Anything, "sender-uuid-001", "+94714627887", "You have a pending notification.",
	).Return(nil)

	resp, err := suite.executor.Execute(ctx)

	suite.NoError(err)
	suite.Equal(common.ExecComplete, resp.Status)
	suite.Equal(dataValueTrue, resp.AdditionalData[common.DataSMSSent])
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_RecipientFromRuntimeData() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs:   make(map[string]string),
		RuntimeData: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
		},
	}

	suite.mockTemplateService.On("Render", mock.Anything, template.ScenarioSMSInvite, mock.Anything).
		Return(&template.RenderedTemplate{Body: "Notification."}, nil)

	suite.mockSMSSenderSvc.On("SendSMS",
		mock.Anything, "sender-uuid-001", "+94714627887", "Notification.",
	).Return(nil)

	resp, err := suite.executor.Execute(ctx)

	suite.NoError(err)
	suite.Equal(common.ExecComplete, resp.Status)
	suite.Equal(dataValueTrue, resp.AdditionalData[common.DataSMSSent])
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_UserInputOverridesRuntimeData() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: map[string]string{
			userAttributeMobileNumber: "+94771111111",
		},
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
		},
	}

	suite.mockTemplateService.On("Render", mock.Anything, template.ScenarioSMSInvite, mock.Anything).
		Return(&template.RenderedTemplate{Body: "Notification."}, nil)

	suite.mockSMSSenderSvc.On("SendSMS",
		mock.Anything, "sender-uuid-001", "+94714627887", "Notification.",
	).Return(nil)

	resp, err := suite.executor.Execute(ctx)

	suite.NoError(err)
	suite.Equal(common.ExecComplete, resp.Status)
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_MissingRecipient() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs:   make(map[string]string),
		RuntimeData:  make(map[string]string),
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
		},
	}

	resp, err := suite.executor.Execute(ctx)

	suite.NoError(err)
	suite.Equal(common.ExecFailure, resp.Status)
	suite.Equal("SMS recipient is required", resp.FailureReason)
	suite.mockSMSSenderSvc.AssertNotCalled(suite.T(), "SendSMS",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_MissingSenderID() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData:    make(map[string]string),
		NodeProperties: map[string]interface{}{},
	}

	resp, err := suite.executor.Execute(ctx)

	suite.Error(err)
	suite.Nil(resp)
	suite.Contains(err.Error(), "senderId is not configured")
	suite.mockSMSSenderSvc.AssertNotCalled(suite.T(), "SendSMS",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_InvalidSenderIDType() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: make(map[string]string),
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: 123,
		},
	}

	resp, err := suite.executor.Execute(ctx)

	suite.Error(err)
	suite.Nil(resp)
	suite.Contains(err.Error(), "senderId is not configured")
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_TemplateNotFound() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: make(map[string]string),
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
			propertyKeySMSTemplate: "SMS_INVITE",
		},
	}

	templateErr := &serviceerror.I18nServiceError{
		Code: "TMP-1001",
	}
	suite.mockTemplateService.On("Render", mock.Anything, template.ScenarioSMSInvite, mock.Anything).
		Return(nil, templateErr)

	resp, err := suite.executor.Execute(ctx)

	suite.Error(err)
	suite.Nil(resp)
	suite.Contains(err.Error(), "failed to render SMS template")
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_NilTemplateService() {
	mockBaseExecutor := coremock.NewExecutorInterfaceMock(suite.T())
	mockFactory := coremock.NewFlowFactoryInterfaceMock(suite.T())
	mockFactory.On("CreateExecutor",
		ExecutorNameSMSExecutor,
		common.ExecutorTypeUtility,
		[]common.Input{},
		[]common.Input{
			{Identifier: userAttributeMobileNumber, Type: common.InputTypePhone, Required: true},
		},
	).Return(mockBaseExecutor)

	noTemplateExecutor := newSMSExecutor(mockFactory, suite.mockSMSSenderSvc, nil)

	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: make(map[string]string),
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
		},
	}

	resp, err := noTemplateExecutor.Execute(ctx)

	suite.Error(err)
	suite.Nil(resp)
	suite.Contains(err.Error(), "template service is not configured")
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_NilSMSSenderService_NoOp() {
	mockBaseExecutor := coremock.NewExecutorInterfaceMock(suite.T())
	mockFactory := coremock.NewFlowFactoryInterfaceMock(suite.T())
	mockFactory.On("CreateExecutor",
		ExecutorNameSMSExecutor,
		common.ExecutorTypeUtility,
		[]common.Input{},
		[]common.Input{
			{Identifier: userAttributeMobileNumber, Type: common.InputTypePhone, Required: true},
		},
	).Return(mockBaseExecutor)

	noServiceExecutor := newSMSExecutor(mockFactory, nil, suite.mockTemplateService)

	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: make(map[string]string),
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
		},
	}

	resp, err := noServiceExecutor.Execute(ctx)

	suite.NoError(err)
	suite.Equal(common.ExecComplete, resp.Status)
	suite.Equal(dataValueFalse, resp.AdditionalData[common.DataSMSSent])
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_ClientError() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: make(map[string]string),
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
		},
	}

	suite.mockTemplateService.On("Render", mock.Anything, template.ScenarioSMSInvite, mock.Anything).
		Return(&template.RenderedTemplate{Body: "Notification."}, nil)

	clientErr := &serviceerror.ServiceError{
		Type:             serviceerror.ClientErrorType,
		Code:             "MNS-1001",
		Error:            "Sender not found",
		ErrorDescription: "The requested notification sender could not be found",
	}
	suite.mockSMSSenderSvc.On("SendSMS", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(clientErr)

	resp, err := suite.executor.Execute(ctx)

	suite.NoError(err)
	suite.Equal(common.ExecFailure, resp.Status)
	suite.Equal("The requested notification sender could not be found", resp.FailureReason)
}

func (suite *SMSExecutorTestSuite) TestExecute_SendMode_ServerError() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: ExecutorModeSend,
		UserInputs: map[string]string{
			userAttributeMobileNumber: "+94714627887",
		},
		RuntimeData: make(map[string]string),
		NodeProperties: map[string]interface{}{
			propertyKeySMSSenderID: "sender-uuid-001",
		},
	}

	suite.mockTemplateService.On("Render", mock.Anything, template.ScenarioSMSInvite, mock.Anything).
		Return(&template.RenderedTemplate{Body: "Notification."}, nil)

	serverErr := &serviceerror.ServiceError{
		Type:             serviceerror.ServerErrorType,
		Code:             "MNS-5000",
		ErrorDescription: "internal server error",
	}
	suite.mockSMSSenderSvc.On("SendSMS", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(serverErr)

	resp, err := suite.executor.Execute(ctx)

	suite.Error(err)
	suite.Nil(resp)
	suite.Contains(err.Error(), "SMS send failed")
}

func (suite *SMSExecutorTestSuite) TestExecute_InvalidMode() {
	ctx := &core.NodeContext{
		FlowID:       "test-flow-id",
		ExecutorMode: "invalid",
		UserInputs:   make(map[string]string),
		RuntimeData:  make(map[string]string),
	}

	resp, err := suite.executor.Execute(ctx)

	suite.Error(err)
	suite.Contains(err.Error(), "invalid executor mode for SMSExecutor")
	suite.Nil(resp)
}

func TestSMSExecutorSuite(t *testing.T) {
	suite.Run(t, new(SMSExecutorTestSuite))
}
