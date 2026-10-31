import crypto from 'node:crypto';

function requireGeneration(value) {
  if (
    typeof value !== 'string' ||
    !/^(0|[1-9][0-9]{0,19})$/.test(value) ||
    BigInt(value) > 18446744073709551615n
  ) {
    throw new Error('Fixture apply requires central-issued decimal generations');
  }
}

// TEST ONLY in-memory apply boundary, not kernel or AdGuard enforcement.
// Desired/lease signatures, key rotation and clock anomalies remain agent work.
export function createAppliedStateFixture({
  fetchResolverDocument,
  now = () => new Date(),
}) {
  let applied = null;
  let queue = Promise.resolve();

  async function applyOnce(desired) {
    const staged = structuredClone(desired);
    for (const value of [
      staged?.config_generation,
      staged?.binding_revision,
      staged?.policy_refs?.authorization_generation,
      staged?.policy_refs?.fup_generation,
      staged?.policy_refs?.dns_generation,
    ]) {
      requireGeneration(value);
    }
    if (!Array.isArray(staged.device_bindings) || !staged.dns) {
      throw new Error('Fixture apply requires desired bindings and DNS configuration');
    }

    let dnsEvidence = null;
    const ref = staged.dns.resolver_document;
    if (ref != null) {
      requireGeneration(ref.generation);
      if (
        typeof ref.document_id !== 'string' ||
        !/^[0-9a-f-]{36}$/.test(ref.document_id) ||
        !/^[0-9a-f]{64}$/.test(ref.content_sha256) ||
        !Number.isInteger(ref.size_bytes) ||
        ref.size_bytes < 1 ||
        ref.size_bytes > 1048576
      ) {
        throw new Error('Invalid resolver document reference');
      }
      const bytes = await fetchResolverDocument(ref.document_id);
      if (!Buffer.isBuffer(bytes) || bytes.length !== ref.size_bytes) {
        throw new Error('Resolver document size mismatch');
      }
      const hash = crypto.createHash('sha256').update(bytes).digest('hex');
      if (hash !== ref.content_sha256) {
        throw new Error('Resolver document hash mismatch');
      }
      const doc = JSON.parse(bytes.toString('utf8'));
      if (
        doc?.kind !== 'resolver_document' ||
        doc.schema_version !== 1 ||
        !Array.isArray(doc.feeds) ||
        doc.feeds.length > 32
      ) {
        throw new Error('Invalid resolver document');
      }
      const seen = new Set();
      const feeds = doc.feeds.map((feed) => {
        if (
          typeof feed?.feed_code !== 'string' ||
          !/^[a-z0-9][a-z0-9_-]{0,63}$/.test(feed.feed_code) ||
          seen.has(feed.feed_code) ||
          !/^[0-9a-f]{64}$/.test(feed.content_sha256) ||
          !Array.isArray(feed.entries) ||
          feed.entries.length > 500000 ||
          feed.rule_count !== feed.entries.length
        ) {
          throw new Error('Invalid resolver feed evidence');
        }
        seen.add(feed.feed_code);
        return {
          feed_code: feed.feed_code,
          content_sha256: feed.content_sha256,
          rule_count: feed.entries.length,
        };
      });
      dnsEvidence = {
        document_id: ref.document_id,
        content_sha256: hash,
        generation: ref.generation,
        outcome: 'applied',
        applied_at: now().toISOString(),
        feeds,
        error_code: null,
      };
    }

    // Swap only after the whole candidate is ready. A failed fetch/validation
    // keeps the previous snapshot and cannot acknowledge the desired generation.
    applied = { desired: staged, dnsEvidence };
    return read();
  }

  function read() {
    if (!applied) throw new Error('No fixture state has been applied');
    return structuredClone(applied);
  }

  function apply(desired) {
    const candidate = structuredClone(desired);
    const pending = queue.then(() => applyOnce(candidate));
    queue = pending.catch(() => {});
    return pending;
  }

  return { apply, read };
}
