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

package notification

import (
	"context"

	"github.com/asgardeo/thunder/internal/notification/common"
	"github.com/asgardeo/thunder/internal/system/error/serviceerror"
	"github.com/asgardeo/thunder/internal/system/log"
)

// SMSSenderServiceInterface defines the interface for sending SMS messages.
type SMSSenderServiceInterface interface {
	SendSMS(ctx context.Context, senderID string, recipient string, body string) *serviceerror.ServiceError
}

// smsSenderService implements SMSSenderServiceInterface.
type smsSenderService struct {
	senderMgtService NotificationSenderMgtSvcInterface
	clientProvider   notificationClientProviderInterface
}

// newSMSSenderService returns a new instance of SMSSenderServiceInterface.
func newSMSSenderService(senderMgtService NotificationSenderMgtSvcInterface) SMSSenderServiceInterface {
	return &smsSenderService{
		senderMgtService: senderMgtService,
		clientProvider:   newNotificationClientProvider(),
	}
}

// SendSMS looks up the sender by ID, obtains the appropriate message client, and sends the SMS.
func (s *smsSenderService) SendSMS(ctx context.Context, senderID string, recipient string,
	body string) *serviceerror.ServiceError {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "SMSSenderService"))

	if len([]rune(body)) > common.SMSMaxLength {
		logger.Warn("SMS body exceeds single message limit",
			log.Int("length", len([]rune(body))), log.Int("limit", common.SMSMaxLength))
	}

	sender, svcErr := s.senderMgtService.GetSender(ctx, senderID)
	if svcErr != nil {
		return svcErr
	}

	if sender.Type != common.NotificationSenderTypeMessage {
		return &ErrorRequestedSenderIsNotOfExpectedType
	}

	client, svcErr := s.clientProvider.GetMessageClient(*sender)
	if svcErr != nil {
		return svcErr
	}

	if err := client.SendSMS(common.SMSData{To: recipient, Body: body}); err != nil {
		return &ErrorInternalServerError
	}

	return nil
}
