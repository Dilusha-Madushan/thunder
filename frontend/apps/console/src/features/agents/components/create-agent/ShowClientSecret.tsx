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

import {useCopyToClipboard} from '@thunderid/hooks';
import {useThunderID} from '@thunderid/react';
import {Box, Typography, Stack, TextField, IconButton, InputAdornment, Alert, Button, Divider} from '@wso2/oxygen-ui';
import {Check, Copy, Eye, EyeOff, AlertTriangle, Info} from '@wso2/oxygen-ui-icons-react';
import type {JSX} from 'react';
import {useState} from 'react';
import {useTranslation} from 'react-i18next';
import CopyableField from '../../../applications/components/common/CopyableField';
import type {McpDiscoveryEndpoints} from '../../../applications/models/mcp-client';
import getAgentDiscoveryEndpointRows from '../../utils/getAgentDiscoveryEndpointRows';

const cardSx = {
  p: 3,
  bgcolor: 'background.paper',
  border: '1px solid',
  borderColor: 'divider',
  borderRadius: 1,
} as const;

const secretHighlightSx = {
  p: 2,
  borderRadius: 1,
  bgcolor: 'action.hover',
  borderLeft: '3px solid',
  borderColor: 'primary.main',
} as const;

export interface ShowClientSecretProps {
  agentName: string;
  agentId?: string;
  clientId?: string;
  clientSecret: string;
  onContinue: () => void;
}

export default function ShowClientSecret({
  agentName,
  agentId = undefined,
  clientId = undefined,
  clientSecret,
  onContinue,
}: ShowClientSecretProps): JSX.Element {
  const {t} = useTranslation();
  const {discovery} = useThunderID();
  const [showSecret, setShowSecret] = useState(false);
  const {copied, copy} = useCopyToClipboard({resetDelay: 2000}) as {
    copied: boolean;
    copy: (text: string) => Promise<void>;
  };

  const wellKnown = (discovery as {wellKnown?: McpDiscoveryEndpoints | null} | undefined)?.wellKnown;
  const endpointRows = getAgentDiscoveryEndpointRows(wellKnown, t);
  const copyLabel = t('common:actions.copy');

  const handleCopy = async (): Promise<void> => {
    await copy(clientSecret);
  };

  return (
    <Stack direction="column" spacing={4} sx={{width: '100%'}} data-testid="agent-show-client-secret">
      <Box
        sx={{
          width: 64,
          height: 64,
          borderRadius: 2,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          alignSelf: 'center',
        }}
      >
        <AlertTriangle size={64} color="var(--mui-palette-warning-main)" />
      </Box>

      <Stack direction="column" spacing={1} sx={{textAlign: 'center'}}>
        <Typography variant="h3" component="h1">
          {t('agents:clientSecret.saveTitle', 'Save your client secret')}
        </Typography>
        <Typography variant="body1" color="text.secondary">
          {t(
            'agents:clientSecret.saveSubtitle',
            "This secret won't be shown again. Copy it and store it somewhere safe.",
          )}
        </Typography>
      </Stack>

      <Box sx={cardSx}>
        <Typography variant="subtitle1" sx={{fontWeight: 600, mb: 2}}>
          {t('agents:clientSecret.agentInfoTitle', 'Agent')}
        </Typography>
        <Stack direction="column" spacing={2}>
          <Box>
            <Typography variant="caption" color="text.secondary" sx={{display: 'block', mb: 0.5}}>
              {t('agents:clientSecret.agentNameLabel', 'Agent name')}
            </Typography>
            <Typography variant="body1">{agentName}</Typography>
          </Box>

          {agentId && (
            <>
              <Divider />
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{display: 'block', mb: 0.5}}>
                  {t('agents:clientSecret.agentIdLabel', 'Agent ID')}
                </Typography>
                <Typography variant="body1" sx={{fontFamily: 'monospace', fontSize: '0.875rem'}}>
                  {agentId}
                </Typography>
              </Box>
            </>
          )}
        </Stack>
      </Box>

      <Box sx={cardSx}>
        <Typography variant="subtitle1" sx={{fontWeight: 600, mb: 2}}>
          {t('agents:clientSecret.credentialsTitle', 'Credentials')}
        </Typography>
        <Stack direction="column" spacing={2}>
          <Alert severity="info" icon={<Info size={20} />}>
            <Typography variant="body2">
              {t(
                'agents:clientSecret.credentialsInfo',
                'Your agent authenticates with these credentials to obtain access tokens.',
              )}
            </Typography>
          </Alert>

          {clientId && (
            <Box>
              <Typography variant="caption" color="text.secondary" sx={{display: 'block', mb: 0.5}}>
                {t('agents:clientSecret.clientIdLabel', 'Client ID')}
              </Typography>
              <Typography variant="body1" sx={{fontFamily: 'monospace', fontSize: '0.875rem'}}>
                {clientId}
              </Typography>
            </Box>
          )}

          {clientId && <Divider />}

          <Box sx={secretHighlightSx}>
            <Typography variant="caption" color="text.secondary" sx={{display: 'block', mb: 1}}>
              {t('agents:clientSecret.clientSecretLabel', 'Client Secret')}
            </Typography>
            <TextField
              fullWidth
              data-testid="agent-client-secret-value"
              type={showSecret ? 'text' : 'password'}
              value={clientSecret}
              InputProps={{
                readOnly: true,
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      aria-label={t('agents:clientSecret.toggleVisibility', 'Show or hide client secret')}
                      onClick={() => setShowSecret(!showSecret)}
                      edge="end"
                      size="small"
                    >
                      {showSecret ? <EyeOff size={16} /> : <Eye size={16} />}
                    </IconButton>
                    <IconButton
                      aria-label={`${copyLabel} ${t('agents:clientSecret.clientSecretLabel', 'Client Secret')}`}
                      onClick={() => {
                        handleCopy().catch(() => null);
                      }}
                      edge="end"
                      size="small"
                      sx={{ml: 0.5}}
                    >
                      {copied ? <Check size={16} /> : <Copy size={16} />}
                    </IconButton>
                  </InputAdornment>
                ),
              }}
            />
          </Box>

          <Alert severity="warning" icon={<AlertTriangle size={20} />}>
            <Typography variant="body2" sx={{fontWeight: 'medium', mb: 1}}>
              {t('agents:clientSecret.securityReminder.title', "You won't be able to see this secret again")}
            </Typography>
            <Typography variant="body2">
              {t(
                'agents:clientSecret.securityReminder.description',
                'Store the client secret somewhere safe. If you lose it, you will need to regenerate it from the agent settings.',
              )}
            </Typography>
          </Alert>
        </Stack>
      </Box>

      {endpointRows.length > 0 && (
        <Box sx={cardSx}>
          <Typography variant="subtitle1" sx={{fontWeight: 600, mb: 2}}>
            {t('agents:clientSecret.endpoints.title', 'Endpoints')}
          </Typography>
          <Stack direction="column" spacing={2}>
            {endpointRows.map((row) => (
              <CopyableField
                key={row.key}
                id={`agent-client-secret-endpoint-${row.key}`}
                label={row.label}
                value={row.value}
                copyAriaLabel={`${copyLabel} ${row.label}`}
              />
            ))}
          </Stack>
        </Box>
      )}

      <Button data-testid="agent-client-secret-continue" variant="contained" fullWidth onClick={onContinue}>
        {t('common:actions.continue')}
      </Button>
    </Stack>
  );
}
