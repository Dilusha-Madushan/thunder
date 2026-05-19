/*
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
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

import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import InputLabel from '@mui/material/InputLabel';
import OutlinedInput from '@mui/material/OutlinedInput';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Layout from '../components/Layout';
import { submitNativeAuth } from '../services/authService';

interface AuthInput {
    identifier: string;
    type: string;
    required: boolean;
}

interface ActionPrompt {
    ref: string;
    nextNode?: string;
}

interface FlowResponse {
    flowStatus?: string;
    failureReason?: string;
    type?: string;
    executionId?: string;
    challengeToken?: string;
    data?: {
        inputs?: AuthInput[];
        actions?: ActionPrompt[];
    };
}

/**
 * InvitePage handles the recovery email link callback.
 * Reads executionId and inviteToken from URL params, submits them to verify
 * the recovery token, then renders the new password form.
 */
const InvitePage = () => {

    const navigate = useNavigate();
    const isInitialized = useRef(false);

    const [loading, setLoading] = useState<boolean>(true);
    const [step, setStep] = useState<'verifying' | 'password' | 'success' | 'error'>('verifying');
    const [errorMessage, setErrorMessage] = useState<string>('');

    const [executionId, setExecutionId] = useState<string>('');
    const [challengeToken, setChallengeToken] = useState<string>('');
    const [availableActions, setAvailableActions] = useState<ActionPrompt[]>([]);
    const [formData, setFormData] = useState<Record<string, string>>({});
    const [showPassword, setShowPassword] = useState<boolean>(false);

    const handleResponse = (data: FlowResponse) => {
        if (data.flowStatus === 'COMPLETE') {
            setStep('success');
            setLoading(false);
            return;
        }
        if (data.flowStatus === 'ERROR') {
            setStep('error');
            setErrorMessage(data.failureReason || 'Recovery failed. Please request a new reset link.');
            setLoading(false);
            return;
        }
        if (data.type === 'VIEW') {
            setExecutionId(data.executionId || '');
            setChallengeToken(data.challengeToken || '');
            setAvailableActions(data.data?.actions || []);
            setStep('password');
            setLoading(false);
        }
    };

    useEffect(() => {
        if (isInitialized.current) return;
        isInitialized.current = true;

        const params = new URLSearchParams(window.location.search);
        const execId = params.get('executionId') || '';
        const inviteToken = params.get('inviteToken') || '';

        if (!execId || !inviteToken) {
            setStep('error');
            setErrorMessage('Invalid recovery link. Please request a new password reset.');
            setLoading(false);
            return;
        }

        // SkipChallengeValidation is true for the verify step — no challenge token needed
        submitNativeAuth(execId, { inviteToken }, undefined, undefined)
            .then((result) => {
                handleResponse(result.data);
            })
            .catch(() => {
                setStep('error');
                setErrorMessage('Invalid or expired recovery link. Please request a new password reset.');
                setLoading(false);
            });
    }, []);

    const handleInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = event.target;
        setFormData(prev => ({ ...prev, [name]: value }));
    };

    const handleSubmitPassword = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        setLoading(true);

        const actionRef = availableActions.length > 0 ? availableActions[0].ref : undefined;
        submitNativeAuth(executionId, formData, actionRef, challengeToken)
            .then((result) => {
                handleResponse(result.data);
            })
            .catch(() => {
                setStep('error');
                setErrorMessage('Failed to reset password. Please try again.');
                setLoading(false);
            });
    };

    const GradientCircularProgress = () => (
        <>
            <svg width={0} height={0}>
                <defs>
                    <linearGradient id="invite_gradient" x1="0%" y1="0%" x2="0%" y2="100%">
                        <stop offset="0%" stopColor="#fc4700" />
                        <stop offset="100%" stopColor="#f87643" />
                    </linearGradient>
                </defs>
            </svg>
            <CircularProgress sx={{ 'svg circle': { stroke: 'url(#invite_gradient)' } }} />
        </>
    );

    const renderContent = () => {
        if (step === 'verifying') {
            return (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                    <GradientCircularProgress />
                    <Typography sx={{ mt: 2 }}>Verifying your recovery link...</Typography>
                </Box>
            );
        }

        if (step === 'error') {
            return (
                <Box>
                    <Box sx={{ mb: 4 }}>
                        <Typography variant="h5" gutterBottom>
                            Password Reset
                        </Typography>
                    </Box>
                    <Alert severity="error" sx={{ mb: 3 }}>
                        {errorMessage}
                    </Alert>
                    <Button
                        variant="contained"
                        color="primary"
                        fullWidth
                        onClick={() => navigate('/')}
                    >
                        Back to Login
                    </Button>
                </Box>
            );
        }

        if (step === 'success') {
            return (
                <Box sx={{ textAlign: 'center', py: 2 }}>
                    <Typography variant="h3" sx={{ mb: 2 }}>✅</Typography>
                    <Typography variant="h5" gutterBottom>
                        Password Reset Successful
                    </Typography>
                    <Typography sx={{ mb: 3 }}>
                        Your password has been reset. You can now sign in with your new password.
                    </Typography>
                    <Button
                        variant="contained"
                        color="primary"
                        fullWidth
                        onClick={() => navigate('/')}
                    >
                        Sign In
                    </Button>
                </Box>
            );
        }

        // step === 'password'
        return (
            <Box>
                <Box sx={{ mb: 4 }}>
                    <Typography variant="h5" gutterBottom>
                        Set New Password
                    </Typography>
                    <Typography>
                        Enter a new password for your account.
                    </Typography>
                </Box>
                <form onSubmit={handleSubmitPassword}>
                    <Box display="flex" flexDirection="column" gap={2}>
                        <Box display="flex" flexDirection="column" gap={0.5}>
                            <InputLabel htmlFor="password">New Password</InputLabel>
                            <OutlinedInput
                                type={showPassword ? 'text' : 'password'}
                                id="password"
                                name="password"
                                placeholder="Enter your new password"
                                size="small"
                                value={formData.password || ''}
                                onChange={handleInputChange}
                                required
                                endAdornment={
                                    <InputAdornment position="end">
                                        <IconButton
                                            aria-label="toggle password visibility"
                                            onClick={() => setShowPassword(prev => !prev)}
                                            onMouseDown={(e) => e.preventDefault()}
                                            edge="end"
                                        >
                                            {showPassword ? <VisibilityOff /> : <Visibility />}
                                        </IconButton>
                                    </InputAdornment>
                                }
                            />
                        </Box>
                        <Button
                            variant="contained"
                            color="primary"
                            type="submit"
                            fullWidth
                            disabled={loading}
                            sx={{ mt: 2 }}
                        >
                            {loading ? <CircularProgress size={24} /> : 'Reset Password'}
                        </Button>
                    </Box>
                </form>
            </Box>
        );
    };

    return (
        <Layout>
            {loading && step === 'verifying' ? (
                <GradientCircularProgress />
            ) : (
                <Grid size={{ xs: 12, md: 6 }}>
                    <Paper
                        sx={{
                            display: 'flex',
                            width: '100%',
                            height: '100%',
                            flexDirection: 'column',
                        }}
                    >
                        <Box
                            sx={{
                                alignItems: 'center',
                                justifyContent: 'center',
                                padding: 6,
                                width: '100%',
                                maxWidth: 500,
                                margin: 'auto',
                            }}
                        >
                            {renderContent()}
                            <Box component="footer" sx={{ mt: 6 }}>
                                <Typography sx={{ textAlign: 'center' }}>
                                    © Copyright {new Date().getFullYear()}
                                </Typography>
                            </Box>
                        </Box>
                    </Paper>
                </Grid>
            )}
        </Layout>
    );
};

export default InvitePage;
