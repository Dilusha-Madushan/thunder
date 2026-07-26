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
import {render, screen, waitFor} from '@thunderid/test-utils';
import {describe, it, expect, beforeEach, vi} from 'vitest';
import ShowClientSecret, {type ShowClientSecretProps} from '../ShowClientSecret';

// Mock the useCopyToClipboard hook
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

  describe('rendering', () => {
    it('should render the title and subtitle', () => {
      renderComponent();

      expect(screen.getByRole('heading', {level: 1, name: /save your client secret/i})).toBeInTheDocument();
      expect(screen.getByText(/store it somewhere safe/i)).toBeInTheDocument();
    });

    it('should display the agent name', () => {
      renderComponent();

      expect(screen.getByText('Agent name')).toBeInTheDocument();
      expect(screen.getByText('Test Agent')).toBeInTheDocument();
    });

    it('should display the agent ID when provided', () => {
      renderComponent();

      expect(screen.getByText('Agent ID')).toBeInTheDocument();
      expect(screen.getByText('agent-id-abc')).toBeInTheDocument();
    });

    it('should not display the agent ID field when not provided', () => {
      renderComponent({agentId: undefined});

      expect(screen.queryByText('Agent ID')).not.toBeInTheDocument();
    });

    it('should display the clientId when provided', () => {
      renderComponent();

      expect(screen.getByText('Client ID')).toBeInTheDocument();
      expect(screen.getByText('client-id-xyz')).toBeInTheDocument();
    });

    it('should not display the clientId field when not provided', () => {
      renderComponent({clientId: undefined});

      expect(screen.queryByText('Client ID')).not.toBeInTheDocument();
    });

    it('should render the client secret field as masked password', () => {
      renderComponent();

      const input = screen.getByDisplayValue('test_secret_12345');
      expect(input).toBeInTheDocument();
      expect(input).toHaveAttribute('type', 'password');
      expect(input).toHaveAttribute('readonly');
    });

    it('should render the credentials info note', () => {
      renderComponent();

      expect(
        screen.getByText(/your agent authenticates with these credentials to obtain access tokens/i),
      ).toBeInTheDocument();
    });

    it('should render security reminder alert', () => {
      renderComponent();

      expect(screen.getByText(/you won't be able to see this secret again/i)).toBeInTheDocument();
      expect(screen.getByText(/store the client secret somewhere safe/i)).toBeInTheDocument();
    });

    it('should render the inline copy affordance and a single continue action', () => {
      renderComponent();

      expect(screen.getByRole('button', {name: /copy client secret/i})).toBeInTheDocument();
      expect(screen.getByTestId('agent-client-secret-continue')).toBeInTheDocument();
    });
  });

  describe('endpoints', () => {
    it('should render the discovery endpoint fields from the well-known document', () => {
      renderComponent();

      expect(screen.getByText('Issuer')).toBeInTheDocument();
      expect(screen.getByDisplayValue('https://localhost:8090')).toBeInTheDocument();
      expect(screen.getByText('OpenID Connect discovery')).toBeInTheDocument();
      expect(screen.getByDisplayValue('https://localhost:8090/.well-known/openid-configuration')).toBeInTheDocument();
      expect(screen.getByText('Authorization endpoint')).toBeInTheDocument();
      expect(screen.getByDisplayValue('https://localhost:8090/oauth2/authorize')).toBeInTheDocument();
      expect(screen.getByText('Token endpoint')).toBeInTheDocument();
      expect(screen.getByDisplayValue('https://localhost:8090/oauth2/token')).toBeInTheDocument();
    });

    it('should not render the endpoints card when discovery is unavailable', () => {
      mockUseThunderID.mockReturnValue({discovery: {wellKnown: null}});
      renderComponent();

      expect(screen.queryByText('Token endpoint')).not.toBeInTheDocument();
      expect(screen.queryByText('Issuer')).not.toBeInTheDocument();
    });
  });

  describe('visibility toggle', () => {
    it('should toggle client secret visibility when eye icon is clicked', async () => {
      const user = userEvent.setup();
      renderComponent();

      const input = screen.getByDisplayValue('test_secret_12345');
      expect(input).toHaveAttribute('type', 'password');

      const visibilityButton = screen.getByRole('button', {name: /show or hide client secret/i});

      await user.click(visibilityButton);

      expect(input).toHaveAttribute('type', 'text');

      await user.click(visibilityButton);

      expect(input).toHaveAttribute('type', 'password');
    });
  });

  describe('copy functionality', () => {
    it('should copy the secret when the inline copy button is clicked', async () => {
      const user = userEvent.setup();
      renderComponent();

      const copyButton = screen.getByRole('button', {name: /copy client secret/i});
      await user.click(copyButton);

      await waitFor(() => {
        expect(mockCopy).toHaveBeenCalledWith('test_secret_12345');
      });
    });

    it('should show the copied confirmation icon when copied is true', () => {
      vi.mocked(useCopyToClipboard).mockReturnValue({
        copied: true,
        copy: mockCopy,
      });

      renderComponent();

      const copyButton = screen.getByRole('button', {name: /copy client secret/i});
      expect(copyButton.querySelector('svg.lucide-check')).toBeInTheDocument();
    });

    it('should configure useCopyToClipboard with resetDelay 2000', () => {
      renderComponent();

      const hookCall = vi.mocked(useCopyToClipboard).mock.calls[0][0];
      expect(hookCall).toHaveProperty('resetDelay', 2000);
    });
  });

  describe('continue action', () => {
    it('should call onContinue when continue button is clicked', async () => {
      const user = userEvent.setup();
      renderComponent();

      const continueButton = screen.getByTestId('agent-client-secret-continue');
      await user.click(continueButton);

      expect(mockOnContinue).toHaveBeenCalledTimes(1);
    });
  });
});
