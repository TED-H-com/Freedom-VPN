import { expect, test } from '@playwright/test';

const BASE_URL = process.env.PORTAL_E2E_BASE_URL || 'http://127.0.0.1:4173';
const SEEDED_BATCH_ID = '85ef720a-31a0-4250-90d2-4b92c58695bc';
const SEEDED_NODE_2_ID = '0195f100-0000-7000-8000-000000000202';

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
    const checkoutUrlText = await page.getByTestId('hosted-redirect-url').textContent();
    expect(checkoutUrlText).toContain(
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

    // 3. Revoke the first device (202 Accepted operation -> succeeded) and verify count drops to 2 / 3 (AT02)
    const firstDeviceId = seededDeviceIds[0];
    await page.getByTestId(`revoke-own-device-${firstDeviceId}`).click();

    await expect(page.getByTestId('operation-banner')).toBeVisible();
    await expect(page.getByTestId('device-quota-badge')).toContainText('2 / 3', {
      timeout: 15_000,
    });
    await expect(page.getByTestId(`device-status-${firstDeviceId}`)).toHaveText(
      'revoked',
    );

    // 4. Switch to Admin view (/admin/members/{id}) and suspend the member with If-Match (AT04)
    await page.goto(`/admin/members/${activeMemberId}`);
    await expect(page.getByTestId('admin-member-detail-page')).toBeVisible();
    await page
      .locator('#suspend-reason-input')
      .fill('AT04 E2E compliance suspension check');
    await page.getByTestId('suspend-member-btn').click();

    await expect(page.getByTestId('operation-banner')).toBeVisible();
    await expect(page.getByTestId('member-detail-entitlement-badge')).toHaveText(
      'suspended',
      { timeout: 15_000 },
    );

    // 5. Issue a finite entitlement grant on the member and verify 202 operation banner
    await page.getByTestId('create-grant-btn').click();
    await expect(page.getByTestId('operation-banner')).toBeVisible();

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
      items: Array<{ node_id: string; name?: string | null }>;
    };
    const enrolledNode =
      nodesJson.items.find((n) => n.name === 'tw-tpe-01-updated') ??
      nodesJson.items.find((n) => n.name === 'tw-tpe-01') ??
      nodesJson.items[0];
    const enrolledNodeId = enrolledNode!.node_id;

    // 1. Navigate to the seeded fleet update batch on /admin/updates
    await page.goto(`/admin/updates?batch_id=${SEEDED_BATCH_ID}`);
    await expect(page.getByTestId('admin-fleet-updates-page')).toBeVisible();
    await expect(page.getByTestId('update-batch-progress-card')).toBeVisible();

    // 2. Verify 12-bucket target accounting and verified success count
    await expect(page.getByTestId('bucket-sum-check')).toContainText('2 / 2');
    await expect(page.getByTestId('verified-success-summary')).toContainText('1');
    await expect(page.getByTestId('bucket-succeeded')).toContainText('1');
    await expect(page.getByTestId('bucket-failed')).toContainText('1');

    // 3. Verify per-target `needs_attention`, `unconfirmed`, and D-048 `policy_generation` / `policy_refs` evidence
    await expect(
      page.getByTestId(`target-needs-attention-${SEEDED_NODE_2_ID}`),
    ).toBeVisible();
    await expect(
      page.getByTestId(`target-unconfirmed-${SEEDED_NODE_2_ID}`),
    ).toBeVisible();
    await expect(page.getByTestId(`target-policy-gen-${enrolledNodeId}`)).toContainText(
      '2',
    );
    await expect(page.getByTestId(`target-policy-refs-${enrolledNodeId}`)).toContainText(
      '0195f100-0000-7000-8000-000000000401',
    );
    await expect(page.getByTestId(`target-policy-gen-${SEEDED_NODE_2_ID}`)).toContainText(
      '未回報 (not reported)',
    );
    await expect(
      page.getByTestId(`target-policy-refs-${SEEDED_NODE_2_ID}`),
    ).toContainText('未回報 (not reported)');

    // 4. Verify locale switching (zh-TW <-> en) and accessible labels
    await page.getByTestId('locale-switcher').selectOption('en');
    await expect(
      page.getByTestId(`target-needs-attention-${SEEDED_NODE_2_ID}`),
    ).toContainText('Needs Attention');
    await expect(
      page.getByTestId(`target-unconfirmed-${SEEDED_NODE_2_ID}`),
    ).toContainText('Unconfirmed');
    await expect(page.getByTestId(`target-policy-gen-${SEEDED_NODE_2_ID}`)).toContainText(
      'not reported',
    );
    await page.getByTestId('locale-switcher').selectOption('zh-TW');
  });

  test('TestOperator_FupPolicyActivationAndAuditEvents', async ({ page }) => {
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
    await expect(page.getByTestId('operation-banner')).toBeVisible();

    // 3. Navigate to /admin/audit and verify real audit events recorded by fvpn-control
    await page.goto('/admin/audit');
    await expect(page.getByTestId('admin-audit-page')).toBeVisible();
    await expect(page.getByTestId('audit-table')).toBeVisible();
    const rowCount = await page.locator('[data-testid^="audit-row-"]').count();
    expect(rowCount).toBeGreaterThan(0);
  });
});
