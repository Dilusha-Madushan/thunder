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
import AgentClientSecretDialog, {type AgentClientSecretDialogProps} from '../AgentClientSecretDialog';

vi.mock('@thunderid/hooks', () => ({
  useCopyToClipboard: vi.fn(),
}));

const {useCopyToClipboard} = await import('@thunderid/hooks');

describe('AgentClientSecretDialog', () => {
  const mockOnClose = vi.fn();
  const mockCopy = vi.fn().mockResolvedValue(undefined);

  const defaultProps: AgentClientSecretDialogProps = {
    open: true,
    clientId: 'client-id-xyz',
    clientSecret: 'test_secret_12345',
    onClose: mockOnClose,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useCopyToClipboard).mockReturnValue({
      copied: false,
      copy: mockCopy,
    });
  });

  const renderComponent = (props: Partial<AgentClientSecretDialogProps> = {}) =>
    render(<AgentClientSecretDialog {...defaultProps} {...props} />);

  it('should render the title and subtitle when open', () => {
    renderComponent();

    expect(screen.getByRole('heading', {level: 2, name: /save your client secret/i})).toBeInTheDocument();
    expect(screen.getByText(/copy it and store it somewhere safe/i)).toBeInTheDocument();
  });

  it('should not render when closed', () => {
    renderComponent({open: false});

    expect(screen.queryByTestId('agent-client-secret-dialog')).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', {name: /save your client secret/i})).not.toBeInTheDocument();
  });

  it('should render the client ID field when provided', () => {
    renderComponent();

    expect(screen.getByText('Client ID')).toBeInTheDocument();
    expect(screen.getByDisplayValue('client-id-xyz')).toBeInTheDocument();
  });

  it('should not render the client ID field when not provided', () => {
    renderComponent({clientId: undefined});

    expect(screen.queryByText('Client ID')).not.toBeInTheDocument();
  });

  it('should render the client secret as a masked read-only field', () => {
    renderComponent();

    const input = screen.getByDisplayValue('test_secret_12345');
    expect(input).toHaveAttribute('type', 'password');
    expect(input).toHaveAttribute('readonly');
  });

  it('should toggle the client secret visibility', async () => {
    const user = userEvent.setup();
    renderComponent();

    const input = screen.getByDisplayValue('test_secret_12345');
    expect(input).toHaveAttribute('type', 'password');

    await user.click(screen.getByRole('button', {name: /show or hide client secret/i}));
    expect(input).toHaveAttribute('type', 'text');
  });

  it('should copy the secret when the prominent copy button is clicked', async () => {
    const user = userEvent.setup();
    renderComponent();

    await user.click(screen.getByTestId('agent-client-secret-dialog-copy'));

    await waitFor(() => {
      expect(mockCopy).toHaveBeenCalledWith('test_secret_12345');
    });
  });

  it('should show the copied confirmation state', () => {
    vi.mocked(useCopyToClipboard).mockReturnValue({
      copied: true,
      copy: mockCopy,
    });

    renderComponent();

    const copyButton = screen.getByTestId('agent-client-secret-dialog-copy');
    expect(copyButton).toHaveTextContent(/copied/i);
    expect(copyButton).toBeDisabled();
    expect(copyButton.querySelector('svg.lucide-check')).toBeInTheDocument();
  });

  it('should call onClose when Done is clicked', async () => {
    const user = userEvent.setup();
    renderComponent();

    await user.click(screen.getByTestId('agent-client-secret-dialog-done'));

    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('should render the security reminder', () => {
    renderComponent();

    expect(screen.getByText(/you won't be able to see this secret again/i)).toBeInTheDocument();
  });
});
