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

import {useThunderID} from '@thunderid/react';
import {Alert, Box, Button, Divider, Stack, Typography} from '@wso2/oxygen-ui';
import {CheckCircle, Info} from '@wso2/oxygen-ui-icons-react';
import type {JSX} from 'react';
import {useState} from 'react';
import {useTranslation} from 'react-i18next';
import AgentClientSecretDialog from './AgentClientSecretDialog';
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
  const [secretDialogOpen, setSecretDialogOpen] = useState(true);

  const wellKnown = (discovery as {wellKnown?: McpDiscoveryEndpoints | null} | undefined)?.wellKnown;
  const endpointRows = getAgentDiscoveryEndpointRows(wellKnown, t);
  const copyLabel = t('common:actions.copy');
  const clientIdLabel = t('agents:clientSecret.clientIdLabel', 'Client ID');

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
        <CheckCircle size={64} color="var(--mui-palette-success-main)" />
      </Box>

      <Stack direction="column" spacing={1} sx={{textAlign: 'center'}}>
        <Typography variant="h3" component="h1">
          {t('agents:clientSecret.createdTitle', 'Agent created')}
        </Typography>
        <Typography variant="body1" color="text.secondary">
          {t('agents:clientSecret.createdSubtitle', 'Your agent is ready. Save its client secret before you continue.')}
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
            <CopyableField
              id="agent-client-secret-client-id"
              label={clientIdLabel}
              value={clientId}
              copyAriaLabel={`${copyLabel} ${clientIdLabel}`}
            />
          )}

          <Divider />

          <Box>
            <Typography variant="caption" color="text.secondary" sx={{display: 'block', mb: 1}}>
              {t('agents:clientSecret.clientSecretLabel', 'Client Secret')}
            </Typography>
            <Stack
              direction="row"
              spacing={2}
              useFlexGap
              flexWrap="wrap"
              sx={{alignItems: 'center', justifyContent: 'space-between'}}
            >
              <Typography variant="body2" color="text.secondary" sx={{flex: 1, minWidth: 200}}>
                {t(
                  'agents:clientSecret.secretHiddenNote',
                  'The client secret is shown only once. Regenerate it from the agent settings if you lose it.',
                )}
              </Typography>
              <Button
                data-testid="agent-client-secret-view"
                variant="outlined"
                size="small"
                onClick={() => setSecretDialogOpen(true)}
              >
                {t('agents:clientSecret.viewSecret', 'View client secret')}
              </Button>
            </Stack>
          </Box>
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

      <AgentClientSecretDialog
        open={secretDialogOpen}
        clientId={clientId}
        clientSecret={clientSecret}
        onClose={() => setSecretDialogOpen(false)}
      />
    </Stack>
  );
}
