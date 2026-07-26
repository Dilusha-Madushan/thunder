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

import type {McpDiscoveryEndpointRow, McpDiscoveryEndpoints} from '../../applications/models/mcp-client';

/**
 * Derives the labeled, copyable discovery endpoint rows shown on the agent onboarding
 * completion screen (`ShowClientSecret`): the issuer, OIDC discovery URL, authorization
 * endpoint, and token endpoint. The token endpoint covers both the agent's own
 * `client_credentials` identity and the token-exchange OBO flow; the authorization endpoint
 * covers the delegated OBO paths. Rows whose value is missing are omitted.
 *
 * @param wellKnown - The parsed OIDC discovery document, or `null`/`undefined` if not yet loaded
 * @param t - Translation function used to resolve each row's label, with an inline English fallback
 * @returns The endpoint rows with a value present
 *
 * @public
 */
export default function getAgentDiscoveryEndpointRows(
  wellKnown: McpDiscoveryEndpoints | null | undefined,
  t: (key: string, fallback: string) => string,
): McpDiscoveryEndpointRow[] {
  const issuer = wellKnown?.issuer;
  const authorizationEndpoint = wellKnown?.authorization_endpoint;
  const tokenEndpoint = wellKnown?.token_endpoint;
  const oidcDiscoveryUrl = issuer ? `${issuer}/.well-known/openid-configuration` : undefined;

  const allEndpointRows: {key: string; label: string; value?: string}[] = [
    {key: 'issuer', label: t('agents:clientSecret.endpoints.issuer', 'Issuer'), value: issuer},
    {
      key: 'oidcDiscovery',
      label: t('agents:clientSecret.endpoints.oidcDiscovery', 'OpenID Connect discovery'),
      value: oidcDiscoveryUrl,
    },
    {
      key: 'authorize',
      label: t('agents:clientSecret.endpoints.authorize', 'Authorization endpoint'),
      value: authorizationEndpoint,
    },
    {key: 'token', label: t('agents:clientSecret.endpoints.token', 'Token endpoint'), value: tokenEndpoint},
  ];

  return allEndpointRows.filter((row): row is McpDiscoveryEndpointRow => Boolean(row.value));
}
