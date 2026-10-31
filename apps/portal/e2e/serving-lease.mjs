import crypto from 'node:crypto';

// TEST ONLY agent simulator. Central alone creates lease IDs, grants and deadlines.
export function createServingLeaseSession({
  nodeId,
  incarnationId,
  request,
  heartbeatMs = 250,
  now = () => performance.now(),
}) {
  let lease = null;
  let deadline = 0;
  let renewAt = 0;
  let provisionalWait = heartbeatMs;
  let queue = Promise.resolve();

  function renewalDue() {
    return !lease || now() >= renewAt;
  }

  async function ensureOnce(desired) {
    if (!desired?.config_generation || !Array.isArray(desired.device_bindings)) {
      throw new Error('Serving lease requires a central-issued desired state');
    }
    if (lease?.config_generation !== desired.config_generation || renewalDue()) {
      const sentAt = now();
      const envelope = await request({
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': crypto.randomUUID(),
        },
        body: JSON.stringify({
          renewal_request_id: crypto.randomUUID(),
          node_id: nodeId,
          node_incarnation_id: incarnationId,
          current_lease_id: lease?.lease_id ?? null,
          applied_config_generation: desired.config_generation,
        }),
      });
      const issued = envelope.signed;
      const windowMs = Date.parse(issued?.expires_at) - Date.parse(issued?.issued_at);
      if (
        issued?.node_id !== nodeId ||
        issued?.node_incarnation_id !== incarnationId ||
        !issued.lease_id ||
        !Array.isArray(issued.members) ||
        !Number.isFinite(windowMs) ||
        windowMs <= 0
      ) {
        throw new Error('Central returned an invalid serving lease');
      }
      lease = issued;
      // §16.3: anchor at request send, never restart TTL on receipt or observation.
      deadline = sentAt + windowMs;
      const normalDelay = windowMs - Math.min(30_000, windowMs / 2);
      renewAt =
        sentAt + Math.min(normalDelay, lease.members.length ? Infinity : provisionalWait);
      provisionalWait = lease.members.length
        ? heartbeatMs
        : Math.min(60_000, provisionalWait * 2);
    }
    return lease;
  }

  function ensure(desired) {
    // Observation callers and the heartbeat timer share one renewal stream.
    const pending = queue.then(() => ensureOnce(desired));
    queue = pending.catch(() => {});
    return pending;
  }

  function evidence(desired, observedAt) {
    if (!lease) throw new Error('No central-issued serving lease');
    let reason = null;
    let code = null;
    let detail = null;
    if (lease.config_generation !== desired.config_generation) {
      reason = 'policy_missing';
      code = 'STALE_GENERATION';
      detail = 'Serving lease is not for the applied generation.';
    } else if (now() >= deadline) {
      reason = 'lease_expired';
      code = 'LEASE_EXPIRED';
      detail = 'Serving lease is not in force.';
    } else if (desired.device_bindings.length && !lease.members.length) {
      reason = 'awaiting_member_grants';
      code = 'AWAITING_MEMBER_GRANTS';
      detail = 'Serving lease has no member grants yet.';
    }
    return {
      lease: { lease_id: lease.lease_id, expires_at: lease.expires_at },
      member_forwarding_enabled: !reason,
      forwarding_block_reasons: reason ? [reason] : [],
      checks: [
        ['lease', detail, 'serving lease valid and unexpired'],
        [
          'qos',
          'Rate policy could not be applied.',
          'tc HTB + fq_codel classes installed',
        ],
        [
          'source_binding',
          'No member binding was admitted.',
          'member peer source IP binding verified',
        ],
      ].map(([name, failedDetail, passedDetail]) => ({
        name,
        status: code ? 'failed' : 'passed',
        observed_at: observedAt,
        error_code: code,
        detail: code ? failedDetail : passedDetail,
      })),
      errors: code
        ? [{ code, stage: 'lease', message: detail, observed_at: observedAt }]
        : [],
    };
  }

  return { ensure, evidence, renewalDue };
}
