import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import test from 'node:test';
import { createAppliedStateFixture } from './applied-state.mjs';

const sha256 = (bytes) => crypto.createHash('sha256').update(bytes).digest('hex');
function fixture(feeds = []) {
  const bytes = Buffer.from(
    JSON.stringify({ kind: 'resolver_document', schema_version: 1, feeds }),
  );
  const desired = {
    config_generation: '9007199254740993',
    binding_revision: '18446744073709551615',
    policy_refs: {
      authorization_generation: '0',
      fup_generation: '9007199254740994',
      dns_generation: '3',
    },
    device_bindings: [],
    dns: {
      resolver_document: {
        document_id: '01900000-0000-7000-8000-000000000056',
        generation: '3',
        content_sha256: sha256(bytes),
        size_bytes: bytes.length,
      },
    },
  };
  return { bytes, desired };
}

test('D093-06: acknowledge exact generations only after the complete fixture swap', async () => {
  const { bytes, desired } = fixture();
  let releaseFetch;
  const state = createAppliedStateFixture({
    fetchResolverDocument: () =>
      new Promise((r) => {
        releaseFetch = r;
      }),
  });
  const applying = state.apply(desired);
  await new Promise((r) => setImmediate(r));
  assert.throws(state.read, /No fixture state has been applied/);
  desired.config_generation = '9'; // The staged candidate is an isolated snapshot.
  releaseFetch(bytes);
  const applied = await applying;
  assert.equal(applied.desired.config_generation, '9007199254740993');
  assert.equal(applied.desired.binding_revision, '18446744073709551615');
  assert.equal(applied.desired.policy_refs.authorization_generation, '0');
  assert.equal(applied.desired.policy_refs.fup_generation, '9007199254740994');
  applied.desired.config_generation = '8';
  assert.equal(state.read().desired.config_generation, '9007199254740993');
});

test('D093-06: missing or inexact generations never gain a fallback', async () => {
  for (const change of [
    (d) => {
      delete d.config_generation;
    },
    (d) => {
      delete d.binding_revision;
    },
    (d) => {
      delete d.policy_refs.authorization_generation;
    },
    (d) => {
      delete d.policy_refs.fup_generation;
    },
    (d) => {
      delete d.policy_refs.dns_generation;
    },
    (d) => {
      d.config_generation = 9007199254740992;
    },
    (d) => {
      d.config_generation = '18446744073709551616';
    },
  ]) {
    const { bytes, desired } = fixture();
    const state = createAppliedStateFixture({ fetchResolverDocument: async () => bytes });
    change(desired);
    await assert.rejects(state.apply(desired), /decimal generations/);
    assert.throws(state.read, /No fixture state has been applied/);
  }
});

test('D093-10: DNS metadata comes from fetched bytes, including actual empty feeds', async () => {
  for (const feeds of [
    [],
    [
      {
        feed_code: 'test-feed',
        content_sha256: sha256('TEST ONLY feed bytes'),
        entries: [{ domain: 'example.invalid' }],
        rule_count: 1,
      },
    ],
  ]) {
    const { bytes, desired } = fixture(feeds);
    const state = createAppliedStateFixture({
      fetchResolverDocument: async (id) => {
        assert.equal(id, desired.dns.resolver_document.document_id);
        return bytes;
      },
    });
    const applied = await state.apply(desired);
    assert.equal(applied.dnsEvidence.content_sha256, sha256(bytes));
    assert.equal(applied.dnsEvidence.generation, '3');
    assert.deepEqual(
      applied.dnsEvidence.feeds,
      feeds.map(({ entries, ...feed }) => ({ ...feed, rule_count: entries.length })),
    );
    desired.dns.resolver_document = null;
    assert.equal((await state.apply(desired)).dnsEvidence, null);
  }
});

test('D093-06/D093-10: failed DNS fetch, size, hash or feed validation preserves last applied state', async () => {
  const { bytes, desired } = fixture();
  let fetch = async () => bytes;
  const state = createAppliedStateFixture({ fetchResolverDocument: (id) => fetch(id) });
  const previous = await state.apply(desired);
  desired.config_generation = '9007199254740994';
  for (const result of [
    async () => {
      throw new Error('fetch failed');
    },
    async () => Buffer.concat([bytes, Buffer.from(' ')]),
    async () => Buffer.alloc(bytes.length),
  ]) {
    fetch = result;
    await assert.rejects(state.apply(desired));
    assert.deepEqual(state.read(), previous);
  }
  const bad = fixture([
    {
      feed_code: 'test-feed',
      content_sha256: sha256('TEST ONLY'),
      entries: [],
      rule_count: 1420,
    },
  ]);
  fetch = async () => bad.bytes;
  await assert.rejects(state.apply(bad.desired), /Invalid resolver feed evidence/);
  assert.deepEqual(state.read(), previous);
  fetch = async () => bytes;
  assert.equal(
    (await state.apply(desired)).desired.config_generation,
    '9007199254740994',
  );
});
