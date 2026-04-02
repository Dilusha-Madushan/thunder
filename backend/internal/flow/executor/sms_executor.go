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
	"errors"
	"fmt"

	"github.com/asgardeo/thunder/internal/flow/common"
	"github.com/asgardeo/thunder/internal/flow/core"
	"github.com/asgardeo/thunder/internal/notification"
	"github.com/asgardeo/thunder/internal/system/error/serviceerror"
	"github.com/asgardeo/thunder/internal/system/log"
)

// smsExecutor sends an SMS message using the configured sender and message body from node properties.
// When smsSenderService is nil, it completes as a no-op with smsSent=false.
type smsExecutor struct {
	core.ExecutorInterface
	logger           *log.Logger
	smsSenderService notification.SMSSenderServiceInterface
}

// newSMSExecutor creates a new instance of smsExecutor.
// smsSenderService may be nil if no SMS provider is configured; the executor completes as a no-op in that case.
func newSMSExecutor(flowFactory core.FlowFactoryInterface,
	smsSenderService notification.SMSSenderServiceInterface) *smsExecutor {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "SMSExecutor"))
	base := flowFactory.CreateExecutor(
		ExecutorNameSMSExecutor,
		common.ExecutorTypeUtility,
		[]common.Input{},
		[]common.Input{
			{Identifier: userAttributeMobileNumber, Type: common.InputTypePhone, Required: true},
		},
	)
	return &smsExecutor{
		ExecutorInterface: base,
		logger:            logger,
		smsSenderService:  smsSenderService,
	}
}

// Execute sends an SMS using data from the runtime context.
func (e *smsExecutor) Execute(ctx *core.NodeContext) (*common.ExecutorResponse, error) {
	switch ctx.ExecutorMode {
	case ExecutorModeSend:
		return e.executeSend(ctx)
	default:
		return nil, fmt.Errorf("invalid executor mode for SMSExecutor: %s", ctx.ExecutorMode)
	}
}

// executeSend resolves the recipient, sender ID, and message body from node properties, then sends the SMS.
// If the SMS sender service is not configured, it completes without sending (no-op).
func (e *smsExecutor) executeSend(ctx *core.NodeContext) (*common.ExecutorResponse, error) {
	logger := e.logger.With(log.String(log.LoggerKeyFlowID, ctx.FlowID))
	logger.Debug("Executing SMS executor in send mode")

	execResp := &common.ExecutorResponse{
		AdditionalData: make(map[string]string),
		RuntimeData:    make(map[string]string),
	}

	if e.smsSenderService == nil {
		execResp.AdditionalData[common.DataSMSSent] = dataValueFalse
		logger.Debug("SMS sender service not configured, skipping SMS send")
		execResp.Status = common.ExecComplete
		return execResp, nil
	}

	recipient := resolveRecipientMobile(ctx)
	if recipient == "" {
		logger.Debug("SMS recipient not found in user inputs or runtime data")
		execResp.Status = common.ExecFailure
		execResp.FailureReason = "SMS recipient is required"
		return execResp, nil
	}

	senderID, err := resolveStringNodeProperty(ctx, propertyKeySMSSenderID)
	if err != nil {
		return nil, fmt.Errorf("senderId is not configured in node properties: %w", err)
	}

	// TODO: Replace smsDefaultMessage with a proper template-based message body in a future PR.
	svcErr := e.smsSenderService.SendSMS(ctx.Context, senderID, recipient, smsDefaultMessage)
	if svcErr != nil {
		if svcErr.Type == serviceerror.ClientErrorType {
			execResp.Status = common.ExecFailure
			execResp.FailureReason = svcErr.ErrorDescription
			return execResp, nil
		}
		return nil, fmt.Errorf("SMS send failed: %s", svcErr.ErrorDescription)
	}

	logger.Debug("SMS sent successfully", log.String("recipient", log.MaskString(recipient)))

	execResp.AdditionalData[common.DataSMSSent] = dataValueTrue
	execResp.Status = common.ExecComplete
	return execResp, nil
}

// resolveRecipientMobile retrieves the recipient mobile number from user inputs or runtime data.
func resolveRecipientMobile(ctx *core.NodeContext) string {
	if mobile, ok := ctx.UserInputs[userAttributeMobileNumber]; ok && mobile != "" {
		return mobile
	}
	if mobile, ok := ctx.RuntimeData[userAttributeMobileNumber]; ok && mobile != "" {
		return mobile
	}
	return ""
}

// resolveStringNodeProperty reads a string property from NodeProperties, returning an error if missing or wrong type.
func resolveStringNodeProperty(ctx *core.NodeContext, key string) (string, error) {
	val, ok := ctx.NodeProperties[key]
	if !ok {
		return "", errors.New("property not found")
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid type for %s: expected string, got %T", key, val)
	}
	if str == "" {
		return "", errors.New("property is empty")
	}
	return str, nil
}
