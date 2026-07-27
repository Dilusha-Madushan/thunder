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
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  IconButton,
  InputAdornment,
  Stack,
  TextField,
  Typography,
} from '@wso2/oxygen-ui';
import {AlertTriangle, Check, Copy, Eye, EyeOff} from '@wso2/oxygen-ui-icons-react';
import type {JSX} from 'react';
import {useState} from 'react';
import {useTranslation} from 'react-i18next';
import CopyableField from '../../../applications/components/common/CopyableField';

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

export interface AgentClientSecretDialogProps {
  open: boolean;
  clientId?: string;
  clientSecret: string;
  onClose: () => void;
}

export default function AgentClientSecretDialog({
  open,
  clientId = undefined,
  clientSecret,
  onClose,
}: AgentClientSecretDialogProps): JSX.Element {
  const {t} = useTranslation();
  const [showSecret, setShowSecret] = useState(false);
  const {copied, copy} = useCopyToClipboard({resetDelay: 2000}) as {
    copied: boolean;
    copy: (text: string) => Promise<void>;
  };

  const copyLabel = t('common:actions.copy');
  const clientIdLabel = t('agents:clientSecret.clientIdLabel', 'Client ID');

  const handleCopy = (): void => {
    copy(clientSecret).catch(() => null);
  };

  const handleClose = (): void => {
    setShowSecret(false);
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth data-testid="agent-client-secret-dialog">
      <DialogContent>
        <Stack direction="column" spacing={3} sx={{width: '100%', pt: 2}}>
          <Stack direction="column" spacing={1} sx={{textAlign: 'center'}}>
            <Typography variant="h5" component="h2">
              {t('agents:clientSecret.saveTitle', 'Save your client secret')}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {t(
                'agents:clientSecret.saveSubtitle',
                "This secret won't be shown again. Copy it and store it somewhere safe.",
              )}
            </Typography>
          </Stack>

          <Box sx={cardSx}>
            <Stack direction="column" spacing={2}>
              {clientId && (
                <CopyableField
                  id="agent-client-secret-dialog-client-id"
                  label={clientIdLabel}
                  value={clientId}
                  copyAriaLabel={`${copyLabel} ${clientIdLabel}`}
                />
              )}

              <Box sx={secretHighlightSx}>
                <Typography variant="caption" color="text.secondary" sx={{display: 'block', mb: 1}}>
                  {t('agents:clientSecret.clientSecretLabel', 'Client Secret')}
                </Typography>
                <TextField
                  fullWidth
                  data-testid="agent-client-secret-dialog-value"
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
                      </InputAdornment>
                    ),
                    sx: {fontFamily: 'monospace', fontSize: '0.875rem'},
                  }}
                />
              </Box>
            </Stack>
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
      </DialogContent>
      <DialogActions sx={{px: 3, pb: 3, pt: 1}}>
        <Stack direction="row" spacing={2} sx={{width: '100%'}}>
          <Button
            data-testid="agent-client-secret-dialog-copy"
            variant="contained"
            fullWidth
            startIcon={copied ? <Check size={16} /> : <Copy size={16} />}
            onClick={handleCopy}
            disabled={copied}
          >
            {copied
              ? t('agents:clientSecret.copied', 'Copied')
              : t('agents:clientSecret.copySecret', 'Copy client secret')}
          </Button>
          <Button data-testid="agent-client-secret-dialog-done" variant="outlined" fullWidth onClick={handleClose}>
            {t('common:actions.done', 'Done')}
          </Button>
        </Stack>
      </DialogActions>
    </Dialog>
  );
}
