import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import test from 'node:test';
import { createServingLeaseSession } from './serving-lease.mjs';

// TEST ONLY: unit-level central response double, never sent to the live stack.
function fixture() {
  const nodeId = crypto.randomUUID();
  const incarnationId = crypto.randomUUID();
  const desired = { config_generation: '9007199254740993', device_bindings: [{}] };
  const requests = [];
  let elapsed = 0;
  let members = [{}];
  let fail = false;
  let latency = 0;
  let issued;
  const session = createServingLeaseSession({
    nodeId,
    incarnationId,
    now: () => elapsed,
    request: async (options) => {
      requests.push({ ...options, body: JSON.parse(options.body) });
      if (fail) throw new Error('central unavailable');
      issued = {
        node_id: nodeId,
        node_incarnation_id: incarnationId,
        lease_id: crypto.randomUUID(),
        config_generation: desired.config_generation,
        issued_at: new Date(Date.UTC(2026, 9, 31) + elapsed).toISOString(),
        expires_at: new Date(Date.UTC(2026, 9, 31) + elapsed + 60_000).toISOString(),
        members,
      };
      elapsed += latency;
      return { signed: issued };
    },
  });
  return {
    session,
    desired,
    requests,
    nodeId,
    incarnationId,
    issued: () => issued,
    advance: (ms) => {
      elapsed += ms;
    },
    members: (value) => {
      members = value;
    },
    fail: () => {
      fail = true;
    },
    latency: (ms) => {
      latency = ms;
    },
  };
}

test('D-101: central lease identity and expiry survive observations; renewals have fresh IDs', async () => {
  const f = fixture();
  await Promise.all([f.session.ensure(f.desired), f.session.ensure(f.desired)]);
  assert.equal(f.requests.length, 1);
  assert.deepEqual(f.requests[0].body, {
    renewal_request_id: f.requests[0].body.renewal_request_id,
    node_id: f.nodeId,
    node_incarnation_id: f.incarnationId,
    current_lease_id: null,
    applied_config_generation: '9007199254740993',
  });
  const original = f.issued();
  assert.deepEqual(f.session.evidence(f.desired).lease, {
    lease_id: original.lease_id,
    expires_at: original.expires_at,
  });
  f.advance(29_999);
  assert.equal(f.session.renewalDue(), false);
  await f.session.ensure(f.desired);
  assert.equal(f.requests.length, 1);
  f.advance(1);
  assert.equal(f.session.renewalDue(), true);
  await f.session.ensure(f.desired);
  assert.equal(f.requests.length, 2);
  assert.equal(f.requests[1].body.current_lease_id, original.lease_id);
  assert.notEqual(
    f.requests[0].body.renewal_request_id,
    f.requests[1].body.renewal_request_id,
  );
  assert.notEqual(
    f.requests[0].headers['Idempotency-Key'],
    f.requests[1].headers['Idempotency-Key'],
  );
  assert.notEqual(f.issued().lease_id, original.lease_id);
});

test('D-067/D-071: provisional report blocks forwarding, then renewed member grants unblock it', async () => {
  const f = fixture();
  f.members([]);
  await f.session.ensure(f.desired);
  const at = '2026-10-31T00:00:00Z';
  const report = f.session.evidence(f.desired, at);
  assert.equal(report.member_forwarding_enabled, false);
  assert.deepEqual(report.forwarding_block_reasons, ['awaiting_member_grants']);
  assert.deepEqual(
    report.checks.map(({ name, status, error_code, detail }) => [
      name,
      status,
      error_code,
      detail,
    ]),
    [
      [
        'lease',
        'failed',
        'AWAITING_MEMBER_GRANTS',
        'Serving lease has no member grants yet.',
      ],
      ['qos', 'failed', 'AWAITING_MEMBER_GRANTS', 'Rate policy could not be applied.'],
      [
        'source_binding',
        'failed',
        'AWAITING_MEMBER_GRANTS',
        'No member binding was admitted.',
      ],
    ],
  );
  assert.deepEqual(report.errors, [
    {
      code: 'AWAITING_MEMBER_GRANTS',
      stage: 'lease',
      message: 'Serving lease has no member grants yet.',
      observed_at: at,
    },
  ]);
  f.advance(250);
  await f.session.ensure(f.desired);
  f.advance(499);
  assert.equal(f.session.renewalDue(), false);
  f.members([{}]);
  f.advance(1);
  await f.session.ensure(f.desired);
  const populated = f.session.evidence(f.desired, at);
  assert.equal(populated.member_forwarding_enabled, true);
  assert.deepEqual(populated.errors, []);
  assert.deepEqual(populated.forwarding_block_reasons, []);
  assert.ok(
    populated.checks.every((c) => c.status === 'passed' && c.error_code === null),
  );
});

test('D-101: new desired generation renews immediately without rounding decimal strings', async () => {
  const f = fixture();
  f.members([]);
  f.desired.device_bindings = [];
  await f.session.ensure(f.desired);
  assert.deepEqual(f.session.evidence(f.desired).forwarding_block_reasons, []);
  f.desired.config_generation = '9007199254740994';
  f.desired.device_bindings = [{}];
  assert.deepEqual(f.session.evidence(f.desired).forwarding_block_reasons, [
    'policy_missing',
  ]);
  f.members([{}]);
  await f.session.ensure(f.desired);
  assert.equal(f.requests[1].body.applied_config_generation, '9007199254740994');
  assert.equal(f.session.evidence(f.desired).member_forwarding_enabled, true);
});

test('§16.3: latency and failed renewal never extend the original deadline', async () => {
  const f = fixture();
  f.latency(45_000);
  await f.session.ensure(f.desired);
  const original = f.session.evidence(f.desired).lease;
  assert.equal(f.session.renewalDue(), true);
  f.fail();
  await assert.rejects(f.session.ensure(f.desired), /central unavailable/);
  f.advance(15_000);
  const expired = f.session.evidence(f.desired);
  assert.equal(expired.member_forwarding_enabled, false);
  assert.deepEqual(expired.lease, original);
  assert.deepEqual(expired.forwarding_block_reasons, ['lease_expired']);
  assert.equal(expired.errors[0].code, 'LEASE_EXPIRED');
});
