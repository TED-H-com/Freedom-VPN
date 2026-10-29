/**
 * English (en) locale dictionary — secondary UI language.
 * Keys are strictly checked against `zhTW` at compile time and in unit tests.
 */

import type { TranslationKey } from './zh-TW';

export const en: Record<TranslationKey, string> = {
  // App Shell & Navigation
  'app.title': 'Freedom VPN Admin & Member Portal',
  'app.skipToMain': 'Skip to main content',
  'app.language.label': 'Language',
  'app.language.zhTW': '繁體中文 (zh-TW)',
  'app.language.en': 'English (en)',
  'app.role.admin': 'Admin',
  'app.role.member': 'Member',
  'app.mockSwitcher.label': 'Mock Persona Switcher (TEST ONLY):',
  'app.mockSwitcher.anonymous': 'Anonymous (Unauthenticated)',
  'app.mockSwitcher.adminNoMfa': 'Admin: Without MFA (mfa_verified=false)',
  'app.mockSwitcher.admin': 'Admin',
  'app.mockSwitcher.memberActive': 'Member: Active (Normal Speed)',
  'app.mockSwitcher.memberThrottled': 'Member: Active (FUP Throttled / With Gap)',
  'app.mockSwitcher.memberPending': 'Member: Pending Payment (pending_payment)',
  'app.mockSwitcher.memberCancelEnd':
    'Member: Cancel at Period End (cancel_at_period_end)',
  'app.mockSwitcher.memberGrace': 'Member: Grace Period (past_due / grace)',
  'app.mockSwitcher.memberExpired': 'Member: Expired (expired)',
  'app.mockSwitcher.memberSuspended': 'Member: Admin Suspended (suspended)',
  'app.mockSwitcher.trigger401': 'Simulate Session Expired (401)',
  'app.logout': 'Sign Out',
  'app.footer.securityNote':
    'This portal holds no NetBird management token, release signing key, or payment card data (§18.4, AGENTS.md §2).',

  // Navigation items
  'nav.admin.heading': 'Admin Area (§18.1)',
  'nav.admin.overview': 'Overview',
  'nav.admin.nodes': 'Nodes',
  'nav.admin.addNode': 'Add Node Enrollment',
  'nav.admin.updates': 'Releases & Fleet Update',
  'nav.admin.members': 'Members',
  'nav.admin.fup': 'Plans & Fair Use (FUP)',
  'nav.admin.dns': 'Web & DNS Policy',
  'nav.admin.audit': 'Operations & Audit',
  'nav.member.heading': 'Member Area (§18.4)',
  'nav.member.service': 'My Service',
  'nav.member.gettingStarted': 'Getting Started',
  'nav.member.exits': 'Available Exits',
  'nav.member.devices': 'My Devices',
  'nav.member.usage': 'Usage & Speed',
  'nav.member.billing': 'Billing & Subscription',
  'nav.member.dnsProtection': 'Web Protection Info',

  // Auth & Guards
  'auth.anonymous.title': 'Sign In Required (Unauthenticated)',
  'auth.anonymous.description':
    'Please sign in via single sign-on (OIDC) to access the Admin or Member portal.',
  'auth.anonymous.loginBtn': 'Sign In with OIDC',
  'auth.mfaRequired.title': 'Administrator MFA Required',
  'auth.mfaRequired.description':
    'Your account has the Admin role, but multi-factor authentication has not been verified for this session (mfa_verified: false). Per §16.1, all Admin access requires MFA.',
  'auth.mfaRequired.verifyBtn': 'Complete MFA Verification',
  'auth.sessionExpired.title': 'Session Expired (401)',
  'auth.sessionExpired.description':
    'Your OIDC session has expired or been invalidated. Please sign in again to continue.',
  'auth.sessionExpired.reloginBtn': 'Sign In Again',
  'auth.forbidden.title': 'Access Denied: Insufficient Permissions (403)',
  'auth.forbidden.description':
    'Your current account role is Member and cannot access Admin routes (/admin/*). This client-side route guard is defense in depth only; all backend /api/v1/admin/* endpoints enforce Admin role and MFA (§9.1, §16.1).',
  'auth.forbidden.backToMember': 'Return to Member Area (/member)',
  'errorBoundary.title': 'An unexpected rendering error occurred',
  'errorBoundary.reset': 'Reset View State',

  // Common Operation & Conflict UI (§14.8, §18.3)
  'op.savedPendingApply': 'Saved, waiting to apply',
  'op.acceptedPending':
    'Accepted, waiting for worker and node confirmation (HTTP 202 is not done)',
  'op.completedApplied': 'Applied (confirmed via observed read-back)',
  'op.partiallySucceeded':
    'Partially Succeeded (see step details and lease constraints below)',
  'op.failed': 'Apply Failed',
  'op.routeWithdrawnDeviceUnconfirmed':
    'Central routes withdrawn; device revocation unconfirmed / bounded by lease',
  'op.idLabel': 'Operation ID (operation_id)',
  'op.requestIdLabel': 'Request ID (request_id)',
  'op.generationsLabel':
    'Desired Generation: {desired} / Observed Generation: {observed}',
  'op.step.agent_policy': 'Node policy dispatch & ACK (agent_policy)',
  'op.step.netbird_route_readback':
    'NetBird route config & read-back (netbird_route_readback)',
  'op.step.data_path_probe': 'Synthetic data-path & isolation probe (data_path_probe)',
  'op.step.node_allowset_withdraw':
    'Node member allowset withdrawal (node_allowset_withdraw)',
  'op.step.cert_and_peer_revoke':
    'Machine cert & NetBird peer revocation (cert_and_peer_revoke)',
  'op.step.policy_outbox_sync':
    'Member entitlement & node policy sync (policy_outbox_sync)',
  'op.step.resolver_1_smoke_apply':
    'Resolver #1 syntax check & atomic switch (resolver_1_smoke_apply)',
  'op.step.resolver_2_smoke_apply':
    'Resolver #2 syntax check & atomic switch (resolver_2_smoke_apply)',
  'op.dismiss': 'Dismiss progress panel',

  'conflict.dialog.title':
    'Resource Version Conflict Detected (HTTP 412 Precondition Failed)',
  'conflict.dialog.description':
    'This resource was modified by another admin or background process while you were editing. To prevent silently overwriting their changes, your save was rejected (§14.5, §18.3). Reload the latest resource state before re-applying changes.',
  'conflict.dialog.currentVersion': 'Server current_resource_version: {version}',
  'conflict.dialog.reloadBtn': 'Reload Latest Resource (Do Not Overwrite)',

  // Admin Overview (§18.1)
  'admin.overview.title': 'Fleet Operations Overview',
  'admin.overview.subtitle':
    'Displays exit availability, control connectivity, isolation safety, update progress, and member entitlements separately; never uses a single green dot for overall health (§16.5, §18.1).',
  'admin.overview.asOf': 'Data last updated (UTC): {asOf}',
  'admin.overview.card.availableExits': 'Serviceable Exit Nodes',
  'admin.overview.card.availableExitsDesc':
    'Passing identity, publication intent, provisioning, isolation safety, and supported release checks (§5.1)',
  'admin.overview.card.controlStatus': 'Node Control Connectivity',
  'admin.overview.card.controlBreakdown':
    'Online {online} / Stale {stale} / Offline {offline}',
  'admin.overview.card.safetyFailed': 'Isolation or Safety Check Failed',
  'admin.overview.card.safetyFailedDesc':
    'Control online, but member forwarding disabled fail-closed',
  'admin.overview.card.enrollmentsPending': 'Pending Enrollments or Failed Provisioning',
  'admin.overview.card.enrollmentsPendingDesc':
    'Nodes still enrolling or failing local setup',
  'admin.overview.card.updateAttention': 'Nodes Needing Update Attention',
  'admin.overview.card.updateAttentionDesc':
    'Failed, rolled back, or waiting online to catch up',
  'admin.overview.card.members': 'Active Members',
  'admin.overview.card.membersBreakdown':
    'Active {active} (Throttled {throttled}) / Suspended or Expired {inactive}',
  'admin.overview.link.goNodes': 'View Nodes List',
  'admin.overview.link.goUpdates': 'View Fleet Updates',
  'admin.overview.link.goMembers': 'View Members',

  // Admin Nodes List (§5, §18.1)
  'admin.nodes.title': 'Nodes List',
  'admin.nodes.subtitle':
    'All node sources (pre-installed box, cloud VM, Ubuntu host) share the exact same data model and operations (NODE01, NODE04).',
  'admin.nodes.addBtn': 'Add Node Enrollment',
  'admin.nodes.fleetUpdateBtn': 'Fleet Update All Nodes',
  'admin.nodes.filter.search': 'Search by name or region',
  'admin.nodes.filter.publication': 'Publication Intent (Desired)',
  'admin.nodes.filter.control': 'Control Status',
  'admin.nodes.filter.all': 'All',
  'admin.nodes.col.node': 'Name / Region / Arch',
  'admin.nodes.col.publication': 'Publication Intent vs Observed',
  'admin.nodes.col.controlOnline': '1. Control Online',
  'admin.nodes.col.dataAvailable': '2. Data Path Available',
  'admin.nodes.col.policyValid': '3. Policy & Isolation Valid',
  'admin.nodes.col.updateComplete': '4. Update Complete & Release',
  'admin.nodes.col.summary': 'Service Summary (§18.1)',
  'admin.nodes.col.actions': 'Actions',
  'admin.nodes.viewDetail': 'Manage Detail',

  'node.summary.online_serving': 'Online, actively serving exit traffic',
  'node.summary.online_safety_failed_not_serving':
    'Online, isolation check failed, not serving traffic',
  'node.summary.online_disabled_not_serving':
    'Online, disabled by admin, not serving traffic',
  'node.summary.ready_unpublished_not_serving':
    'Provisioned ready, unpublished, not serving traffic',
  'node.summary.offline_route_withdrawn_unconfirmed':
    'Central routes withdrawn; device revocation unconfirmed / bounded by lease',
  'node.summary.rolled_back_serving':
    'Online, update rolled back to verified last_good release',
  'node.summary.revoked_not_serving': 'Machine identity permanently revoked, not serving',

  // Admin Node Detail (§18.1, §18.3)
  'admin.nodeDetail.back': '← Back to Nodes List',
  'admin.nodeDetail.title': 'Node Detail: {name}',
  'admin.nodeDetail.meta':
    'Node ID: {nodeId} | Incarnation: {incarnationId} | ETag: {etag}',
  'admin.nodeDetail.desiredVsObserved.title':
    'Desired State vs Observed State Comparison',
  'admin.nodeDetail.field.setting': 'Setting',
  'admin.nodeDetail.field.desired': 'Central Desired State',
  'admin.nodeDetail.field.observed': 'Node & Route Observed State',
  'admin.nodeDetail.field.publication': 'Exit Publication State',
  'admin.nodeDetail.field.configGen': 'Config Generation (config_generation)',
  'admin.nodeDetail.field.release': 'Software Release',
  'admin.nodeDetail.field.memberGroups': 'Allowed Member Groups',
  'admin.nodeDetail.field.capacity': 'Node Total Capacity (Up / Down)',
  'admin.nodeDetail.field.safetyAndIp': 'Internet-only Isolation / Observed Exit IP',
  'admin.nodeDetail.lastAckAt': 'Last successful ACK at: {time}',
  'admin.nodeDetail.pubActions.title':
    'Publish, Unpublish / Disable & Revoke Identity (§5.2, §18.3)',
  'admin.nodeDetail.pubActions.publishBtn': 'Publish Exit',
  'admin.nodeDetail.pubActions.disableBtn': 'Disable / Unpublish Exit',
  'admin.nodeDetail.pubActions.revokeSectionTitle':
    'Permanently Revoke Node Machine Identity',
  'admin.nodeDetail.pubActions.revokeDistinctNote':
    'Important: Revoking machine identity is distinct from disabling publication. Revocation permanently invalidates the Agent mTLS certificate, NetBird peer, and routes; rejoining requires a brand-new enrollment token. Historical usage ledger and audit logs are preserved (§5.2, §18.3).',
  'admin.nodeDetail.pubActions.revokeBtn': 'Permanently Revoke Node Identity',
  'admin.nodeDetail.editForm.title':
    'Edit Node Classification & Capacity (Requires If-Match ETag)',
  'admin.nodeDetail.editForm.name': 'Display Name',
  'admin.nodeDetail.editForm.region': 'Region',
  'admin.nodeDetail.editForm.groups': 'Allowed Member Groups (comma-separated)',
  'admin.nodeDetail.editForm.upBps': 'Upstream Capacity Cap (bps decimal string)',
  'admin.nodeDetail.editForm.downBps': 'Downstream Capacity Cap (bps decimal string)',
  'admin.nodeDetail.editForm.notes':
    'Source & Operator Notes (never affects permissions)',
  'admin.nodeDetail.editForm.saveBtn': 'Save Node Settings',
  'admin.nodeDetail.editForm.simulateConflictBtn':
    'Simulate Concurrent Edit Conflict (Trigger 412)',
  'admin.nodeDetail.errors.title': 'Recent Errors & Operation Diagnostics (Expandable)',
  'admin.nodeDetail.errors.empty': 'No recent errors recorded',
  'admin.nodeDetail.history.title': 'Node Update History',

  // Admin Add Node / Enrollment (§18.2)
  'admin.enrollment.title': 'Add Node & One-Time Enrollment Token (§18.2)',
  'admin.enrollment.subtitle':
    'A valid enrollment token authorizes joining; once automatic provisioning and isolation checks pass, an admin classifies and publishes the node without a second manual network-join approval (ENR01, ENR04).',
  'admin.enrollment.form.name': 'Pre-filled Node Name (optional)',
  'admin.enrollment.form.region': 'Pre-filled Region (optional)',
  'admin.enrollment.form.groups': 'Pre-filled Member Groups (comma-separated)',
  'admin.enrollment.form.expiresHours':
    'Token Validity (hours, recommended default: 168h / 7d for pre-installed box)',
  'admin.enrollment.form.notes':
    'Notes (e.g. pre-installed box / OCI VM; informational only)',
  'admin.enrollment.form.submitBtn': 'Create One-Time Enrollment Token',
  'admin.enrollment.oneTime.bannerTitle':
    'One-Time Enrollment Token Created (Shown once only; leaving this view hides plaintext forever)',
  'admin.enrollment.oneTime.securityNote':
    'Only the token hash is stored centrally and this response carries Cache-Control: no-store. The token is never persisted in browser storage, URL parameters, or logs. Copy it securely into your single-node config file; avoid pasting secrets directly into persistent shell history (§18.2).',
  'admin.enrollment.oneTime.copyBtn': 'Copy Enrollment Token Securely',
  'admin.enrollment.oneTime.copied': 'Copied to clipboard',
  'admin.enrollment.oneTime.viewStatusBtn':
    'Simulate Leaving & Re-entering Enrollment View (Verify Token Hidden)',
  'admin.enrollment.detail.title': 'Enrollment Status Record',
  'admin.enrollment.detail.secretGone':
    'Per security contract (§14.2, §14.5, §18.2), the plaintext enrollment token is shown only once upon creation and cannot be retrieved again (secret_available: false). If lost, revoke this enrollment and create a new one.',
  'admin.enrollment.detail.stateLabel': 'Current State: {state}',
  'admin.enrollment.detail.revokeBtn': 'Revoke Unused Enrollment Token',

  // Admin Fleet Update (§13.12)
  'admin.updates.title': 'Releases & Fleet Update All Nodes (§13.12)',
  'admin.updates.subtitle':
    'Pick an approved, signed release and run "Update All Nodes" once to snapshot all non-revoked/non-retired nodes. Desired target release and minimum_serving_version are managed separately (UPD01, UPD04).',
  'admin.updates.currentFleetDefault':
    'Current Fleet Default Release (fleet_default_release_id): {releaseId}',
  'admin.updates.minServingVersion':
    'Minimum Allowed Serving Version (minimum_serving_version): {version}',
  'admin.updates.selectReleaseLabel': 'Select Target Release',
  'admin.updates.restartRequiredYes':
    'Requires Data-Plane Service Restart (requires_service_restart: true) — brief forwarding pause during switch',
  'admin.updates.restartRequiredNo':
    'No Data-Plane Service Restart Required (requires_service_restart: false)',
  'admin.updates.runAllBtn': 'Update All Nodes (scope=all_nodes)',
  'admin.updates.acceptedBanner':
    'Accepted: Update batch created; tracking per-node progress (HTTP 202 does not mean completed)',
  'admin.updates.batchTitle': 'Update Batch Progress: {batchId}',
  'admin.updates.batchStateLabel': 'Batch State: {state}',
  'admin.updates.targetCountLabel':
    'Immutable Snapshot Target Count (target_count): {count}',
  'admin.updates.bucketSumCheck': 'Mutually Exclusive Bucket Sum: {sum} / {targetCount}',
  'admin.updates.verifiedSuccessLabel':
    'Verified Success (succeeded + already_current): {count} (Note: rolled_back, superseded, and cancelled are NEVER counted as success, §13.12)',
  'admin.updates.retryFailedBtn': 'Retry Failed Targets',
  'admin.updates.col.node': 'Node / Region / Arch',
  'admin.updates.col.pubIntent': 'Publication Intent (Preserved)',
  'admin.updates.col.versions': 'Current → Target Release (Last Good)',
  'admin.updates.col.bucket': 'Mutually Exclusive Bucket',
  'admin.updates.col.error': 'Error Code / Last Reported',

  // Admin Members (§9.2, §18.1)
  'admin.members.title': 'Members & Entitlement Management (§9.2, §18.1)',
  'admin.members.subtitle':
    'View effective entitlements, paid periods, finite grants, device quotas, and cross-node decimal usage totals.',
  'admin.members.filter.state': 'Filter by Entitlement State',
  'admin.members.col.member': 'Member / Email',
  'admin.members.col.entitlement': 'Entitlement State',
  'admin.members.col.planAndEnd': 'Plan / Effective Service End',
  'admin.members.col.devices': 'Authorized Devices',
  'admin.members.col.usage': 'Monthly Usage (Decimal GB / GiB)',
  'admin.members.col.fupTier': 'Fair Use Tier (FUP)',
  'admin.members.col.payment': 'Payment & Reconciliation',
  'admin.members.col.actions': 'Actions',
  'admin.members.manageBtn': 'Manage Member',

  'admin.memberDetail.back': '← Back to Members List',
  'admin.memberDetail.title': 'Member Detail: {name}',
  'admin.memberDetail.entitlementBox': 'Entitlement & Deadline Breakdown (§9.2, §15.2)',
  'admin.memberDetail.paidThrough': 'Billing Paid Through (paid_through): {time}',
  'admin.memberDetail.effectiveEnd':
    'Effective Service End (effective_service_end): {time}',
  'admin.memberDetail.reconciledAt': 'Last Provider Reconciliation: {time}',
  'admin.memberDetail.suspendSection':
    'Admin Suspension & Resume (Overrides payment success, §9.2)',
  'admin.memberDetail.suspendReasonLabel': 'Suspension Reason',
  'admin.memberDetail.suspendBtn': 'Suspend Member Immediately',
  'admin.memberDetail.resumeBtn': 'Resume Member Access',
  'admin.memberDetail.grantSection': 'Create Finite Entitlement Grant (§14.2)',
  'admin.memberDetail.grantValidUntil':
    'Grant Valid Until (UTC RFC 3339, required & must be finite)',
  'admin.memberDetail.grantReason':
    'Grant Reason (required, e.g. migration overlap or credit)',
  'admin.memberDetail.grantSubmitBtn': 'Grant Finite Access',
  'admin.memberDetail.devicesSection': 'Member Registered Devices & Revocation (§9.4)',
  'admin.memberDetail.revokeDeviceBtn': 'Revoke Device',

  // Admin FUP & Plans (§11.1, §18.1)
  'admin.fup.title': 'Plans & Fair Use Policy Draft Editor (§11.1)',
  'admin.fup.testOnlyBanner':
    'TEST ONLY (Pending Business Decision): All thresholds, speed caps, and plan parameters below are test placeholders and not production commitments (AGENTS.md §1, §11).',
  'admin.fup.modeLabel': 'Policy Execution Mode (mode)',
  'admin.fup.mode.shadow':
    'Shadow Mode (Computes would_apply_tier and impact only; data plane keeps normal rate)',
  'admin.fup.mode.enforce':
    'Enforce Mode (Applies per-member HTB + fq_codel upload/download shaping on nodes)',
  'admin.fup.effectiveAtLabel': 'Effective At (effective_at, UTC RFC 3339)',
  'admin.fup.minRateLabel':
    'Minimum Usable Rate Floor (minimum_rate_bps, decimal string)',
  'admin.fup.reasonLabel': 'Revision Reason',
  'admin.fup.tiersTitle':
    'Tier Definitions (Must start at 0, strictly increasing thresholds, both directions present and non-negative)',
  'admin.fup.addTierBtn': '+ Add Tier',
  'admin.fup.removeTierBtn': 'Remove',
  'admin.fup.validateAndPublishBtn': 'Validate & Publish Policy Revision',
  'admin.fup.validationPassed':
    'Draft validation passed: strictly increasing thresholds, valid bidirectional rates, complete coverage.',
  'fup.validation.emptyTiers': 'At least one tier (Tier 0) must be defined.',
  'fup.validation.missingZeroStart':
    'Tier 0 threshold_bytes must be "0" so there are no undefined coverage gaps.',
  'fup.validation.invalidThreshold':
    'Threshold must be a valid non-negative integer decimal string.',
  'fup.validation.nonIncreasingThreshold':
    'Tier threshold_bytes must be strictly increasing (T_i > T_{i-1}).',
  'fup.validation.missingDirectionRate':
    'Every tier must specify both download (down_bps) and upload (up_bps) rates.',
  'fup.validation.invalidRateDecimal':
    'Upload and download rates must be valid non-negative integer decimal strings.',
  'fup.validation.rateBelowMinimum':
    'Upload and download rates cannot be below minimum_rate_bps.',
  'fup.validation.invalidMinimumRate':
    'Minimum usable rate must be a decimal integer string greater than 0.',
  'fup.validation.invalidPriority': 'Priority must be an integer between 1 and 10.',
  'fup.validation.emptyTierName': 'Every tier must have both zh-TW and en display names.',
  'fup.validation.invalidEffectiveAt':
    'effective_at must be a valid UTC RFC 3339 timestamp ending in Z.',

  // Admin DNS Policy (§12.4, §12.5)
  'admin.dns.title': 'Web & DNS Protection Policy (§12.4, §12.5)',
  'admin.dns.subtitle':
    'Manage domain-level force block, explicit allow rules, and threat feeds. No DNS allow rule can ever relax fixed Internet-only IP isolation (ISO04, §12.4).',
  'admin.dns.resolversTitle': 'Controlled DNS Resolver Applied Status',
  'admin.dns.publishBtn': 'Compile & Publish Policy Bundle (202)',
  'admin.dns.rollbackBtn': 'Rollback to Last Good Revision (last_good)',
  'admin.dns.addRuleTitle':
    'Add Domain Rule (Automatic IDNA normalization, lowercase, trailing dot stripped)',
  'admin.dns.domainInput': 'Domain Name (no http://, /path, :port, or * wildcards)',
  'admin.dns.actionInput': 'Action',
  'admin.dns.action.block': 'Force Block (Priority 1: Admin Block)',
  'admin.dns.action.allow':
    'Explicit Allow (Priority 2: Admin Allow, overrides threat feed)',
  'admin.dns.includeSubdomains':
    'Include Subdomains (include_subdomains, full label-boundary match)',
  'admin.dns.reasonInput': 'Rule Reason',
  'admin.dns.addRuleSubmit': 'Add Rule to Draft',
  'admin.dns.previewTitle': 'Rule Match & Priority Preview Tool (§12.4 Match Preview)',
  'admin.dns.previewInput':
    'Enter test query domain (e.g. sub.tracker.example.com or nottracker.example.com)',
  'admin.dns.previewOutcome.blocked_admin':
    'Outcome: Blocked (NXDOMAIN) — Matched Priority 1: Admin Force Block',
  'admin.dns.previewOutcome.allowed_admin':
    'Outcome: Allowed — Matched Priority 2: Admin Explicit Allow',
  'admin.dns.previewOutcome.blocked_threat_feed':
    'Outcome: Blocked (NXDOMAIN) — Matched Priority 3: Enabled Threat Feed',
  'admin.dns.previewOutcome.resolved_normal':
    'Outcome: Normal Resolution — Priority 4: Forwarded to upstream resolver',
  'admin.dns.previewIsolationWarning':
    'Highest Priority Reminder: Even if DNS resolves or is allowlisted, any traffic targeting RFC1918, host LAN, or cloud metadata IPs is unconditionally blocked by the IP-layer Internet-only firewall (§8.1, §12.4).',
  'dns.error.EMPTY_DOMAIN': 'Please enter a domain name.',
  'dns.error.URL_OR_PATH_NOT_ALLOWED':
    'Domain name only: URL schemes (https://), ports (:443), paths (/path), and queries are rejected (§12.4).',
  'dns.error.WILDCARD_NOT_ALLOWED':
    'Wildcards (*) are not accepted; use the include_subdomains toggle instead (§12.4).',
  'dns.error.INVALID_LABEL':
    'Invalid DNS label format; must contain at least two valid DNS labels.',

  // Admin Audit (§14.2, §18.1)
  'admin.audit.title': 'Operations & Security Audit Log (§14.2, §18.1)',
  'admin.audit.subtitle':
    'Read-only, sanitized history of configuration and authorization changes (append-only). Per spec, no delete button is provided (§18.1).',
  'admin.audit.filter.resource': 'Filter by Resource Type',
  'admin.audit.filter.search': 'Search Request ID / Actor / Action',
  'admin.audit.col.time': 'Occurred At (UTC)',
  'admin.audit.col.actor': 'Actor / Role',
  'admin.audit.col.action': 'Action / Resource',
  'admin.audit.col.trace': 'Request ID / Operation ID',
  'admin.audit.col.result': 'Result & Sanitized Diff',

  // Member Portal (§18.4)
  'member.service.title': 'My Service (§18.4)',
  'member.service.stateLabel': 'Current Entitlement State',
  'member.service.reasonLabel': 'Status Reason & Details',
  'member.service.planLabel': 'Current Plan',
  'member.service.effectiveEndLabel': 'Effective Service End (effective_service_end)',
  'member.service.paidThroughLabel': 'Billing Paid Through (paid_through)',
  'member.service.nextResetLabel': 'Next Fair-Use (FUP) Reset Time',
  'member.service.fupThrottledBanner':
    'High usage this period: speed has been adjusted under the Fair Use Policy; next reset is {resetAt}',

  'member.gettingStarted.title': 'Getting Started & Per-OS Setup Guide (§7.3, §18.4)',
  'member.gettingStarted.subtitle':
    'Freedom VPN uses the official NetBird client apps with our self-hosted management URL and OIDC SSO. You never need to manually copy WireGuard private keys.',
  'member.gettingStarted.mgmtUrlLabel': 'Self-Hosted Management URL',
  'member.gettingStarted.copyUrlBtn': 'Copy Management URL',
  'member.gettingStarted.osSelectLabel': 'Select Your Operating System',
  'member.gettingStarted.step1':
    '1. Download and install the official NetBird app for your OS.',
  'member.gettingStarted.step2':
    '2. Open the app preferences and set the Management URL to the self-hosted URL above.',
  'member.gettingStarted.step3':
    '3. Sign in with SSO using the same account as this portal to register and authorize your device.',
  'member.gettingStarted.step4':
    '4. Open the Exit Node selector in the app and manually select your desired exit region.',
  'member.gettingStarted.killSwitchWarningTitle':
    'Important Limitation: No Complete Always-On / Kill-Switch Mode (§7.3)',
  'member.gettingStarted.killSwitchWarningBody':
    'The official NetBird client does not provide a complete OS-level always-on / kill-switch mode. Server-side isolation protects traffic inside the VPN tunnel, but if the VPN tunnel disconnects or switches networks, your device may send traffic directly over your local ISP or Wi-Fi.',
  'member.gettingStarted.androidPrivateDnsWarning':
    'Android Specific Note (§12.6): Android "Private DNS" bypasses the VPN DNS configuration. On Android, go to "Settings → Network & internet → Private DNS" and set it to "Off" so domain-level malware/phishing protection works properly.',

  'member.exits.title': 'Available Exit Nodes (§18.4, §18.5)',
  'member.exits.manualSelectionNotice':
    'Important: Exit selection must be performed manually inside the NetBird app on your device. This page shows exits available to your plan and their health, but does not remotely switch your app; list order is not an automatic fastest-node ranking (EXIT02, §18.4).',
  'member.exits.col.name': 'Exit Name',
  'member.exits.col.region': 'Region',
  'member.exits.col.status': 'Availability Status',
  'member.exits.col.notes': 'Notes',

  'member.devices.title': 'My Devices (§9.4, §18.4)',
  'member.devices.quotaSummary': 'Authorized Device Quota: {used} / {limit} devices',
  'member.devices.quotaHelp':
    'If you reach your device limit, you can revoke an old or lost device here and then connect your new device once synchronization completes (§7.3, AT02). You can only view and revoke your own devices.',
  'member.devices.col.name': 'Device Name / Platform',
  'member.devices.col.overlayIp': 'Overlay IP / Peer ID',
  'member.devices.col.status': 'Authorization Status',
  'member.devices.col.lastSeen': 'Last Activity / Authorized At',
  'member.devices.col.action': 'Action',
  'member.devices.revokeBtn': 'Revoke This Device',

  'member.usage.title': 'Usage & Speed (§10.1, §11.6, §18.4)',
  'member.usage.honestyBanner':
    'Honest Metering Notice: This view shows "Recorded VPN Usage (forward_admission_v1)" and is NOT a live real-time speedometer. Counters update as nodes batch-upload usage events and central rollups advance (§10.1, §18.4).',
  'member.usage.asOfLabel': 'Aggregate As Of (aggregate_as_of): {asOf}',
  'member.usage.periodLabel': 'Accounting Period (UTC Calendar Month): {period}',
  'member.usage.totalLabel': 'Monthly Recorded Total (Upload + Download)',
  'member.usage.uploadLabel': 'Monthly Recorded Upload (upload_bytes)',
  'member.usage.downloadLabel': 'Monthly Recorded Download (download_bytes)',
  'member.usage.speedStatusLabel': 'Current Speed Status & Fair-Use Tier',
  'member.usage.rateCapsLabel':
    'Current Per-Exit Combined Rate Cap: Down {down} / Up {up}',
  'member.usage.nextResetLabel': 'Next Reset Time: {resetAt}',
  'member.usage.gapWarningTitle': 'Metering Data Quality Notice ({quality})',
  'member.usage.gapWarningBody':
    'This accounting period includes a metering gap or boundary estimate: {note}. Only verified events are summed; missing intervals are never hidden with fabricated numbers (§10.7, §18.4).',

  'member.billing.title': 'Billing & Subscription (§9.3, §18.4, AT03)',
  'member.billing.noCardNotice':
    'Security Guarantee: Freedom VPN never collects payment card numbers in portal forms. All checkout and subscription changes happen exclusively on the payment provider’s hosted pages (§18.4).',
  'member.billing.checkoutBtn': 'Proceed to Hosted Provider Checkout',
  'member.billing.portalBtn': 'Open Hosted Billing Management Portal',
  'member.billing.simulateReturnBtn':
    'Simulate Provider Return Redirect (?checkout=return, AT03)',
  'member.billing.returnPendingBadge': 'Verifying (確認中)',
  'member.billing.returnPendingBody':
    'You have returned from the payment provider checkout page; status is currently "Verifying (確認中)". Per security contract (BILL03, AT03, §18.4), the browser return redirect never activates entitlement on its own; access updates only after the signed provider webhook or reconciliation is verified.',

  'member.dns.title': 'Web Protection Information (§12.1, §12.6, §12.8, §18.4)',
  'member.dns.scopeTitle': 'v1 Protection Scope: DNS Domain-Level Filtering Only',
  'member.dns.scopeBody':
    'Freedom VPN blocks known malware, phishing, and admin-blocked domains at the DNS resolution layer using controlled resolvers. We never perform TLS man-in-the-middle (MITM) decryption, and we do not inspect HTTPS page contents, downloaded files, or URL paths (/path).',
  'member.dns.limitationsTitle':
    'Endpoint Limitations: Android Private DNS & DoH (DNS over HTTPS)',
  'member.dns.limitationsBody':
    'If Android "Private DNS" is enabled on your phone, or if your browser/application uses encrypted DNS (DoH / DoT) or connects to raw IP addresses directly, DNS queries bypass our controlled VPN resolvers and domain filtering will not apply (§12.6). Additionally, OS and browser DNS caches or existing HTTPS connections may delay rule changes until caches expire.',
  'member.dns.falsePositiveTitle': 'Reporting False Positives',
  'member.dns.falsePositiveBody':
    'If a legitimate domain is mistakenly blocked, please send the domain name (never send full URLs with private paths) to our operator team at support@example.com for allowlist review.',
};
