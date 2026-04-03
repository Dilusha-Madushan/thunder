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

package template

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/asgardeo/thunder/internal/system/config"
	declarativeresource "github.com/asgardeo/thunder/internal/system/declarative_resource"
	"github.com/asgardeo/thunder/internal/system/declarative_resource/entity"
)

type TemplateDeclarativeResourceTestSuite struct {
	suite.Suite
}

func TestTemplateDeclarativeResourceTestSuite(t *testing.T) {
	suite.Run(t, new(TemplateDeclarativeResourceTestSuite))
}

func (suite *TemplateDeclarativeResourceTestSuite) TestParseToTemplateDTO() {
	validYAML := []byte(`
id: "test-id"
displayName: "Test Name"
scenario: "USER_INVITE"
type: "email"
subject: "Test Subject"
contentType: "text/html"
body: "Hello {{ctx(inviteLink)}}"
`)
	result, err := parseToTemplateDTO(validYAML)
	suite.NoError(err)
	dto, ok := result.(*TemplateDTO)
	suite.True(ok)
	suite.Equal("test-id", dto.ID)
	suite.Equal(ScenarioUserInvite, dto.Scenario)
	suite.Equal("Test Subject", dto.Subject)
	suite.Equal("Hello {{ctx(inviteLink)}}", dto.Body)
}

func (suite *TemplateDeclarativeResourceTestSuite) TestParseToTemplateDTO_InvalidYAML() {
	invalidYAML := []byte(`id: "test-id"
  invalid-indent`)
	_, err := parseToTemplateDTO(invalidYAML)
	suite.Error(err)
}

func (suite *TemplateDeclarativeResourceTestSuite) TestValidateTemplateDTO() {
	validDTO := &TemplateDTO{
		ID:       "test-id",
		Scenario: ScenarioUserInvite,
		Subject:  "Test Subject",
		Body:     "Test Body",
	}
	err := validateTemplateDTO(validDTO)
	suite.NoError(err)
}

func (suite *TemplateDeclarativeResourceTestSuite) TestValidateTemplateDTO_MissingFields() {
	tests := []struct {
		name string
		dto  *TemplateDTO
	}{
		{
			name: "Missing ID",
			dto: &TemplateDTO{
				Scenario: ScenarioUserInvite,
				Subject:  "Test",
				Body:     "Test",
			},
		},
		{
			name: "Missing Scenario",
			dto: &TemplateDTO{
				ID:      "test",
				Subject: "Test",
				Body:    "Test",
			},
		},
		{
			name: "Missing Subject",
			dto: &TemplateDTO{
				ID:       "test",
				Scenario: ScenarioUserInvite,
				Body:     "Test",
			},
		},
		{
			name: "Missing Body",
			dto: &TemplateDTO{
				ID:       "test",
				Scenario: ScenarioUserInvite,
				Subject:  "Test",
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := validateTemplateDTO(tt.dto)
			suite.Error(err)
		})
	}
}

func (suite *TemplateDeclarativeResourceTestSuite) TestValidateTemplateDTO_InvalidType() {
	err := validateTemplateDTO("invalid type")
	if suite.Error(err) {
		suite.Contains(err.Error(), "invalid type")
	}
}

func (suite *TemplateDeclarativeResourceTestSuite) TestValidateTemplateDTO_UnsupportedScenario() {
	dto := &TemplateDTO{
		ID:       "test-id",
		Scenario: ScenarioType("UNSUPPORTED_SCENARIO"),
		Subject:  "Test Subject",
		Body:     "Test Body",
	}
	err := validateTemplateDTO(dto)
	if suite.Error(err) {
		suite.Contains(err.Error(), "unsupported template scenario")
	}
}

func (suite *TemplateDeclarativeResourceTestSuite) TestValidateTemplateDTO_SMSWithoutSubject() {
	dto := &TemplateDTO{
		ID:       "sms-otp",
		Scenario: ScenarioSMSOTP,
		Type:     TemplateTypeSMS,
		Body:     "Your code is {{ctx(otp)}}",
	}
	err := validateTemplateDTO(dto)
	suite.NoError(err)
}

func (suite *TemplateDeclarativeResourceTestSuite) TestLoadDeclarativeResources_WithSMSTemplateFile() {
	tempDir := suite.T().TempDir()
	testConfig := &config.Config{}
	config.ResetThunderRuntime()
	err := config.InitializeThunderRuntime(tempDir, testConfig)
	suite.NoError(err)

	runtime := config.GetThunderRuntime()
	resourceDir := filepath.Join(runtime.ThunderHome, "repository", "resources", "templates")
	err = os.MkdirAll(resourceDir, 0o750)
	suite.NoError(err)

	yamlData := []byte(`id: sms-otp
displayName: SMS OTP
scenario: SMS_OTP
type: sms
body: "Your code is {{ctx(otp)}}. Expires in {{ctx(expiryTime)}} minutes."
`)
	err = os.WriteFile(filepath.Join(resourceDir, "sms_otp.yaml"), yamlData, 0o600)
	suite.NoError(err)

	genericStore := declarativeresource.NewGenericFileBasedStoreForTest(entity.KeyTypeTemplate)
	store := &templateFileBasedStore{
		GenericFileBasedStore: genericStore,
	}
	err = loadDeclarativeResources(store)
	suite.NoError(err)

	tmpl, err := store.GetTemplate(context.Background(), "sms-otp")
	suite.NoError(err)
	suite.Equal("sms-otp", tmpl.ID)
	suite.Equal(ScenarioSMSOTP, tmpl.Scenario)
	suite.Empty(tmpl.Subject)
}

func (suite *TemplateDeclarativeResourceTestSuite) TestLoadDeclarativeResources_Integration() {
	tempDir := suite.T().TempDir()
	testConfig := &config.Config{}
	config.ResetThunderRuntime()
	err := config.InitializeThunderRuntime(tempDir, testConfig)
	suite.NoError(err)

	genericStore := declarativeresource.NewGenericFileBasedStoreForTest(entity.KeyTypeTemplate)
	store := &templateFileBasedStore{
		GenericFileBasedStore: genericStore,
	}
	err = loadDeclarativeResources(store)
	// Error is expected if directory doesn't exist, which is acceptable for this test
	_ = err
}

func (suite *TemplateDeclarativeResourceTestSuite) TestLoadDeclarativeResources_WithTemplateFiles() {
	tempDir := suite.T().TempDir()
	testConfig := &config.Config{}
	config.ResetThunderRuntime()
	err := config.InitializeThunderRuntime(tempDir, testConfig)
	suite.NoError(err)

	runtime := config.GetThunderRuntime()
	resourceDir := filepath.Join(runtime.ThunderHome, "repository", "resources", "templates")
	err = os.MkdirAll(resourceDir, 0o750)
	suite.NoError(err)

	yamlData := []byte(`id: test-template
displayName: Test Template
scenario: USER_INVITE
type: email
subject: Test Subject
contentType: text/html
body: "Hello {{ctx(inviteLink)}}"
`)
	err = os.WriteFile(filepath.Join(resourceDir, "test-template.yaml"), yamlData, 0o600)
	suite.NoError(err)

	genericStore := declarativeresource.NewGenericFileBasedStoreForTest(entity.KeyTypeTemplate)
	store := &templateFileBasedStore{
		GenericFileBasedStore: genericStore,
	}
	err = loadDeclarativeResources(store)
	suite.NoError(err)

	tmpl, err := store.GetTemplate(context.Background(), "test-template")
	suite.NoError(err)
	suite.Equal("test-template", tmpl.ID)
	suite.Equal(ScenarioUserInvite, tmpl.Scenario)
}

func (suite *TemplateDeclarativeResourceTestSuite) TestLoadDeclarativeResources_WithEmptyDirectoryPath() {
	tempDir := suite.T().TempDir()
	testConfig := &config.Config{}
	config.ResetThunderRuntime()
	err := config.InitializeThunderRuntime(tempDir, testConfig)
	suite.NoError(err)

	genericStore := declarativeresource.NewGenericFileBasedStoreForTest(entity.KeyTypeTemplate)
	store := &templateFileBasedStore{
		GenericFileBasedStore: genericStore,
	}
	err = loadDeclarativeResources(store)
	// Should not panic and may return error if default directory doesn't exist
	_ = err
}
