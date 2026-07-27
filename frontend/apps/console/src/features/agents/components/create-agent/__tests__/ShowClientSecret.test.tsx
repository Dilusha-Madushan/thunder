/**
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import userEvent from '@testing-library/user-event';
import {render, screen, waitForElementToBeRemoved, within} from '@thunderid/test-utils';
import {describe, it, expect, beforeEach, vi} from 'vitest';
import ShowClientSecret, {type ShowClientSecretProps} from '../ShowClientSecret';

vi.mock('@thunderid/hooks', () => ({
  useCopyToClipboard: vi.fn(),
}));

const {mockUseThunderID} = vi.hoisted(() => ({
  mockUseThunderID: vi.fn(),
}));

vi.mock('@thunderid/react', () => ({
  useThunderID: mockUseThunderID,
}));

const {useCopyToClipboard} = await import('@thunderid/hooks');

describe('ShowClientSecret', () => {
  const mockOnContinue = vi.fn();
  const mockCopy = vi.fn().mockResolvedValue(undefined);

  const defaultProps: ShowClientSecretProps = {
    agentName: 'Test Agent',
    agentId: 'agent-id-abc',
    clientId: 'client-id-xyz',
    clientSecret: 'test_secret_12345',
    onContinue: mockOnContinue,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useCopyToClipboard).mockReturnValue({
      copied: false,
      copy: mockCopy,
    });
    mockUseThunderID.mockReturnValue({
      discovery: {
        wellKnown: {
          issuer: 'https://localhost:8090',
          authorization_endpoint: 'https://localhost:8090/oauth2/authorize',
          token_endpoint: 'https://localhost:8090/oauth2/token',
        },
      },
    });
  });

  const renderComponent = (props: Partial<ShowClientSecretProps> = {}) =>
    render(<ShowClientSecret {...defaultProps} {...props} />);

  // The completion window renders inside the `agent-show-client-secret` container; the secret
  // dialog is portalled outside it, so scoping to the container isolates window-only assertions.
  const getWindow = () => within(screen.getByTestId('agent-show-client-secret'));

  describe('completion window', () => {
    it('should render the created title and subtitle', () => {
      renderComponent();

      expect(getWindow().getByText(/agent created/i)).toBeInTheDocument();
      expect(getWindow().getByText(/save its client secret before you continue/i)).toBeInTheDocument();
    });

    it('should display the agent name and agent ID', () => {
      renderComponent();

      expect(getWindow().getByText('Agent name')).toBeInTheDocument();
      expect(getWindow().getByText('Test Agent')).toBeInTheDocument();
      expect(getWindow().getByText('Agent ID')).toBeInTheDocument();
      expect(getWindow().getByText('agent-id-abc')).toBeInTheDocument();
    });

    it('should not display the agent ID field when not provided', () => {
      renderComponent({agentId: undefined});

      expect(getWindow().queryByText('Agent ID')).not.toBeInTheDocument();
    });

    it('should render the credentials info note and the client ID', () => {
      renderComponent();

      expect(
        getWindow().getByText(/your agent authenticates with these credentials to obtain access tokens/i),
      ).toBeInTheDocument();
      expect(getWindow().getByText('Client ID')).toBeInTheDocument();
      expect(getWindow().getByDisplayValue('client-id-xyz')).toBeInTheDocument();
    });

    it('should not render the client secret as a field on the window', () => {
      renderComponent();

      expect(getWindow().queryByDisplayValue('test_secret_12345')).not.toBeInTheDocument();
      expect(getWindow().getByText(/shown only once/i)).toBeInTheDocument();
    });

    it('should render the discovery endpoint fields', () => {
      renderComponent();

      const window = getWindow();
      expect(window.getByText('Issuer')).toBeInTheDocument();
      expect(window.getByDisplayValue('https://localhost:8090')).toBeInTheDocument();
      expect(window.getByText('OpenID Connect discovery')).toBeInTheDocument();
      expect(window.getByDisplayValue('https://localhost:8090/.well-known/openid-configuration')).toBeInTheDocument();
      expect(window.getByText('Authorization endpoint')).toBeInTheDocument();
      expect(window.getByText('Token endpoint')).toBeInTheDocument();
    });

    it('should not render the endpoints card when discovery is unavailable', () => {
      mockUseThunderID.mockReturnValue({discovery: {wellKnown: null}});
      renderComponent();

      expect(getWindow().queryByText('Token endpoint')).not.toBeInTheDocument();
      expect(getWindow().queryByText('Issuer')).not.toBeInTheDocument();
    });

    it('should call onContinue when the continue button is clicked', async () => {
      const user = userEvent.setup();
      renderComponent();

      await user.click(screen.getByTestId('agent-client-secret-continue'));

      expect(mockOnContinue).toHaveBeenCalledTimes(1);
    });
  });

  describe('secret dialog', () => {
    it('should open the secret dialog automatically on mount', () => {
      renderComponent();

      expect(screen.getByTestId('agent-client-secret-dialog')).toBeInTheDocument();
      expect(screen.getByDisplayValue('test_secret_12345')).toBeInTheDocument();
    });

    it('should close the dialog on Done and reopen it via the view button', async () => {
      const user = userEvent.setup();
      renderComponent();

      await user.click(screen.getByTestId('agent-client-secret-dialog-done'));
      await waitForElementToBeRemoved(() => screen.queryByTestId('agent-client-secret-dialog'));

      await user.click(screen.getByTestId('agent-client-secret-view'));
      expect(screen.getByTestId('agent-client-secret-dialog')).toBeInTheDocument();
    });
  });
});
