import { expect, test, type APIRequestContext, type Page } from '@playwright/test';

const BASE_URL = process.env.PORTAL_E2E_BASE_URL || 'http://127.0.0.1:4173';

interface OperationPollPayload {
  operation_id: string;
  kind: string;
  status: string;
  error?: { code?: string; message?: string } | null;
  steps?: Array<{
    name: string;
    status: string;
    error_code?: string | null;
    message?: string | null;
    observed_at?: string | null;
  }>;
}

interface OperationPollResult {
  finalOp: OperationPollPayload;
  polledStatuses: string[];
}

async function waitForOperationHeadline(
  page: Page,
  request: APIRequestContext,
  scope: 'admin' | 'member',
  expectedHeadline: string,
  expectedStatus = 'succeeded',
): Promise<OperationPollResult> {
  const banner = page.getByTestId('operation-banner');
  await expect(banner).toBeVisible();
  const opId = await banner.getAttribute('data-operation-id');
  expect(opId).toBeTruthy();

  const endpoint =
    scope === 'member'
      ? `${BASE_URL}/api/v1/member/operations/${opId}`
      : `${BASE_URL}/api/v1/admin/operations/${opId}`;
  const deadline = Date.now() + 14_000;
  let lastOp: OperationPollPayload | null = null;
  const polledStatuses: string[] = [];

  const initialDomStatus = await banner.getAttribute('data-operation-status');
  if (initialDomStatus) {
    polledStatuses.push(initialDomStatus);
    if (expectedStatus === 'succeeded' && initialDomStatus === 'failed') {
      const failRes = await request.get(endpoint);
      const failBody = failRes.ok() ? await failRes.json() : null;
      throw new Error(
        `Operation ${opId} observed failed on initial banner render: ${JSON.stringify(failBody)}`,
      );
    }
  }

  while (Date.now() < deadline) {
    const res = await request.get(endpoint);
    if (res.ok()) {
      lastOp = (await res.json()) as OperationPollPayload;
      polledStatuses.push(lastOp.status);
      const domStatus = await banner.getAttribute('data-operation-status');
      if (domStatus) {
        polledStatuses.push(domStatus);
      }
      const hasFailedStep = (lastOp.steps ?? []).some((s) => s.status === 'failed');
      if (
        expectedStatus === 'succeeded' &&
        (lastOp.status === 'failed' || domStatus === 'failed' || hasFailedStep)
      ) {
        throw new Error(
          `Operation ${opId} (${lastOp.kind}) was observed failed during polling (polledStatuses=${JSON.stringify(polledStatuses)}): ${JSON.stringify(lastOp)}`,
        );
      }
      if (lastOp.status === expectedStatus) {
        break;
      }
    }
    await page.waitForTimeout(50);
  }

  if (!lastOp || lastOp.status !== expectedStatus) {
    throw new Error(
      `Operation ${opId} (${lastOp?.kind ?? 'unknown'}) did not reach ${expectedStatus}; status=${lastOp?.status ?? 'null'} error_code=${lastOp?.error?.code ?? 'none'} error_message=${lastOp?.error?.message ?? 'none'} steps=${JSON.stringify(lastOp?.steps ?? [])}`,
    );
  }

  await expect(page.getByTestId('operation-status-headline')).toHaveText(
    expectedHeadline,
  );
  return { finalOp: lastOp, polledStatuses };
}

test.describe('Freedom VPN Portal Live Control Plane E2E Suite (§18, AT01–AT04, AT06, AT23–AT24)', () => {
  test('TestAT01_AT03_MemberJourney_SignInPendingPaymentCheckoutReturnAndWebhookActivation', async ({
    page,
    request,
  }) => {
    // 1. Provision a fresh member via real OIDC login in `pending_payment` state
    const resetRes = await request.post(`${BASE_URL}/__e2e/reset-member-journey`);
    expect(resetRes.ok()).toBeTruthy();
    const { member_id: journeyMemberId } = (await resetRes.json()) as {
      member_id: string;
    };
    expect(journeyMemberId).toBeTruthy();

    // 2. Navigate to /member/service and verify pending_payment state (no VPN access granted)
    await page.goto('/member/service');
    await expect(page.getByTestId('member-service-page')).toBeVisible();
    await expect(page.getByTestId('member-entitlement-state-badge')).toHaveText(
      'pending_payment',
    );
    await expect(page.getByTestId('member-state-reason')).toContainText('Webhook');

    // Verify no exits are available while pending_payment
    await page.goto('/member/exits');
    await expect(page.getByTestId('member-exits-page')).toBeVisible();
    await expect(page.locator('[data-testid^="member-exit-row-"]')).toHaveCount(0);

    // 3. Navigate to /member/billing and start synchronous checkout hand-off (201 checkout_url, D-031)
    await page.goto('/member/billing');
    await expect(page.getByTestId('member-billing-page')).toBeVisible();
    await expect(page.getByTestId('no-card-collection-notice')).toBeVisible();
    await expect(page.getByTestId('billing-entitlement-state')).toHaveText(
      'pending_payment',
    );

    await page.getByTestId('start-hosted-checkout-btn').click();
    await expect(page.getByTestId('hosted-redirect-banner')).toBeVisible();
    await expect(page.getByTestId('hosted-redirect-url')).toContainText(
      'https://checkout.stripe.example.invalid/pay/cs_test_',
    );

    // 4. Simulate browser return from hosted checkout BEFORE webhook confirmation (AT03)
    await page.goto('/member/billing?checkout=return');
    await expect(page.getByTestId('billing-return-pending-badge')).toBeVisible();
    await expect(page.getByTestId('billing-return-pending-badge')).toContainText(
      '確認中',
    );
    await expect(page.getByTestId('checkout-return-pending-banner')).toBeVisible();
    // Entitlement MUST remain pending_payment until signed webhook is verified (AT03)
    await expect(page.getByTestId('billing-entitlement-state')).toHaveText(
      'pending_payment',
    );

    // 5. Deliver signed Stripe webhook (`POST /api/v1/webhooks/stripe`) and let `fvpn-worker` reconcile (AT01)
    const whRes = await request.post(`${BASE_URL}/__e2e/stripe-webhook`, {
      data: {},
    });
    expect(whRes.status()).toBe(202);

    // 6. Reload /member/service and /member/exits to verify entitlement is now active with available exit
    await page.goto('/member/service');
    await expect(page.getByTestId('member-entitlement-state-badge')).toHaveText('active');

    await page.goto('/member/exits');
    await expect(page.getByTestId('member-exits-page')).toBeVisible();
    await expect(page.locator('[data-testid^="member-exit-row-"]').first()).toBeVisible();
    await expect(page.locator('[data-testid^="member-exit-row-"]')).not.toHaveCount(0);
    await expect(page.getByTestId('member-exits-table')).toContainText('tw-tpe');

    // 7. Verify billing portal session hand-off on /member/billing
    await page.goto('/member/billing');
    await expect(page.getByTestId('billing-entitlement-state')).toHaveText('active');
    await page.getByTestId('open-billing-portal-btn').click();
    await expect(page.getByTestId('hosted-redirect-banner')).toBeVisible();
    await expect(page.getByTestId('hosted-redirect-url')).toContainText(
      'https://billing.stripe.example.invalid/p/session/bps_test_',
    );
  });

  test('TestAT02_AT04_MemberDevicesLimitRevokeAndAdminSuspensionGrace', async ({
    page,
    request,
  }) => {
    try {
      // 1. Seed 3 authorized devices on the active journey member so device count is 3 / 3 (AT02)
      const seedRes = await request.post(`${BASE_URL}/__e2e/seed-member-devices`);
      expect(seedRes.ok()).toBeTruthy();
      const { device_ids: seededDeviceIds } = (await seedRes.json()) as {
        device_ids: string[];
      };
      expect(seededDeviceIds).toHaveLength(3);

      const meRes = await request.get(`${BASE_URL}/api/v1/member/me`);
      const meJson = (await meRes.json()) as { member_id: string };
      const activeMemberId = meJson.member_id;

      // 2. Navigate to /member/devices and verify 3 / 3 quota, limit warning banner, and Device ID column label (DOC12)
      await page.goto('/member/devices');
      await expect(page.getByTestId('member-devices-page')).toBeVisible();
      await expect(page.getByTestId('member-devices-table')).toContainText(
        '裝置識別碼 (Device ID)',
      );
      await expect(page.getByTestId('device-quota-badge')).toContainText('3 / 3');
      await expect(page.getByTestId('device-limit-warning')).toBeVisible();

      // 3. Revoke the first device (202 Accepted operation -> succeeded, never observed failed at concurrency 4) and verify count drops to 2 / 3 (AT02)
      const firstDeviceId = seededDeviceIds[0];
      await page.getByTestId(`revoke-own-device-${firstDeviceId}`).click();

      const revokePoll = await waitForOperationHeadline(
        page,
        request,
        'member',
        '已套用（已取得實際觀測確認）',
      );
      expect(revokePoll.polledStatuses.length).toBeGreaterThan(0);
      expect(revokePoll.polledStatuses).not.toContain('failed');
      expect(revokePoll.finalOp.status).toBe('succeeded');
      expect(revokePoll.finalOp.steps?.map((s) => s.status)).not.toContain('failed');
      await expect(page.getByTestId('device-quota-badge')).toContainText('2 / 3');
      await expect(page.getByTestId(`device-status-${firstDeviceId}`)).toHaveText(
        'revoked',
      );

      // 4. Switch to Admin view (/admin/members/{id}) and suspend the member with If-Match (AT04; never observed failed at concurrency 4)
      await page.goto(`/admin/members/${activeMemberId}`);
      await expect(page.getByTestId('admin-member-detail-page')).toBeVisible();
      await page
        .locator('#suspend-reason-input')
        .fill('AT04 E2E compliance suspension check');
      await page.getByTestId('suspend-member-btn').click();

      const suspendPoll = await waitForOperationHeadline(
        page,
        request,
        'admin',
        '已套用（已取得實際觀測確認）',
      );
      expect(suspendPoll.polledStatuses.length).toBeGreaterThan(0);
      expect(suspendPoll.polledStatuses).not.toContain('failed');
      expect(suspendPoll.finalOp.status).toBe('succeeded');
      expect(suspendPoll.finalOp.steps?.map((s) => s.status)).not.toContain('failed');
      await expect(page.getByTestId('member-detail-entitlement-badge')).toHaveText(
        'suspended',
      );

      // 5. Issue a finite entitlement grant on the member and verify 202 operation banner and grant row
      await page.getByTestId('create-grant-btn').click();
      await expect(page.getByTestId('operation-banner')).toBeVisible();
      await expect(page.getByTestId('operation-status-headline')).toBeVisible();
      const grantRows = page.locator('[data-testid^="admin-grant-row-"]');
      await expect(grantRows.first()).toBeVisible();
      await expect(grantRows).not.toHaveCount(0);
      await expect(grantRows.first()).toContainText(
        'Migration parallel transition grant',
      );

      // 6. Restore default seeded active member and verify /member/usage decimal precision (> 2^53) & watermark
      const restoreRes = await request.post(`${BASE_URL}/__e2e/restore-default-member`);
      expect(restoreRes.ok()).toBeTruthy();

      await page.goto('/member/usage');
      await expect(page.getByTestId('member-usage-page')).toBeVisible();
      await expect(page.getByTestId('usage-total-card')).toContainText(
        'raw: 18014398509481988 B',
      );
      await expect(page.getByTestId('usage-as-of-badge')).toContainText(/Z\)/);
      await expect(page.getByTestId('honest-metering-banner')).toBeVisible();
      await expect(page.getByTestId('usage-gap-warning-banner')).toBeVisible();
    } finally {
      const restoreRes = await request.post(`${BASE_URL}/__e2e/restore-default-member`);
      expect(restoreRes.ok()).toBeTruthy();
    }
  });

  test('TestOperator_NodePatchAnd412PreconditionConflict', async ({ page, request }) => {
    const holdRes = await request.post(`${BASE_URL}/__e2e/hold-node-observations`, {
      data: { hold: true },
    });
    expect(holdRes.ok()).toBeTruthy();

    try {
      // 1. Navigate to /admin/nodes and open the primary Taipei node
      await page.goto('/admin/nodes');
      await expect(page.getByTestId('admin-nodes-list-page')).toBeVisible();

      const nodesRes = await request.get(`${BASE_URL}/api/v1/admin/nodes?role=admin`);
      const nodesData = (await nodesRes.json()) as {
        items: Array<{ node_id: string; name: string }>;
      };
      const tpeNode =
        nodesData.items.find((n) => n.name === 'tw-tpe-01') || nodesData.items[0];
      expect(tpeNode).toBeTruthy();

      await page.goto(`/admin/nodes/${tpeNode.node_id}`);
      await expect(page.getByTestId('admin-node-detail-page')).toBeVisible();

      // 2. Verify desired vs observed state separation
      await expect(page.getByTestId('desired-vs-observed-section')).toBeVisible();
      await expect(page.getByTestId('desired-publication')).toBeVisible();
      await expect(page.getByTestId('observed-publication')).toBeVisible();
      await expect(page.getByTestId('desired-config-gen')).toBeVisible();
      await expect(page.getByTestId('observed-config-gen')).toBeVisible();

      // 3. Perform a valid PATCH with If-Match
      await expect(page.locator('#edit-node-name')).toHaveValue('tw-tpe-01');
      await page.locator('#edit-node-notes').fill('Updated note via E2E step 1');
      await page.getByTestId('save-node-patch-btn').click();
      await expect(page.getByTestId('saved-pending-apply-banner')).toBeVisible();

      // 4. Bump node resource_version concurrently on the backend while browser holds stale ETag
      const bumpRes = await request.post(`${BASE_URL}/__e2e/bump-node-version`, {
        data: { node_id: tpeNode.node_id },
      });
      expect(bumpRes.ok()).toBeTruthy();

      // 5. Submit PATCH with stale ETag -> expect 412 RESOURCE_VERSION_CONFLICT and ConflictDialog
      await page.locator('#edit-node-notes').fill('Stale write attempt');
      await page.getByTestId('save-node-patch-btn').click();

      await expect(page.getByTestId('conflict-dialog')).toBeVisible();

      // 6. Click reload in ConflictDialog -> verify latest version & concurrent notes load cleanly
      await page.getByTestId('conflict-reload-btn').click();
      await expect(page.getByTestId('conflict-dialog')).toHaveCount(0);
      await expect(page.locator('#edit-node-notes')).toHaveValue(
        'Concurrent operator edit',
      );
    } finally {
      await request.post(`${BASE_URL}/__e2e/hold-node-observations`, {
        data: { hold: false },
      });
    }
  });

  test('TestAT06_AT07_EnrollmentTokenSingleDisplayNoStorageLeakAndRevoke', async ({
    page,
  }) => {
    // 1. Navigate to /admin/nodes/new and issue a one-time enrollment token
    await page.goto('/admin/nodes/new');
    await expect(page.getByTestId('admin-add-node-page')).toBeVisible();

    await page.getByTestId('enr-delivery-select').selectOption('preinstalled');
    await page.locator('#enr-name').fill('tw-khh-01');
    await page.locator('#enr-region').fill('tw-khh');
    await page.locator('#enr-notes').fill('Kaohsiung preinstalled appliance');
    await page.getByTestId('create-enrollment-submit-btn').click();

    // 2. Verify token is displayed once in the creation response banner
    await expect(page.getByTestId('one-time-enrollment-banner')).toBeVisible();
    await expect(page.getByTestId('one-time-token-value')).toBeVisible();
    await expect(page.getByTestId('one-time-token-value')).toHaveText(/\S{17,}/);
    const rawSecretToken = (
      await page.getByTestId('one-time-token-value').textContent()
    )?.trim();
    expect(rawSecretToken).toBeTruthy();
    expect(rawSecretToken!.length).toBeGreaterThan(16);

    // 3. Verify token is NEVER persisted in localStorage, sessionStorage, cookies, or URL (SEC01, ENR02)
    const storageDump = await page.evaluate(() => ({
      local: JSON.stringify(window.localStorage),
      session: JSON.stringify(window.sessionStorage),
      cookie: document.cookie,
      href: window.location.href,
    }));
    expect(storageDump.local).not.toContain(rawSecretToken!);
    expect(storageDump.session).not.toContain(rawSecretToken!);
    expect(storageDump.cookie).not.toContain(rawSecretToken!);
    expect(storageDump.href).not.toContain(rawSecretToken!);

    // 4. Navigate to /admin/enrollments/{id} and verify secret_available=false (never re-displayed)
    await page.getByTestId('navigate-enrollment-status-link').click();
    await expect(page).toHaveURL(/\/admin\/enrollments\//);
    await expect(page.getByTestId('admin-enrollment-detail-page')).toBeVisible();
    await expect(page.getByTestId('enrollment-secret-gone-notice')).toBeVisible();
    await expect(page.getByTestId('secret-available-flag')).toHaveText('false');
    await expect(page.getByTestId('one-time-enrollment-banner')).toHaveCount(0);
    await expect(page.locator('body')).not.toContainText(rawSecretToken!);

    // 5. Revoke the unused enrollment token
    await page.getByTestId('revoke-enrollment-btn').click();
    await expect(page.getByTestId('enrollment-state-badge')).toContainText('revoked');
  });

  test('TestAT23_AT24_UpdateBatchProgressNeedsAttentionAndUnconfirmed', async ({
    page,
    request,
  }) => {
    const nodesRes = await request.get(`${BASE_URL}/api/v1/admin/nodes`);
    const nodesJson = (await nodesRes.json()) as {
      items: Array<{
        node_id: string;
        name?: string | null;
        update?: { batch_id?: string | null } | null;
      }>;
    };
    const enrolledNode =
      nodesJson.items.find((n) => n.name === 'tw-tpe-01-updated') ??
      nodesJson.items.find((n) => n.name === 'tw-tpe-01') ??
      nodesJson.items[0];
    const secondaryNode =
      nodesJson.items.find((n) => n.name === 'jp-tyo-01') ??
      nodesJson.items.find((n) => n.node_id !== enrolledNode!.node_id);
    const rollbackNode = nodesJson.items.find((n) => n.name === 'tw-tpe-02');
    expect(secondaryNode).toBeTruthy();
    expect(rollbackNode).toBeTruthy();
    const enrolledNodeId = enrolledNode!.node_id;
    const secondaryNodeId = secondaryNode!.node_id;
    const rollbackNodeId = rollbackNode!.node_id;
    const activeBatchId = enrolledNode?.update?.batch_id;
    expect(activeBatchId).toBeTruthy();

    // 1. Navigate to /admin/updates (auto-discovers active batch from real update-batch API)
    await page.goto(`/admin/updates?batch_id=${activeBatchId}`);
    await expect(page.getByTestId('admin-fleet-updates-page')).toBeVisible();
    await expect(page.getByTestId('update-batch-progress-card')).toBeVisible();

    // 2. Verify 12-bucket target accounting and verified success count from the real batch
    await expect(page.getByTestId('bucket-sum-check')).toContainText('3 / 3');
    await expect(page.getByTestId('verified-success-summary')).toContainText('1');
    await expect(page.getByTestId('bucket-succeeded')).toContainText('1');
    await expect(page.getByTestId('bucket-rolled_back')).toContainText('1');
    await expect(page.getByTestId('bucket-waiting_online')).toContainText('1');

    // 3. Verify per-target `unconfirmed` and D-048 `policy_generation` / `policy_refs` evidence on target rows
    await expect(page.getByTestId(`target-unconfirmed-${secondaryNodeId}`)).toBeVisible();
    await expect(page.getByTestId(`target-policy-gen-${enrolledNodeId}`)).toContainText(
      /[1-9][0-9]*/,
    );
    await expect(page.getByTestId(`target-policy-refs-${enrolledNodeId}`)).toContainText(
      '0195f100-0000-7000-8000-000000000401',
    );
    await expect(page.getByTestId(`target-policy-gen-${secondaryNodeId}`)).toContainText(
      '尚未回報 (not yet reported)',
    );
    await expect(page.getByTestId(`target-policy-refs-${secondaryNodeId}`)).toContainText(
      '尚未回報 (not yet reported)',
    );

    // 4. Verify Target Detail panel (D-048, D-062, API 1.3.0) for confirmed live target, rolled-back target, and offline target
    await page.getByTestId(`inspect-target-${enrolledNodeId}`).click();
    await expect(page.getByTestId('batch-target-detail-panel')).toBeVisible();
    await expect(page.getByTestId('target-detail-node-id')).toContainText(enrolledNodeId);
    await expect(page.getByTestId('target-detail-policy-gen')).toContainText(
      /[1-9][0-9]*/,
    );
    await expect(page.getByTestId('target-detail-policy-refs')).toContainText(
      '0195f100-0000-7000-8000-000000000401',
    );

    // Inspect rolled-back target (modeled on internal/fleetupdate/testdata/at26-direct-rollback.json)
    await page.getByTestId(`inspect-target-${rollbackNodeId}`).click();
    await expect(page.getByTestId('target-detail-node-id')).toContainText(rollbackNodeId);
    await expect(page.getByTestId('target-detail-state')).toHaveText('rolled_back');
    await expect(page.getByTestId('target-detail-rollback-health-section')).toBeVisible();
    await expect(page.getByTestId('target-detail-restored-release-id')).toHaveText(
      'cdb25ffa-5395-400c-ad3e-b32899f14bb8',
    );
    await expect(page.getByTestId('target-detail-previous-release-id')).toHaveText(
      '6d2797d0-b9a5-4724-96af-9b3039d351b1',
    );
    for (const checkKey of [
      'processes',
      'identity_and_state',
      'internet_only',
      'dns_policy',
      'tunnel_probe',
    ]) {
      await expect(page.getByTestId(`restored-health-check-${checkKey}`)).toContainText(
        'passed',
      );
    }
    for (const checkKey of [
      'processes',
      'identity_and_state',
      'internet_only',
      'dns_policy',
    ]) {
      await expect(page.getByTestId(`candidate-health-check-${checkKey}`)).toContainText(
        'passed',
      );
    }
    await expect(page.getByTestId('candidate-health-check-tunnel_probe')).toContainText(
      'failed',
    );

    // Inspect target with no current receipt (waiting_online) -> explicit "no evidence" state rather than blanks
    await page.getByTestId(`inspect-target-${secondaryNodeId}`).click();
    await expect(page.getByTestId('target-detail-node-id')).toContainText(
      secondaryNodeId,
    );
    await expect(page.getByTestId('target-detail-policy-gen')).toContainText(
      '尚未回報 (not yet reported)',
    );
    await expect(page.getByTestId('target-detail-policy-refs')).toContainText(
      '尚未回報 (not yet reported)',
    );
    await expect(page.getByTestId('target-detail-no-receipt-evidence')).toBeVisible();
    await expect(page.getByTestId('target-detail-previous-release-id')).toContainText(
      '尚未回報 (not yet reported)',
    );
    await expect(page.getByTestId('target-detail-restored-health-null')).toContainText(
      '尚未回報 (not yet reported)',
    );
    await expect(page.getByTestId('target-detail-candidate-health-null')).toContainText(
      '尚未回報 (not yet reported)',
    );

    // 5. Verify locale switching (zh-TW <-> en) on target rows and target detail panel
    await page.getByTestId('locale-switcher').selectOption('en');
    await expect(page.getByTestId(`target-unconfirmed-${secondaryNodeId}`)).toContainText(
      'Unconfirmed',
    );
    await expect(page.getByTestId(`target-policy-gen-${secondaryNodeId}`)).toContainText(
      'Not yet reported',
    );
    await expect(page.getByTestId('target-detail-policy-gen')).toContainText(
      'Not yet reported',
    );
    await expect(page.getByTestId('target-detail-policy-refs')).toContainText(
      'Not yet reported',
    );
    await expect(page.getByTestId('target-detail-no-receipt-evidence')).toContainText(
      'No current update receipt evidence reported',
    );
    await expect(page.getByTestId('target-detail-previous-release-id')).toContainText(
      'Not yet reported',
    );
    await expect(page.getByTestId('target-detail-restored-health-null')).toContainText(
      'Not yet reported',
    );
    await expect(page.getByTestId('target-detail-candidate-health-null')).toContainText(
      'Not yet reported',
    );
    await page.getByTestId('locale-switcher').selectOption('zh-TW');

    // 6. Navigate to the node detail view and verify update-target policy evidence (D-048)
    await page.goto(`/admin/nodes/${enrolledNodeId}`);
    await expect(page.getByTestId('node-update-target-evidence-section')).toBeVisible();
    await expect(page.getByTestId('node-detail-target-policy-gen')).toContainText(
      /[1-9][0-9]*/,
    );
    await expect(page.getByTestId('node-detail-target-policy-refs')).toContainText(
      '0195f100-0000-7000-8000-000000000401',
    );
  });

  test('TestD057_D058_GroupsAdminNodeAttachmentCheckDetailAndThreatFeedStatus', async ({
    page,
    request,
  }) => {
    // 1. Member groups admin (/admin/groups, D-057) in zh-TW and en
    await page.goto('/admin/groups');
    await expect(page.getByTestId('admin-groups-page')).toBeVisible();
    await expect(page.getByTestId('groups-table')).toContainText('standard-exits');
    await expect(page.getByTestId('groups-table')).toContainText('default-members');

    // 422 validation detail on invalid group code and whitespace name
    await page.locator('#group-code-input').fill('INVALID_UPPER');
    await page.locator('#group-name-input').fill('Invalid Group');
    await page.getByTestId('group-create-submit-btn').click();
    await expect(page.getByTestId('group-validation-detail')).toBeVisible();
    await expect(page.getByTestId('group-code-field-error')).toBeVisible();

    await page.locator('#group-code-input').fill('valid-code');
    await page.locator('#group-name-input').fill('   ');
    await page.getByTestId('group-create-submit-btn').click();
    await expect(page.getByTestId('group-validation-detail')).toContainText('/name');
    await expect(page.getByTestId('group-name-field-error')).toBeVisible();

    // 409 duplicate code/name conflict surfaced as field error in zh-TW and en
    await page.locator('#group-code-input').fill('standard-exits');
    await page.locator('#group-name-input').fill('Duplicate Exit Group');
    await page.getByTestId('group-create-submit-btn').click();
    await expect(page.getByTestId('group-conflict-field-error')).toBeVisible();
    await expect(page.getByTestId('group-conflict-field-error')).toContainText(
      '群組代碼或名稱已存在',
    );
    await page.getByTestId('locale-switcher').selectOption('en');
    await expect(page.getByTestId('group-conflict-field-error')).toContainText(
      'Duplicate group code or name (409 STATE_CONFLICT)',
    );
    await page.getByTestId('locale-switcher').selectOption('zh-TW');

    // Create a new member group via POST /api/v1/admin/groups
    await page.locator('#group-code-input').fill('asia-premium');
    await page.locator('#group-name-input').fill('Asia Premium Exits');
    await page.getByTestId('group-create-submit-btn').click();
    await expect(page.getByTestId('group-row-asia-premium')).toContainText(
      'Asia Premium Exits',
    );

    // Fetch groups and nodes for cross-checking IDs
    const groupsRes = await request.get(`${BASE_URL}/api/v1/admin/groups`);
    const groupsJson = (await groupsRes.json()) as {
      items: Array<{ group_id: string; code: string; name: string }>;
    };
    const standardExitsGroup = groupsJson.items.find((g) => g.code === 'standard-exits');
    const asiaPremiumGroup = groupsJson.items.find((g) => g.code === 'asia-premium');
    expect(standardExitsGroup).toBeTruthy();
    expect(asiaPremiumGroup).toBeTruthy();

    const nodesRes = await request.get(`${BASE_URL}/api/v1/admin/nodes`);
    const nodesJson = (await nodesRes.json()) as {
      items: Array<{ node_id: string; name?: string | null }>;
    };
    const tpeNode =
      nodesJson.items.find((n) => n.name === 'tw-tpe-01') ?? nodesJson.items[0];
    const tyoNode = nodesJson.items.find((n) => n.name === 'jp-tyo-01');
    expect(tpeNode).toBeTruthy();
    expect(tyoNode).toBeTruthy();

    // 2. Node detail attached groups, PUT /publication with group_ids + 412 conflict, and NodeCheck.detail (D-057, D-058)
    await page.goto(`/admin/nodes/${tpeNode.node_id}`);
    await expect(page.getByTestId('admin-node-detail-page')).toBeVisible();
    await expect(page.getByTestId('node-attached-groups')).toContainText(
      'standard-exits (Standard Exit Group)',
    );

    // Verify NodeCheck.detail (D-058) and DnsPolicyEvidence (D-056) in zh-TW and en
    await expect(page.getByTestId('node-checks-table')).toBeVisible();
    await expect(page.getByTestId('node-check-detail-internet_only')).toContainText(
      'inet fvpn table verified; RFC1918 egress blocked',
    );
    await expect(page.getByTestId('node-check-detail-meter_path')).toContainText(
      'forward_admission_v1 counters active on wg-nb0',
    );
    await expect(page.getByTestId('node-check-detail-tunnel_probe')).toContainText(
      'NetBird WireGuard peer handshake verified',
    );
    await expect(page.getByTestId('node-dns-policy-evidence-section')).toBeVisible();
    await expect(page.getByTestId('node-dns-evidence-disclaimer')).toContainText(
      '絕非中央驗證已套用之證明',
    );
    await expect(page.getByTestId('node-dns-evidence-outcome')).toContainText('applied');
    await expect(
      page.getByTestId('node-dns-evidence-feed-malware-domains-default'),
    ).toContainText('rules=1420');

    // 412 conflict handling on PUT /publication with stale ETag
    const bumpRes = await request.post(`${BASE_URL}/__e2e/bump-node-version`, {
      data: { node_id: tpeNode.node_id },
    });
    expect(bumpRes.ok()).toBeTruthy();
    await page.getByTestId('pub-group-checkbox-asia-premium').check();
    await page.getByTestId('publish-node-btn').click();
    await expect(page.getByTestId('conflict-dialog')).toBeVisible();
    await page.getByTestId('conflict-reload-btn').click();
    await expect(page.getByTestId('conflict-dialog')).toHaveCount(0);
    await expect(page.getByTestId('pub-group-checkbox-asia-premium')).not.toBeChecked();

    // Replace exit group attachment set together with publication intent via PUT /publication
    await page.getByTestId('pub-group-checkbox-asia-premium').check();
    await expect(page.getByTestId('pub-group-checkbox-asia-premium')).toBeChecked();
    await page.getByTestId('publish-node-btn').click();
    await waitForOperationHeadline(
      page,
      request,
      'admin',
      '已套用（已取得實際觀測確認）',
    );
    await expect(page.getByTestId('node-attached-groups')).toContainText(
      'asia-premium (Asia Premium Exits)',
    );
    await expect(page.getByTestId('node-attached-groups')).toContainText(
      'standard-exits (Standard Exit Group)',
    );

    await page.getByTestId('locale-switcher').selectOption('en');
    await expect(page.getByTestId('node-publication-groups-editor')).toContainText(
      'Exit Attached Groups (D-057)',
    );
    await expect(page.getByTestId('node-checks-section')).toContainText(
      'Node Local Safety & Health Checks (D-058)',
    );
    await expect(page.getByTestId('node-dns-evidence-disclaimer')).toContainText(
      'reported by the node and are never central proof of apply',
    );
    await page.getByTestId('locale-switcher').selectOption('zh-TW');

    // Verify "not yet reported" state when dns_policy_evidence is null on secondary offline node (D-056)
    await page.goto(`/admin/nodes/${tyoNode!.node_id}`);
    await expect(page.getByTestId('admin-node-detail-page')).toBeVisible();
    await expect(page.getByTestId('node-dns-evidence-not-reported')).toContainText(
      '尚未回報 (not yet reported)',
    );
    await page.getByTestId('locale-switcher').selectOption('en');
    await expect(page.getByTestId('node-dns-evidence-not-reported')).toContainText(
      'Not yet reported',
    );
    await page.getByTestId('locale-switcher').selectOption('zh-TW');

    // 3. Members never see group_id values on member exit views (D-057)
    const restoreMemberRes = await request.post(
      `${BASE_URL}/__e2e/restore-default-member`,
    );
    expect(restoreMemberRes.ok()).toBeTruthy();
    await page.goto('/member/exits');
    await expect(page.getByTestId('member-exits-page')).toBeVisible();
    const exitRows = page.locator('[data-testid^="member-exit-row-"]');
    await expect(exitRows.first()).toBeVisible();
    await expect(exitRows).not.toHaveCount(0);
    await expect(page.getByTestId('member-exits-table')).toContainText('tw-tpe');
    await expect(page.locator('body')).not.toContainText(standardExitsGroup!.group_id);
    await expect(page.locator('body')).not.toContainText(asiaPremiumGroup!.group_id);
    await expect(page.locator('body')).not.toContainText('standard-exits');
    await expect(page.locator('body')).not.toContainText('asia-premium');

    await page.getByTestId('locale-switcher').selectOption('en');
    await expect(page.getByTestId('manual-exit-selection-notice')).toContainText(
      'NetBird',
    );
    await expect(page.locator('body')).not.toContainText(standardExitsGroup!.group_id);
    await expect(page.locator('body')).not.toContainText(asiaPremiumGroup!.group_id);
    await expect(page.locator('body')).not.toContainText('standard-exits');
    await expect(page.locator('body')).not.toContainText('asia-premium');
    await page.getByTestId('locale-switcher').selectOption('zh-TW');

    // 4. DNS Policy threat_feed_status and ThreatFeedPayload.update_interval_seconds (D-058) in zh-TW and en
    await page.goto('/admin/policies/dns');
    await expect(page.getByTestId('admin-dns-policy-page')).toBeVisible();
    await expect(page.getByTestId('threat-feed-status-section')).toBeVisible();
    await expect(
      page.getByTestId('threat-feed-state-malware-domains-default'),
    ).toContainText(
      '初始空白狀態（尚未完成首次抓取，bootstrap-empty / last_result=never）',
    );

    await page.getByTestId('locale-switcher').selectOption('en');
    await expect(
      page.getByTestId('threat-feed-state-malware-domains-default'),
    ).toContainText(
      'Bootstrap empty (initial fetch not yet completed, last_result=never)',
    );
    await page.getByTestId('locale-switcher').selectOption('zh-TW');

    // Verify contract default (21600) and client-side bounds (300-604800)
    await page.getByTestId('extra-policy-schema-select').selectOption('threat_feed');
    await expect(page.getByTestId('threat-feed-update-interval-input')).toHaveValue(
      '21600',
    );
    await page.getByTestId('threat-feed-update-interval-input').fill('120');
    await page.getByTestId('create-extra-policy-revision-btn').click();
    await expect(page.getByTestId('threat-feed-interval-field-error')).toBeVisible();

    // Bypass client check to verify server-side 422 validation field error detail
    await page.getByTestId('threat-feed-bypass-client-check').check();
    await page.getByTestId('create-extra-policy-revision-btn').click();
    await expect(page.getByTestId('threat-feed-interval-field-error')).toContainText(
      '/payload/update_interval_seconds',
    );
    await expect(page.getByTestId('dns-api-error')).toBeVisible();

    // Valid update_interval_seconds (3600) creates threat_feed revision
    await page.getByTestId('threat-feed-bypass-client-check').uncheck();
    await page.getByTestId('threat-feed-update-interval-input').fill('3600');
    await page.getByTestId('create-extra-policy-revision-btn').click();
    await expect(page.getByTestId('threat-feed-interval-field-error')).toHaveCount(0);
    await expect(page.getByTestId('dns-api-error')).toHaveCount(0);
    await expect(page.getByTestId('policy-definitions-table')).toContainText('#2');
  });

  test('TestOperator_FupPolicyActivationAndAuditEvents', async ({ page, request }) => {
    // 1. Navigate to /admin/policies/fup and verify TEST ONLY fair-use policy & overrides
    await page.goto('/admin/policies/fup');
    await expect(page.getByTestId('admin-fup-policy-page')).toBeVisible();
    await expect(page.getByTestId('fup-test-only-banner')).toBeVisible();

    await page
      .getByTestId('fup-override-member-id-input')
      .fill('0195f100-0000-7000-8000-000000000101');
    await page.getByTestId('fup-add-override-btn').click();
    await expect(page.getByTestId('fup-overrides-table')).toContainText(
      '0195f100-0000-7000-8000-000000000101',
    );

    // 2. Validate, create immutable revision, and activate with If-Match (202 operation)
    await page.getByTestId('fup-submit-btn').click();
    await expect(page.getByTestId('fup-validation-passed')).toBeVisible();
    await waitForOperationHeadline(
      page,
      request,
      'admin',
      '已套用（已取得實際觀測確認）',
    );
    await expect(page.getByTestId('op-step-desired_state_compile')).toContainText(
      'succeeded',
    );

    // 3. Navigate to /admin/audit and verify real audit events recorded by fvpn-control
    await page.goto('/admin/audit');
    await expect(page.getByTestId('admin-audit-page')).toBeVisible();
    await expect(page.getByTestId('audit-table')).toBeVisible();
    const auditRows = page.locator('[data-testid^="audit-row-"]');
    await expect(auditRows.first()).toBeVisible();
    await expect(auditRows).not.toHaveCount(0);
    await expect(page.getByTestId('audit-table')).toContainText('policy.activate');
  });

  test('TestD065_OperatorFleetConfigSettings_UpdateAuditReload412ConflictAndRestore', async ({
    page,
    request,
  }) => {
    try {
      // 1. Observe initial fleet-config via API and on /admin/settings
      const initialGetRes = await request.get(`${BASE_URL}/api/v1/admin/fleet-config`);
      expect(initialGetRes.status()).toBe(200);
      const initialConfig = (await initialGetRes.json()) as {
        max_offline_lease_seconds: number;
        default_max_offline_lease_seconds: number;
        resource_version: string;
        updated_at: string;
      };
      expect(initialConfig.max_offline_lease_seconds).toBe(86400);
      expect(initialConfig.default_max_offline_lease_seconds).toBe(86400);

      await page.goto('/admin/settings');
      await expect(page.getByTestId('admin-settings-page')).toBeVisible();
      await expect(page.getByTestId('nav-admin-settings')).toBeVisible();
      await expect(page.getByTestId('fleet-config-current-seconds')).toHaveText(
        '86400 s',
      );
      await expect(page.getByTestId('fleet-config-default-seconds')).toHaveText(
        '86400 s',
      );
      await expect(page.getByTestId('fleet-config-updated-at')).toContainText(/Z\)/);
      await expect(page.getByTestId('fleet-config-semantics-copy')).toContainText(
        '86400',
      );

      // Verify sub-900 s warning appears below 900 s without blocking submission
      await page.getByTestId('fleet-config-lease-seconds-input').fill('600');
      await expect(page.getByTestId('fleet-config-sub-900-warning')).toBeVisible();
      await expect(page.getByTestId('fleet-config-submit-btn')).toBeEnabled();

      // 2. Change max_offline_lease_seconds to 3600 with a reason and observe the 200 response
      const updateReason = 'Shorten new leases for the AT29 lab.';
      await page.getByTestId('fleet-config-lease-seconds-input').fill('3600');
      await expect(page.getByTestId('fleet-config-sub-900-warning')).toHaveCount(0);
      await page.getByTestId('fleet-config-reason-input').fill(updateReason);

      const put200Promise = page.waitForResponse(
        (res) =>
          res.url().includes('/api/v1/admin/fleet-config') &&
          res.request().method() === 'PUT',
      );
      await page.getByTestId('fleet-config-submit-btn').click();
      const put200Res = await put200Promise;
      expect(put200Res.status()).toBe(200);

      await expect(page.getByTestId('fleet-config-saved-banner')).toBeVisible();
      await expect(page.getByTestId('fleet-config-saved-banner')).toContainText('3600');
      await expect(page.getByTestId('fleet-config-current-seconds')).toHaveText('3600 s');
      await expect(page.getByTestId('fleet-config-audit-link')).toBeVisible();

      // 3. Follow the audit link to /admin/audit?resource_type=fleet_config and observe the fleet_config.update row with before/after values
      await page.getByTestId('fleet-config-audit-link').click();
      await expect(page.getByTestId('admin-audit-page')).toBeVisible();
      await expect(page.getByTestId('audit-resource-filter')).toHaveValue('fleet_config');
      const auditRows = page.locator('[data-testid^="audit-row-"]');
      await expect(auditRows.first()).toBeVisible();
      await expect(auditRows).not.toHaveCount(0);
      await expect(auditRows.first()).toContainText('fleet_config.update');
      await expect(auditRows.first()).toContainText('86400');
      await expect(auditRows.first()).toContainText('3600');
      await expect(auditRows.first()).toContainText(updateReason);

      // 4. Return to /admin/settings and reload to see the new value (3600)
      await page.goto('/admin/settings');
      await expect(page.getByTestId('admin-settings-page')).toBeVisible();
      await page.reload();
      await expect(page.getByTestId('admin-settings-page')).toBeVisible();
      await expect(page.getByTestId('fleet-config-current-seconds')).toHaveText('3600 s');
      await expect(page.getByTestId('fleet-config-lease-seconds-input')).toHaveValue(
        '3600',
      );

      // 5. Provoke a 412 RESOURCE_VERSION_CONFLICT by advancing the resource via API while the page holds the stale ETag
      const midGetRes = await request.get(`${BASE_URL}/api/v1/admin/fleet-config`);
      expect(midGetRes.status()).toBe(200);
      const midConfig = (await midGetRes.json()) as { resource_version: string };
      const concurrentPutRes = await request.put(
        `${BASE_URL}/api/v1/admin/fleet-config`,
        {
          headers: {
            'If-Match': `"${midConfig.resource_version}"`,
            'Idempotency-Key': `0195f100-0000-7000-8000-${Date.now().toString().slice(-12).padStart(12, '0')}`,
            'Content-Type': 'application/json',
          },
          data: {
            max_offline_lease_seconds: 1800,
            reason: 'Concurrent operator update via API',
          },
        },
      );
      expect(concurrentPutRes.status()).toBe(200);
      const concurrentUpdated = (await concurrentPutRes.json()) as {
        resource_version: string;
      };

      await page.getByTestId('fleet-config-lease-seconds-input').fill('7200');
      await page.getByTestId('fleet-config-reason-input').fill('Attempt with stale ETag');

      const put412Promise = page.waitForResponse(
        (res) =>
          res.url().includes('/api/v1/admin/fleet-config') &&
          res.request().method() === 'PUT',
      );
      await page.getByTestId('fleet-config-submit-btn').click();
      const put412Res = await put412Promise;
      expect(put412Res.status()).toBe(412);

      await expect(page.getByTestId('fleet-config-conflict-banner')).toBeVisible();
      await expect(page.getByTestId('fleet-config-conflict-version')).toHaveText(
        concurrentUpdated.resource_version,
      );
      await expect(page.getByTestId('conflict-dialog')).toBeVisible();

      // Reload from conflict dialog to load the latest version (1800)
      await page.getByTestId('conflict-reload-btn').click();
      await expect(page.getByTestId('conflict-dialog')).toHaveCount(0);
      await expect(page.getByTestId('fleet-config-current-seconds')).toHaveText('1800 s');
      await expect(page.getByTestId('fleet-config-etag')).toHaveText(
        `ETag: "${concurrentUpdated.resource_version}"`,
      );

      // 6. Restore 86400 via the page and verify via both page and API
      await page.getByTestId('fleet-config-lease-seconds-input').fill('86400');
      await page
        .getByTestId('fleet-config-reason-input')
        .fill('Restore default 86400 ceiling after E2E test.');
      const restorePutPromise = page.waitForResponse(
        (res) =>
          res.url().includes('/api/v1/admin/fleet-config') &&
          res.request().method() === 'PUT',
      );
      await page.getByTestId('fleet-config-submit-btn').click();
      const restorePutRes = await restorePutPromise;
      expect(restorePutRes.status()).toBe(200);

      await expect(page.getByTestId('fleet-config-current-seconds')).toHaveText(
        '86400 s',
      );
      const finalGetRes = await request.get(`${BASE_URL}/api/v1/admin/fleet-config`);
      expect(finalGetRes.status()).toBe(200);
      const finalConfig = (await finalGetRes.json()) as {
        max_offline_lease_seconds: number;
      };
      expect(finalConfig.max_offline_lease_seconds).toBe(86400);
    } finally {
      const checkRes = await request.get(`${BASE_URL}/api/v1/admin/fleet-config`);
      if (checkRes.ok()) {
        const current = (await checkRes.json()) as {
          max_offline_lease_seconds: number;
          resource_version: string;
        };
        if (current.max_offline_lease_seconds !== 86400) {
          await request.put(`${BASE_URL}/api/v1/admin/fleet-config`, {
            headers: {
              'If-Match': `"${current.resource_version}"`,
              'Idempotency-Key': `0195f100-ffff-7000-8000-${Date.now().toString().slice(-12).padStart(12, '0')}`,
              'Content-Type': 'application/json',
            },
            data: {
              max_offline_lease_seconds: 86400,
              reason: 'Finally block restore 86400 ceiling.',
            },
          });
        }
      }
    }
  });
});
