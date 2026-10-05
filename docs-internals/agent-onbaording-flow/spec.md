# Agent Onboarding Flow Specification

- **Status:** Draft
- **Version:** 0.3

## Summary

Agent onboarding in ThunderID runs as a configurable administration flow. An administrator selects where the agent belongs, assigns its owner, supplies its details, and receives its client credentials. The flow determines the steps and validation paths; the Console renders them. Changing the onboarding journey therefore does not require rebuilding a fixed creation wizard.

The feature supports agents that act under their own identity and agents configured to act on behalf of a signed-in user. Both are created by an authorized administrator. Agent self-registration is outside this feature.

This work also establishes the creation boundary needed by the flow engine. Users and agents are created through their category management providers, which call the corresponding management services. Those services own validation and resource creation. The flow engine collects information and coordinates steps without taking over those responsibilities.

## Problem and intended outcome

The previous agent creation wizard defined its screens and fields in Console code. That made the UI responsible for the onboarding sequence and required frontend changes whenever that sequence changed. ThunderID already had a flow engine for collecting input and coordinating identity operations, but its provisioning path was built around users.

Adding an agent branch to entity creation would not have solved the whole problem. An agent can have an inbound client and an authentication profile as well as its entity record. The agent management service already coordinates those resources. A flow that created only the entity would bypass that process; a flow that created each resource itself would duplicate it.

The intended outcome is one configurable onboarding journey backed by the existing management rules. Flow authors control what the administrator is asked and when. Management services decide whether the resulting resource is valid. The SDK and Console share the data-loading behavior needed to present management resources as choices.

For example, an administrator can create a service agent with its own client credentials using the default flow. A flow for an assistant acting for a user also asks for callback URIs, a sign-in flow, and an allowed user type. Both journeys use the same provisioning executor and agent service.

## Scope

The feature includes the following:

- User and agent management providers for runtime creation, and removal of creation from the entity provider.
- Shared provisioning and uniqueness behavior for users and agents, with agent type and owner resolution.
- An authorized `ADMINISTRATION` flow for agent onboarding, its default configuration, and the Console entry point.
- Agent name as a schema attribute, including the affected API, display, token, and declarative-resource consumers.
- On-behalf-of configuration, delegation defaults in the agent service, and flow-builder resources for the supported onboarding journeys.
- Reusable selection loading in the JavaScript SDKs and corresponding Console adapters for users, authentication flows, and user types.

The following remain outside the feature:

- Agent self-registration or anonymous onboarding.
- A full replacement of `EntityProvider` or migration of every runtime dependency to a new management provider.
- New OAuth grant implementations or enforcement of application agent-type restrictions across every OAuth path.
- Selecting multiple allowed user types during onboarding.
- A general redesign of agent editing. The schema-name change and forwarding delegation intent on saves are included where the changed contracts require them.
- Unrelated UI changes, including copy-control styling and reusable masking components.
- Claims that all mobile SDKs have adopted the final JavaScript selection design.

## Approach

### Put creation behind the category management service

The first decision was architectural. Runtime components, including the flow engine and OAuth runtime, should reach management capabilities through providers. Management services own the rules for their resource category and may call the underlying services needed to enforce them.

For this feature, that decision produces two creation contracts: `UserMgtProvider.CreateUser` and `AgentMgtProvider.CreateAgent`. The default implementations call the user and agent management services. `EntityProvider` no longer exposes creation, so a provisioning executor cannot use it as an alternative route around those services.

The distinction matters most for agents. The executor submits the agent details and requested configuration to `AgentMgtProvider`. The agent service validates them, resolves a missing owner, and coordinates the entity, inbound client, and authentication profile. It also owns compensation if a later resource-creation operation fails. The provider passes the request and result across the runtime boundary; it does not implement a second agent-creation policy.

User provisioning follows the same pattern. Existing user flows retain their category and collection behavior, while creation reaches the user service through `UserMgtProvider`. The application management service also uses the entity service internally rather than depending on the runtime entity provider.

This is a creation-boundary change, not the removal of the entire entity provider. Existing identification, retrieval, update, deletion, and credential operations remain available there. The broader provider architecture is a direction for subsequent work, not a claim that this feature has migrated every service dependency.

| Layer | Responsibility in onboarding |
|---|---|
| Console or SDK flow renderer | Display the current step, load selection candidates, and submit input. |
| Flow executors | Resolve selections, collect missing values, check preliminary uniqueness, and request creation. |
| Category management provider | Expose the creation capability to the runtime and delegate to its management service. |
| Category management service | Apply business validation and coordinate creation of the complete resource. |
| Entity and inbound services | Perform the underlying operations requested by the management service. |

### Extend the existing provisioning path

Users and agents share the same `ProvisioningExecutor`. A node with `mode: agent` requests agent creation. An absent or empty mode retains user behavior, and `mode: user` selects it explicitly. Unknown modes are rejected.

The shared path handles schema collection, existing-entity checks, creation results, and configured group or role assignments. Category-specific information stays together, including conflict codes and assignment types. The actual create calls remain explicit because the two management contracts represent different resources.

`AttributeUniquenessValidator` follows the same mode convention. `UserTypeResolver` and `AgentTypeResolver` share type-resolution behavior, while `OwnerResolver` supplies the user owner needed for agent onboarding. This keeps the flow mechanics reusable without pretending that every entity category has identical rules.

Errors on these paths identify the relevant category. Attribute-conflict errors from creation are mapped to the flow conflict response; other management-service errors retain their codes and parameters. Consumers such as CIBA compare stable error codes rather than category-dependent message text.

### Make the agent name part of its schema

The initial onboarding work treated the agent name as a system field with its own prompt and validation. That made name different from every other unique schema attribute: the flow uniqueness validator could not check it, and the agent service needed a separate name-conflict path.

The subsequent schema change removes that distinction. The default agent type defines `name` as a required, unique string and uses it as its display attribute. Creation and update requests carry the value in `attributes.name`; responses return it there. There is no top-level agent `name` field in the new contract.

Onboarding now collects name together with the other agent attributes in **Agent Details**. That prompt submits through the schema uniqueness validator before provisioning. The agent service still validates the complete request, including uniqueness, when creation occurs. The precheck gives the administrator an earlier opportunity to correct a conflict; it does not replace creation-time validation.

The change also updates the consumers of an agent name. Console labels, agent display resolution, token attribute selection, and declarative import and export read the schema attributes. The agent package registers its own name index instead of relying on another category to register it. Description and owner remain management fields, and logo remains part of the inbound-client properties.

This contract is a breaking change for callers that send or read top-level `name`. Existing stored names also need upgrade handling before the new readers can use them. The implementation evidence does not establish an automatic migration of old database records; the specification does not claim one is delivered.

### Start with an authorized administrator

Onboarding uses the `ADMINISTRATION` flow type. Starting or continuing an administration flow requires an authenticated caller with the root `system` permission. The flow execution service checks that requirement before running the next step. A runtime context alone is insufficient.

The default flow includes `PermissionValidator`, but authorization does not depend on keeping that node in the definition. A custom administration flow without it is still subject to the execution-service check. Initiation by an explicit flow ID is restricted to administration flows.

Agent self-registration is deliberately excluded. Agent provisioning rejects registration flows before creation, and `AgentTypeResolver` is not offered as a registration capability. An administrator creating an agent that later acts for a user does not change this boundary.

### Resolve the agent type, organization unit, and owner

An entity category says what kind of resource is being created, such as a user or agent. Its type supplies the schema. An organization unit (OU) determines where the resource belongs.

For administration flows, `AgentTypeResolver` obtains agent types from the type catalog. Its `allowedAgentTypes` property can narrow the candidates; an empty list adds no restriction. If an OU is already selected, eligible types must belong to that OU or an ancestor. A submitted selection is checked against the category, allowed list, and OU relationship.

With one eligible type, resolution continues without a selection screen. With several, the flow asks the administrator to select one. With none, resolution fails. The default onboarding flow restricts the type to `default`, resolves it first, and then uses `OUResolverExecutor` to select a compatible target OU.

Administration type selection does not use an application's allowed-agent-type list. That list concerns application behavior, not which agent types an administrator can create. The resolver's authentication-flow check is separate and does not establish enforcement for OAuth grants that never execute the flow.

Owner selection asks for a user through `USER_SELECT`. The client loads the candidate list and submits a user ID. `OwnerResolver` verifies that the entity exists and belongs to the user category, then records the resolved owner. An unknown ID and an ID belonging to another category receive the same selection error. A lookup failure fails execution rather than being presented as a bad selection.

Provisioning reads only the resolved owner value. Submitting an unchecked `owner` or `ownerId` field does not replace it. When the optional choice is omitted, the agent service uses the authenticated caller as owner. The Console preselects the signed-in user to make that default visible.

| Information | Submitted input | Resolved runtime value |
|---|---|---|
| User schema type | `userType` | `categoryType` |
| Agent schema type | `agentType` | `categoryType` |
| Target OU | `ouId` | `ouId` |
| Schema's owning OU | Resolved from the type | `defaultOUID` |
| Agent owner | `owner` | `ownerId` |

Provisioning prefers the resolved `ouId` over `defaultOUID`. The shared `categoryType` value must be interpreted with the node's mode; it is not a category-independent schema identifier. The default flow resolves its target explicitly. An administration flow with no resolved type cannot rely on an application's allowed types or assume the built-in agent type.

### Collect details and provide a correction path

The default details prompt contains name, model provider, model, and function. Name is required by the default schema; the other fields are optional. The provisioning node uses `includeOptional: true` and requests outstanding schema information through its incomplete path.

When the executor generates a missing string attribute with enumeration values, it requests a `SELECT` carrying those values. Other missing attributes use their corresponding input types. This lets a flow collect schema fields without adding field-specific code to the Console page.

The supplied prompts also contain explicitly authored fields. Their model-provider and function options are copied into the flow definition; they do not automatically track later schema changes. Flow authors must keep those options consistent with the schema. Dynamically requested fields and explicitly authored fields therefore have different sources for their options.

The uniqueness validator checks unique attributes from the resolved category's schema against the values submitted at that step. Collecting name and the other details together makes them available to that check. A conflict follows the validator's incomplete connection back to **Agent Details**, identifying the affected attribute.

Provisioning can still fail after the precheck, for example if another request creates a conflicting record. Every supplied onboarding template routes provisioning failure to **Agent Details**. This supports correction of name and schema values. It does not mean every creation error can be corrected on that screen; the limitation for on-behalf-of configuration is described below.

### Configure an agent to act on behalf of a user

The second phase adds the information needed for delegated, or on-behalf-of, agents. An agent acting under its own identity uses its client credentials. The delegated onboarding journey also configures how a user signs in through that agent.

The provisioning node reads delegation intent in this order: a non-empty runtime `delegated` value, a non-empty submitted value, the node's `delegated` property, then `false`. An explicit false value takes precedence over a true property. Collected strings must parse as booleans, while the node property must be a boolean. An unreadable effective value produces an error instead of silently choosing a mode.

The node property is a default, not an immutable policy. The own-identity and on-behalf-of templates omit the choice from their normal screens and supply different defaults. A supplied runtime or submitted value still takes precedence. The generic template asks for the choice, and the Console submits an untouched checkbox as false.

A delegated agent requires three additional inputs:

| Input | What the administrator supplies | Stored meaning |
|---|---|---|
| `redirectUris` | One or more callback URIs, separated by commas. | Redirect URIs on the inbound OAuth client. Whitespace and empty entries are discarded. |
| `authFlowId` | An authentication flow selected by name. | The flow ID on the agent's authentication profile. |
| `allowedUserTypes` | A user type selected by handle. | A one-entry list of allowed user type handles. |

If delegation is enabled and any of these inputs is absent, provisioning requests it as required through its incomplete path. A string containing only commas and whitespace does not count as a callback URI. If delegation is disabled, the executor neither requests nor applies these values, including values left over from an earlier choice.

The dedicated delegated-details prompt marks all three inputs required. The generic choice prompt leaves them optional so an administrator can select own-identity operation without filling them. If the administrator enables delegation without supplying the details, the executor requests what is missing on the following step.

Candidate lists improve selection but do not establish validity. The management services validate the submitted authentication flow, user type, and redirect URIs at creation. A stale or forged selection can still be rejected.

### Keep delegation policy in the agent service

The first provider implementation derived the agent's OAuth settings. The final decision moves that policy into the agent service so flow-based creation and direct management requests use the same rules. The agent provider passes through the request, including its `Delegated` intent.

The onboarding executor always requests an OAuth client because the result screen presents client credentials. For the configuration produced by the flow, the service and inbound defaults give the following result:

| Setting | Own-identity onboarding | Delegated onboarding |
|---|---|---|
| Grant types | `client_credentials` | `client_credentials`, `authorization_code`, and `refresh_token` |
| Client authentication | `client_secret_basic` | `client_secret_basic` |
| Response type | No authorization-code response type required. | `code` |
| PKCE | No authorization-code requirement. | Required. |
| Callback URIs | Not collected. | Supplied by the administrator. |
| Sign-in profile selections | No delegated selections supplied by the executor. | Selected authentication flow and one allowed user type. |

The service also accepts management API requests with explicit inbound settings, so its policy is broader than this flow's defaults. For an OAuth client, it adds `client_credentials` when absent unless the client is public. A non-delegated request without an OAuth configuration does not create a client merely to apply that default.

For a delegated request with no grant beyond client credentials, the service supplies authorization code and refresh token defaults, a code response type when absent, and required PKCE. If other grants are supplied, at least one must be `authorization_code`, `ciba`, or `token_exchange`; otherwise the request is rejected. Other requested grants remain subject to inbound validation. The service does not invent callback URIs or a default logo.

`delegated` is creation or update intent, not a stored agent field and not a response field. The service applies the grant policy on create and update. A full update must supply the intent again to invoke delegated validation. Console saves and secret-regeneration updates derive it from the configured OAuth grants. The current working-tree helper recognizes authorization code, CIBA, and token exchange as delegation grants.

### Reuse selection controls and SDK loaders

Owner and delegated configuration introduced three resource selections: a user, an authentication flow, and a user type. These lists can grow beyond a single response page. Embedding every option in the flow definition would make the definition responsible for a changing management directory.

The flow therefore declares a typed input, and the client loads its candidates. `USER_SELECT`, `AUTH_FLOW_SELECT`, and `USER_TYPE_SELECT` are accepted input types in the engine. They describe how to collect a value; server-side resolvers and management services still decide whether that value is acceptable.

The JavaScript SDK work extends the existing `Select` with an optional `onLoadMore` callback. The select renders the options it receives and asks for another batch. Loading state, cursor advancement, duplicate removal, cancellation, and errors belong to a separate helper, `useInfiniteOptions` in React and the corresponding composable in Vue.

`UserSelect`, `AuthFlowSelect`, and `UserTypeSelect` are thin resource-specific components built on that selection behavior. Their built-in loaders use the SDK's management API functions rather than reimplementing HTTP requests in each screen. A custom data source can use the generic select and loading helper.

| Input type | SDK management operation | Displayed choice | Submitted value |
|---|---|---|---|
| `USER_SELECT` | `getUsers` | User display value, with username or email fallbacks. | User ID. |
| `AUTH_FLOW_SELECT` | `getFlows`, restricted to authentication flows | Flow name. | Flow ID. |
| `USER_TYPE_SELECT` | `getUserTypes` | User type handle. | User type handle. |

User type handles are intentional. Version 0.2's display-name and disambiguation description does not match the chosen implementation. Authentication flow names and user labels are for display; their IDs remain the submitted values.

The Console follows the same composition. Its existing `SelectAdapter` gains loading support, and `UserSelectAdapter`, `AuthFlowSelectAdapter`, and `UserTypeSelectAdapter` wrap it. They reuse the SDK's `useFetchUsers`, `useFetchAuthFlows`, and `useFetchUserTypes` loaders and shared loading helper. The Console owns its Oxygen UI rendering and the event that asks for another page.

The final controls differ in how that event is triggered. The React and Vue SDK selects retain a native select with a **Load more** option. The Console loads the first page when the menu opens and requests more as the administrator reaches the end of the list. Both keep the current selection while more options load. The Console can display the preselected owner's label before that owner's page has loaded.

These management-backed selections require an authenticated context with permission to read the relevant resources. They do not make management lists available to anonymous flows. Failed loads are exposed to the caller; the Console reports them rather than treating a failed request as an empty directory. Pagination removes the earlier onboarding page's one-request, 100-entry limit once the adapter integration is included.

### Let the flow drive the Console journey

The Console reads `flow.agentOnboardingFlow.defaultHandle` from the merged server configuration. The initial handle is `default-agent-onboarding-flow`. It resolves that handle among administration flows, following listing pages rather than assuming the flow is on the first page.

A missing setting, a configured handle with no matching administration flow, and a failed configuration request are separate conditions. The page reports the relevant condition and does not fall back to direct agent creation.

After resolution, the Console starts the flow with its ID and continues with the execution ID, selected action, and inputs. The shared flow renderer supplies the step content. The onboarding page provides its surrounding navigation and the management-resource and OU fields. The OU selector continues to use the Console's organization-unit picker.

The default journey is:

1. Check administrator permission.
2. Resolve the agent type and compatible target OU, prompting when a choice is needed.
3. Resolve the owner, defaulting to the signed-in administrator when omitted.
4. Collect name and other agent details.
5. Check submitted unique attributes and return conflicts to the details prompt.
6. Provision the agent, requesting missing information or returning service errors through the configured connections.
7. Present the agent ID, client ID, and client secret.

The bootstrap flow creates an own-identity agent in its normal journey. Its provisioning node sets `mode: agent`, `delegated: false`, and `includeOptional: true`. Both its incomplete and failure connections return to **Agent Details**.

Successful creation places `agentId`, `clientId`, and `clientSecret` in additional flow response data for the credentials screen. The secret is masked and can be revealed or copied. A later agent read does not retrieve that creation secret. Starting another onboarding run clears the previous run's values; closing returns to the agent list.

Creation-time availability is not a claim that the flow runtime never retains the response during the run. Nor is the whole flow a transaction. The agent service owns compensation for its creation sequence, but a later group assignment, role assignment, or authentication-state failure can occur after the resource exists. Leaving before provisioning creates no agent; leaving after provisioning does not undo creation.

### Provide reusable builder resources

Flow authors use the existing builder and property editor. **Provision User** and **Provision Agent** select the same executor with different modes. Reopening an agent node preserves its agent label and configuration. The builder also exposes agent type resolution, owner resolution, category selection for uniqueness validation, and the delegation property for agent provisioning.

Owner resolution, delegated details, and delegation choice are reusable widgets. Three templates compose them into complete administration flows:

| Template | Behavior after details and uniqueness | Delegation default | Missing-input route |
|---|---|---|---|
| Own-identity agent onboarding | Proceed to provisioning. | `false` | Agent Details. |
| On-behalf-of agent onboarding | Collect callback URIs, authentication flow, and user type as required fields, then provision. | `true` | A dynamic missing-input prompt. |
| Generic agent onboarding | Ask for delegated mode and optional delegated details, then provision. | `false` | A dynamic missing-input prompt. |

All three include permission, agent type, OU, and owner resolution before details, and the credentials screen after creation. All three route provisioning failure to **Agent Details**. The on-behalf-of and generic templates differ in collection behavior and defaults, not in their management-service boundary.

Returning an invalid delegated value to **Agent Details** is a known limitation. That screen does not own the authentication-flow, user-type, or callback-URI inputs. The specification therefore does not promise that every rejected delegated value can be corrected through that route. Server-loaded choices reduce entry mistakes but do not eliminate stale choices, forged submissions, or invalid callback URIs.

## Requirements and acceptance criteria

The approach above explains the design. The following criteria define the behavior to preserve as the separate changes are integrated.

| ID | Requirement | Acceptance criteria |
|---|---|---|
| R1 | Preserve management-service ownership. | User and agent creation call their respective management providers. Default providers call their category services. `EntityProvider` exposes no create operation. Agent resource coordination and compensation remain in the agent service. |
| R2 | Require the configured creation capability. | Missing or disabled management providers cannot produce successful creation or trigger an entity-creation fallback. A creation result without an entity ID is not accepted as success. |
| R3 | Share flow mechanics across categories. | Omitted or empty mode preserves user provisioning; `agent` selects agents; unknown modes fail. Conflict messages and post-creation assignments use the selected category. Other management-service errors retain their identity. |
| R4 | Resolve an eligible type and OU. | One candidate resolves automatically, several prompt, and none fails. Submitted types satisfy the node's restrictions and OU relationship. Provisioning consumes the resolved type and prefers `ouId` over `defaultOUID`, without assuming an agent type. |
| R5 | Verify ownership. | A selected owner must resolve to a user. Missing and wrong-category IDs receive the same selection error. Provisioning consumes only `ownerId` from resolved state. Omission uses the authenticated caller. |
| R6 | Treat name as a schema attribute. | Requests and responses use `attributes.name`. The default schema makes it required, unique, and the display attribute. The details prompt submits name with other attributes through uniqueness validation, and creation validates the complete request again. |
| R7 | Centralize agent configuration policy. | Flow-created agents receive the own-identity or delegated defaults described above. The provider does not derive grant policy. Direct create and update requests use the same service policy, subject to their supplied inbound settings. |
| R8 | Restrict the onboarding caller. | Administration initiation and continuation reject unauthenticated callers and callers without root `system` permission, even without a permission node. Registration-flow agent provisioning creates no agent. |
| R9 | Run the configured Console flow. | The Console resolves the merged handle across all administration-flow listing pages, runs the selected flow, and distinguishes missing configuration, missing flow, and request failure. It has no direct-create fallback. |
| R10 | Present creation results. | Successful onboarding exposes the agent ID, client ID, and masked creation secret with reveal and copy controls. A fresh run clears prior results. Later agent reads are not used to recover the creation secret. |
| R11 | Support reusable authoring. | The builder offers agent provisioning, type and owner resolution, delegated widgets, and three complete onboarding templates. Agent mode survives reopening. Their prompt order and incomplete/failure routes match this specification. |
| R12 | Collect delegated configuration with reusable selectors. | Delegation follows the stated precedence. Missing delegated inputs are requested only when needed; stale values are omitted when delegation is false. Selectors load successive pages through SDK functions and submit user IDs, flow IDs, and user type handles. The server validates the selections independently. |

## Change log

| Version | Date | Change |
|---|---|---|
| 0.1 | 2026-09-07 | Initial specification. |
| 0.2 | 2026-10-06 | Add on-behalf-of onboarding, delegated inputs, widgets, and templates. |
| 0.3 | 2026-10-23 | Rewrite the feature story around the implemented provider boundary, schema-based name, administration journey, service-owned delegation policy, and shared SDK and Console selectors. Correct earlier prompt, failure-routing, labeling, and scope descriptions; distinguish integrated behavior from remaining delivery limits. |