#!/usr/bin/env node
/**
 * Freedom VPN Portal Live Development & E2E Stack (`portal-dev-stack`).
 *
 * Responsibilities (AGENTS.md, Project Spec §18, D-026, D-031, D-032):
 * 1. Creates an isolated PostgreSQL database from `$FVPN_TEST_DATABASE_URL` and drops it on exit.
 * 2. Generates ephemeral TLS certificates, mTLS node CA, and FileProvider vault.
 * 3. Runs local HTTPS mock OIDC provider, fake NetBird management API, and fake Stripe API + HTTPS CONNECT proxy.
 * 4. Builds and launches real `fvpn-control -migrate-on-start` and `fvpn-worker` binaries.
 * 5. Creates member groups via `POST /api/v1/admin/groups`, enrolls two nodes via
 *    `POST /api/v1/admin/enrollments` + `POST /api/v1/agent/enroll` (`tw-tpe-01` amd64 active
 *    and `jp-tyo-01` arm64 offline secondary reporter), drives fleet updates via
 *    `POST /api/v1/admin/update-batches` -> `GET /api/v1/agent/desired-state` ->
 *    `POST /api/v1/agent/update-results` -> `POST /api/v1/agent/observations`, publishes the
 *    exit via `PUT /api/v1/admin/nodes/{id}/publication` with `group_ids`, and runs an mTLS
 *    agent observation loop so multi-step operations reach `succeeded` through real control/worker code.
 * 6. Serves the built Portal frontend on `PORTAL_DEV_PORT` (default 4173) and proxies `/api/v1` and `/auth`
 *    to `fvpn-control`.
 */

import { spawn, execFileSync } from 'node:child_process';
import crypto from 'node:crypto';
import fs from 'node:fs';
import http from 'node:http';
import https from 'node:https';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import pg from 'pg';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const PORTAL_DIR = path.resolve(__dirname, '..');
const REPO_ROOT = path.resolve(PORTAL_DIR, '../..');

const PORTAL_PORT = Number(process.env.PORTAL_DEV_PORT || 4173);
const ADMIN_DSN = process.env.FVPN_TEST_DATABASE_URL;

if (!ADMIN_DSN) {
  console.error('ERROR: FVPN_TEST_DATABASE_URL must be set to an admin PostgreSQL DSN.');
  process.exit(1);
}

const RELEASE_VECTOR_PATH = path.join(
  REPO_ROOT,
  'contracts/vectors/release-manifest-signing.json',
);
const RELEASE_VECTOR = JSON.parse(fs.readFileSync(RELEASE_VECTOR_PATH, 'utf8'));

const AT26_DIRECT_ROLLBACK_PATH = path.join(
  REPO_ROOT,
  'internal/fleetupdate/testdata/at26-direct-rollback.json',
);
const AT26_DIRECT_ROLLBACK = JSON.parse(
  fs.readFileSync(AT26_DIRECT_ROLLBACK_PATH, 'utf8'),
);

// Deterministic UUID constants for seeded test data
const PLAN_ID = 'b4fe029a-ecbf-4b1a-9ee7-de399791ae6e';
const RELEASE_100_ID = 'cdb25ffa-5395-400c-ad3e-b32899f14bb8';
const RELEASE_110_ID = RELEASE_VECTOR.envelope.signed.release_id; // 6d2797d0-b9a5-4724-96af-9b3039d351b1

function canonicalizeJson(val) {
  if (val === null || typeof val !== 'object') {
    return JSON.stringify(val);
  }
  if (Array.isArray(val)) {
    return `[${val.map((item) => canonicalizeJson(item)).join(',')}]`;
  }
  const keys = Object.keys(val).sort();
  const entries = [];
  for (const k of keys) {
    if (val[k] !== undefined) {
      entries.push(`${JSON.stringify(k)}:${canonicalizeJson(val[k])}`);
    }
  }
  return `{${entries.join(',')}}`;
}

function buildSignedRelease100Record() {
  const releaseSigningKey = crypto.createPrivateKey({
    key: {
      kty: 'OKP',
      crv: 'Ed25519',
      d: Buffer.from(RELEASE_VECTOR.seed_hex, 'hex').toString('base64url'),
      x: Buffer.from(RELEASE_VECTOR.public_key_b64, 'base64').toString('base64url'),
    },
    format: 'jwk',
  });
  const signed = {
    ...structuredClone(RELEASE_VECTOR.envelope.signed),
    release_id: RELEASE_100_ID,
    product_version: '1.0.0',
    release_sequence: '41',
    created_at: '2026-10-20T12:00:00Z',
    rollback: {
      ...structuredClone(RELEASE_VECTOR.envelope.signed.rollback),
      allowed_release_ids: [],
    },
  };
  const jcsBytes = Buffer.from(canonicalizeJson(signed), 'utf8');
  const signatureB64 = crypto.sign(null, jcsBytes, releaseSigningKey).toString('base64');
  const envelope = {
    signatures: [
      {
        algorithm: 'ed25519',
        key_id: RELEASE_VECTOR.key_id,
        signature_b64: signatureB64,
      },
    ],
    signed,
  };
  const envelopeJson = JSON.stringify(envelope);
  const envelopeSha256 = crypto
    .createHash('sha256')
    .update(Buffer.from(canonicalizeJson(envelope), 'utf8'))
    .digest('hex');
  return {
    envelope,
    envelopeJson,
    envelopeSha256,
    signatureB64,
  };
}

function getFreePort() {
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.listen(0, '127.0.0.1', () => {
      const addr = srv.address();
      const port = typeof addr === 'object' && addr ? addr.port : 0;
      srv.close((err) => (err ? reject(err) : resolve(port)));
    });
    srv.on('error', reject);
  });
}

function replaceDbName(dsn, newDbName) {
  const u = new URL(dsn);
  u.pathname = `/${newDbName}`;
  return u.toString();
}

function generateCertsAndVault(tmpDir) {
  const caKeyPath = path.join(tmpDir, 'ca.key');
  const caCertPath = path.join(tmpDir, 'ca.pem');
  execFileSync('openssl', [
    'req',
    '-x509',
    '-newkey',
    'ec',
    '-pkeyopt',
    'ec_paramgen_curve:P-256',
    '-days',
    '7',
    '-nodes',
    '-keyout',
    caKeyPath,
    '-out',
    caCertPath,
    '-subj',
    '/CN=fvpn-w3-portal-live-ca',
  ]);

  const serverKeyPath = path.join(tmpDir, 'server.key');
  const serverCsrPath = path.join(tmpDir, 'server.csr');
  const serverCertPath = path.join(tmpDir, 'server.pem');
  const sanCfgPath = path.join(tmpDir, 'san.cnf');
  fs.writeFileSync(
    sanCfgPath,
    [
      '[req]',
      'distinguished_name = dn',
      'req_extensions = v3_req',
      'prompt = no',
      '[dn]',
      'CN = 127.0.0.1',
      '[v3_req]',
      'basicConstraints = CA:FALSE',
      'keyUsage = digitalSignature, keyEncipherment',
      'extendedKeyUsage = serverAuth, clientAuth',
      'subjectAltName = IP:127.0.0.1,DNS:localhost,DNS:api.stripe.com,DNS:portal.example.invalid,DNS:releases.example.invalid',
    ].join('\n'),
  );
  execFileSync('openssl', [
    'req',
    '-newkey',
    'ec',
    '-pkeyopt',
    'ec_paramgen_curve:P-256',
    '-nodes',
    '-keyout',
    serverKeyPath,
    '-out',
    serverCsrPath,
    '-config',
    sanCfgPath,
  ]);
  execFileSync('openssl', [
    'x509',
    '-req',
    '-in',
    serverCsrPath,
    '-CA',
    caCertPath,
    '-CAkey',
    caKeyPath,
    '-CAcreateserial',
    '-out',
    serverCertPath,
    '-days',
    '7',
    '-extensions',
    'v3_req',
    '-extfile',
    sanCfgPath,
  ]);

  const nodeCaKeyPemPath = path.join(tmpDir, 'node-ca.key');
  const nodeCaCertPath = path.join(tmpDir, 'node-ca.pem');
  const nodeCaCfgPath = path.join(tmpDir, 'node-ca.cnf');
  fs.writeFileSync(
    nodeCaCfgPath,
    [
      '[req]',
      'distinguished_name = dn',
      'x509_extensions = v3_ca',
      'prompt = no',
      '[dn]',
      'CN = Freedom VPN development CA (TEST ONLY)',
      '[v3_ca]',
      'basicConstraints = critical, CA:TRUE, pathlen:0',
      'keyUsage = critical, keyCertSign, cRLSign',
    ].join('\n'),
  );
  execFileSync('openssl', [
    'ecparam',
    '-name',
    'prime256v1',
    '-genkey',
    '-noout',
    '-out',
    nodeCaKeyPemPath,
  ]);
  execFileSync('openssl', [
    'req',
    '-new',
    '-x509',
    '-key',
    nodeCaKeyPemPath,
    '-out',
    nodeCaCertPath,
    '-days',
    '30',
    '-config',
    nodeCaCfgPath,
    '-extensions',
    'v3_ca',
  ]);

  const nodeCaPrivKey = crypto.createPrivateKey(
    fs.readFileSync(nodeCaKeyPemPath, 'utf8'),
  );
  const nodeCaPubKey = crypto.createPublicKey(nodeCaPrivKey);
  const nodeCaPkcs8Der = nodeCaPrivKey.export({ type: 'pkcs8', format: 'der' });
  const nodeCaSpkiDer = nodeCaPubKey.export({ type: 'spki', format: 'der' });

  const vaultDir = path.join(tmpDir, 'vault');
  fs.mkdirSync(vaultDir, { mode: 0o700 });
  fs.writeFileSync(path.join(vaultDir, 'lock'), '', { mode: 0o600 });

  const now = new Date();
  const notBefore = new Date(now.getTime() - 3600_000).toISOString();
  const notAfter = new Date(now.getTime() + 7 * 86400_000).toISOString();

  const caKeyId = crypto.randomUUID();
  fs.writeFileSync(path.join(vaultDir, `${caKeyId}.key`), nodeCaPkcs8Der, {
    mode: 0o600,
  });

  const bootKeyId = crypto.randomUUID();
  fs.writeFileSync(path.join(vaultDir, `${bootKeyId}.key`), crypto.randomBytes(32), {
    mode: 0o600,
  });

  function makeEd25519KeyRecord(purpose) {
    const id = crypto.randomUUID();
    const { publicKey, privateKey } = crypto.generateKeyPairSync('ed25519');
    const pkcs8Der = privateKey.export({ type: 'pkcs8', format: 'der' });
    const spkiDer = publicKey.export({ type: 'spki', format: 'der' });
    const rawPub = spkiDer.subarray(spkiDer.length - 32);
    fs.writeFileSync(path.join(vaultDir, `${id}.key`), pkcs8Der, { mode: 0o600 });
    return {
      id,
      purpose,
      public: rawPub.toString('base64'),
      state: 'active',
      not_before: notBefore,
      not_after: notAfter,
    };
  }

  const ledgerState = {
    revision: '7',
    release_revision: '1',
    keys: [
      {
        id: caKeyId,
        purpose: 'node-ca',
        public: nodeCaSpkiDer.toString('base64'),
        state: 'active',
        not_before: notBefore,
        not_after: notAfter,
      },
      {
        id: bootKeyId,
        purpose: 'bootstrap-secret',
        state: 'active',
        not_before: notBefore,
        not_after: notAfter,
      },
      makeEd25519KeyRecord('desired-state'),
      makeEd25519KeyRecord('lease'),
      makeEd25519KeyRecord('recovery'),
      makeEd25519KeyRecord('ledger-epoch'),
      {
        id: RELEASE_VECTOR.key_id,
        purpose: 'release',
        public: RELEASE_VECTOR.public_key_b64,
        state: 'active',
        not_before: notBefore,
        not_after: notAfter,
      },
    ],
  };

  fs.writeFileSync(
    path.join(vaultDir, 'state.json'),
    JSON.stringify(ledgerState, null, 2),
    { mode: 0o600 },
  );

  return {
    caCertPath,
    serverKeyPath,
    serverCertPath,
    nodeCaCertPath,
    vaultDir,
    caKeyId,
  };
}

function createMockOidcServer(tlsOptions, oidcPort, portalBaseUrl) {
  const { publicKey, privateKey } = crypto.generateKeyPairSync('rsa', {
    modulusLength: 2048,
  });
  const pubJwk = publicKey.export({ format: 'jwk' });
  const keyId = 'oidc-test-kid-1';
  const issuer = `https://127.0.0.1:${oidcPort}`;

  const authCodes = new Map();
  let nextLoginPersona = {
    role: 'member',
    sub: 'oidc-sub-member-default',
    name: 'Portal Member',
  };

  const server = https.createServer(tlsOptions, (req, res) => {
    const reqUrl = new URL(req.url || '/', issuer);

    if (req.method === 'GET' && reqUrl.pathname === '/.well-known/openid-configuration') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(
        JSON.stringify({
          issuer,
          authorization_endpoint: `${issuer}/authorize`,
          token_endpoint: `${issuer}/token`,
          jwks_uri: `${issuer}/jwks`,
          response_types_supported: ['code'],
          subject_types_supported: ['public'],
          id_token_signing_alg_values_supported: ['RS256'],
        }),
      );
      return;
    }

    if (req.method === 'GET' && reqUrl.pathname === '/jwks') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(
        JSON.stringify({
          keys: [
            {
              ...pubJwk,
              kid: keyId,
              alg: 'RS256',
              use: 'sig',
            },
          ],
        }),
      );
      return;
    }

    if (req.method === 'GET' && reqUrl.pathname === '/authorize') {
      const state = reqUrl.searchParams.get('state') || '';
      const nonce = reqUrl.searchParams.get('nonce') || '';
      const codeChallenge = reqUrl.searchParams.get('code_challenge') || '';
      const code = `code_${crypto.randomUUID()}`;
      authCodes.set(code, {
        nonce,
        codeChallenge,
        persona: { ...nextLoginPersona },
      });
      const callbackUrl = `${portalBaseUrl}/auth/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`;
      res.writeHead(302, { Location: callbackUrl });
      res.end();
      return;
    }

    if (req.method === 'POST' && reqUrl.pathname === '/token') {
      let body = '';
      req.on('data', (chunk) => {
        body += chunk;
      });
      req.on('end', () => {
        const params = new URLSearchParams(body);
        const code = params.get('code') || '';
        const verifier = params.get('code_verifier') || '';
        const record = authCodes.get(code);
        authCodes.delete(code);

        if (!record) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ error: 'invalid_grant' }));
          return;
        }

        if (record.codeChallenge) {
          const expectedChallenge = crypto
            .createHash('sha256')
            .update(verifier)
            .digest('base64url');
          if (expectedChallenge !== record.codeChallenge) {
            res.writeHead(400, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ error: 'invalid_grant' }));
            return;
          }
        }

        const nowSec = Math.floor(Date.now() / 1000);
        const header = { alg: 'RS256', typ: 'JWT', kid: keyId };
        const payload = {
          iss: issuer,
          sub: record.persona.sub,
          aud: 'portal-web',
          nonce: record.nonce,
          iat: nowSec,
          exp: nowSec + 3600,
          auth_time: nowSec,
          name: record.persona.name,
          amr: ['mfa'],
          roles: record.persona.role === 'admin' ? ['admin'] : ['member'],
        };

        const encodedHeader = Buffer.from(JSON.stringify(header)).toString('base64url');
        const encodedPayload = Buffer.from(JSON.stringify(payload)).toString('base64url');
        const signingInput = `${encodedHeader}.${encodedPayload}`;
        const signature = crypto
          .sign('RSA-SHA256', Buffer.from(signingInput), privateKey)
          .toString('base64url');
        const idToken = `${signingInput}.${signature}`;

        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(
          JSON.stringify({
            access_token: `at_${crypto.randomUUID()}`,
            token_type: 'Bearer',
            expires_in: 3600,
            id_token: idToken,
          }),
        );
      });
      return;
    }

    res.writeHead(404);
    res.end();
  });

  return {
    server,
    issuer,
    setNextLoginPersona(persona) {
      nextLoginPersona = persona;
    },
  };
}

function createFakeNetBirdServer(tlsOptions) {
  const templatesDir = path.join(REPO_ROOT, 'internal/netbird/testdata');
  const templates = {
    groups: JSON.parse(fs.readFileSync(path.join(templatesDir, 'group.json'), 'utf8')),
    routes: JSON.parse(fs.readFileSync(path.join(templatesDir, 'route.json'), 'utf8')),
    peers: JSON.parse(fs.readFileSync(path.join(templatesDir, 'peer.json'), 'utf8')),
    policies: JSON.parse(fs.readFileSync(path.join(templatesDir, 'policy.json'), 'utf8')),
    'dns/nameservers': JSON.parse(
      fs.readFileSync(path.join(templatesDir, 'nameserver.json'), 'utf8'),
    ),
  };

  const objects = {
    groups: new Map([
      ['TEST-all', { id: 'TEST-all', name: 'All', peers: [], resources: [] }],
    ]),
    routes: new Map(),
    policies: new Map(),
    peers: new Map([
      [
        'nb-peer-node-1',
        {
          id: 'nb-peer-node-1',
          name: 'tw-tpe-01',
          user_id: '',
          ephemeral: false,
          ip: '100.64.0.10',
        },
      ],
      [
        'nb-peer-node-2',
        {
          id: 'nb-peer-node-2',
          name: 'jp-tyo-01',
          user_id: '',
          ephemeral: false,
          ip: '100.64.0.11',
        },
      ],
      [
        'nb-peer-node-3',
        {
          id: 'nb-peer-node-3',
          name: 'tw-tpe-02',
          user_id: '',
          ephemeral: false,
          ip: '100.64.0.12',
        },
      ],
      [
        'nb-peer-dev-1',
        {
          id: 'nb-peer-dev-1',
          name: 'Member MacBook Pro',
          user_id: 'oidc-sub-member-1',
          ephemeral: false,
          ip: '100.64.0.21',
        },
      ],
      [
        'nb-peer-dev-2',
        {
          id: 'nb-peer-dev-2',
          name: 'Member iPhone 16',
          user_id: 'oidc-sub-member-1',
          ephemeral: false,
          ip: '100.64.0.22',
        },
      ],
      [
        'nb-peer-dev-3',
        {
          id: 'nb-peer-dev-3',
          name: 'Member Linux Workstation',
          user_id: 'oidc-sub-member-1',
          ephemeral: false,
          ip: '100.64.0.23',
        },
      ],
      [
        'nb-peer-service-neg',
        {
          id: 'nb-peer-service-neg',
          name: 'Service User Peer (Skipped)',
          user_id: 'nb-service-user-1',
          ephemeral: false,
          ip: '100.64.0.91',
        },
      ],
      [
        'nb-peer-blocked-neg',
        {
          id: 'nb-peer-blocked-neg',
          name: 'Blocked User Peer (Skipped)',
          user_id: 'nb-blocked-user-1',
          ephemeral: false,
          ip: '100.64.0.92',
        },
      ],
    ]),
    'setup-keys': new Map(),
    users: new Map([
      [
        'oidc-sub-member-1',
        {
          id: 'oidc-sub-member-1',
          auto_groups: [],
          is_service_user: false,
          is_blocked: false,
        },
      ],
      [
        'nb-service-user-1',
        {
          id: 'nb-service-user-1',
          auto_groups: [],
          is_service_user: true,
          is_blocked: false,
        },
      ],
      [
        'nb-blocked-user-1',
        {
          id: 'nb-blocked-user-1',
          auto_groups: [],
          is_service_user: false,
          is_blocked: true,
        },
      ],
    ]),
    'dns/nameservers': new Map(),
  };

  let serial = 1;
  let nextSetupKeyPeer = { id: 'nb-peer-node-1', name: 'tw-tpe-01' };

  function extractIds(arr) {
    if (!Array.isArray(arr)) return [];
    return arr
      .map((item) => (typeof item === 'string' ? item : item?.id))
      .filter(Boolean);
  }

  const server = https.createServer(tlsOptions, (req, res) => {
    const reqUrl = new URL(req.url || '/', 'https://127.0.0.1');
    const relPath = reqUrl.pathname.replace(/^\/api\//, '');
    const parts = relPath.split('/').filter(Boolean);
    let kind = parts[0] || '';
    let id = parts[1] || '';
    if (kind === 'dns' && parts[1]) {
      kind = `dns/${parts[1]}`;
      id = parts[2] || '';
    }

    const bucket = objects[kind];
    if (!bucket) {
      res.writeHead(404);
      res.end();
      return;
    }

    const sendJson = (obj) => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(obj));
    };

    if (req.method === 'GET') {
      if (id) {
        const item = bucket.get(id);
        if (!item) {
          res.writeHead(404);
          res.end();
          return;
        }
        sendJson(item);
      } else {
        sendJson(Array.from(bucket.values()));
      }
      return;
    }

    if (req.method === 'DELETE') {
      bucket.delete(id);
      sendJson({});
      return;
    }

    let rawBody = '';
    req.on('data', (c) => {
      rawBody += c;
    });
    req.on('end', () => {
      let bodyObj = {};
      try {
        bodyObj = JSON.parse(rawBody || '{}');
      } catch {
        res.writeHead(400);
        res.end();
        return;
      }

      if (req.method === 'POST') {
        serial += 1;
        id = `nb-obj-${serial}`;
      } else if (!bucket.has(id)) {
        res.writeHead(404);
        res.end();
        return;
      }

      const reply = {
        ...(kind === 'setup-keys' && req.method === 'PUT' ? bucket.get(id) || {} : {}),
        ...(templates[kind] || {}),
        ...bodyObj,
        id,
      };

      if (kind === 'groups') {
        const peerIds = extractIds(bodyObj.peers);
        reply.peers = peerIds.map((pid) => ({ id: pid, name: pid }));
        reply.peers_count = reply.peers.length;
      } else if (kind === 'policies') {
        const rules = Array.isArray(bodyObj.rules) ? bodyObj.rules : [];
        reply.rules = rules.map((r) => ({
          ...r,
          id: 'TEST-rule',
          sources: extractIds(r.sources).map((gid) => ({
            id: gid,
            name: 'group',
            peers_count: 0,
            resources_count: 0,
          })),
          destinations: extractIds(r.destinations).map((gid) => ({
            id: gid,
            name: 'group',
            peers_count: 0,
            resources_count: 0,
          })),
        }));
      } else if (kind === 'setup-keys') {
        if (req.method === 'POST') {
          if (nextSetupKeyPeer) {
            const consumedPeer = nextSetupKeyPeer;
            nextSetupKeyPeer = null;
            reply.used_times = 1;
            reply.revoked = false;
            reply.state = 'overused';
            for (const gid of extractIds(reply.auto_groups)) {
              const grp = objects.groups.get(gid);
              if (grp) {
                grp.peers = [consumedPeer];
                grp.peers_count = 1;
              }
            }
          } else {
            reply.used_times = 0;
            reply.revoked = false;
            reply.state = 'valid';
          }
        }
        reply.key = `nb-setup-secret-${serial}`;
        reply.expires = new Date(Date.now() + 86400_000).toISOString();
      }

      const stored = JSON.parse(JSON.stringify(reply));
      if (kind === 'setup-keys') {
        delete stored.key;
      }
      bucket.set(id, stored);
      sendJson(reply);
    });
  });

  return {
    server,
    objects,
    setNextSetupKeyPeer: (peer) => {
      nextSetupKeyPeer = peer;
    },
  };
}

function createFakeStripeAndProxy(tlsOptions, webhookSecret) {
  const checkoutSessions = new Map();
  const subscriptions = new Map();
  let latestCheckoutId = null;

  const stripeServer = https.createServer(tlsOptions, (req, res) => {
    const reqUrl = new URL(req.url || '/', 'https://api.stripe.com');
    const sendJson = (obj, status = 200) => {
      res.writeHead(status, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(obj));
    };

    if (req.method === 'POST' && reqUrl.pathname === '/v1/checkout/sessions') {
      let body = '';
      req.on('data', (c) => {
        body += c;
      });
      req.on('end', () => {
        const params = new URLSearchParams(body);
        const fvpnCheckoutId =
          params.get('metadata[fvpn_checkout_id]') ||
          params.get('client_reference_id') ||
          crypto.randomUUID();
        const priceId = params.get('line_items[0][price]') || 'price_monthly_test';
        const expiresAt =
          Number(params.get('expires_at')) || Math.floor(Date.now() / 1000) + 3600;
        const sessionId = `cs_test_${fvpnCheckoutId.replace(/-/g, '')}`;
        const sessionObj = {
          id: sessionId,
          url: `https://checkout.stripe.example.invalid/pay/${sessionId}`,
          expires_at: expiresAt,
          fvpnCheckoutId,
          priceId,
        };
        checkoutSessions.set(sessionId, sessionObj);
        latestCheckoutId = fvpnCheckoutId;
        sendJson(sessionObj);
      });
      return;
    }

    if (req.method === 'GET' && reqUrl.pathname.startsWith('/v1/checkout/sessions/')) {
      const sessionId = decodeURIComponent(
        reqUrl.pathname.slice('/v1/checkout/sessions/'.length),
      );
      const sessionObj = checkoutSessions.get(sessionId);
      if (!sessionObj) {
        sendJson({ error: { message: 'Not found' } }, 404);
        return;
      }
      sendJson(sessionObj);
      return;
    }

    if (req.method === 'POST' && reqUrl.pathname === '/v1/billing_portal/sessions') {
      let body = '';
      req.on('data', (c) => {
        body += c;
      });
      req.on('end', () => {
        const portalId = `bps_test_${crypto.randomUUID().replace(/-/g, '').slice(0, 16)}`;
        sendJson({
          id: portalId,
          url: `https://billing.stripe.example.invalid/p/session/${portalId}`,
          created: Math.floor(Date.now() / 1000),
        });
      });
      return;
    }

    if (req.method === 'GET' && reqUrl.pathname.startsWith('/v1/subscriptions/')) {
      const subId = decodeURIComponent(
        reqUrl.pathname.slice('/v1/subscriptions/'.length),
      );
      const sub = subscriptions.get(subId);
      if (!sub) {
        sendJson({ error: { message: 'Subscription not found' } }, 404);
        return;
      }
      sendJson({
        id: sub.id,
        customer: sub.customerId,
        status: sub.status,
        cancel_at_period_end: sub.cancelAtPeriodEnd,
        livemode: false,
        metadata: { fvpn_checkout_id: sub.checkoutId },
        items: {
          has_more: false,
          data: [{ price: { id: sub.priceId }, quantity: 1 }],
        },
      });
      return;
    }

    if (req.method === 'GET' && reqUrl.pathname === '/v1/invoices') {
      const subId = reqUrl.searchParams.get('subscription') || '';
      const sub = subscriptions.get(subId);
      if (!sub) {
        sendJson({ has_more: false, data: [] });
        return;
      }
      sendJson({
        has_more: false,
        data: [
          {
            id: sub.invoiceId,
            customer: sub.customerId,
            status: 'paid',
            currency: 'twd',
            amount_paid: 24000,
            parent: { subscription_details: { subscription: sub.id } },
          },
        ],
      });
      return;
    }

    if (
      req.method === 'GET' &&
      reqUrl.pathname.startsWith('/v1/invoices/') &&
      reqUrl.pathname.endsWith('/lines')
    ) {
      const invoiceId = reqUrl.pathname.split('/')[3];
      const sub = Array.from(subscriptions.values()).find(
        (s) => s.invoiceId === invoiceId,
      );
      if (!sub) {
        sendJson({ has_more: false, data: [] });
        return;
      }
      sendJson({
        has_more: false,
        data: [
          {
            id: `il_${invoiceId}`,
            amount: 24000,
            quantity: 1,
            period: { start: sub.periodStart, end: sub.periodEnd },
            pricing: { price_details: { price: sub.priceId } },
            parent: {
              type: 'subscription_item_details',
              subscription_item_details: {
                subscription: sub.id,
                proration: false,
              },
            },
          },
        ],
      });
      return;
    }

    if (req.method === 'GET' && reqUrl.pathname === '/v1/invoice_payments') {
      const invoiceId = reqUrl.searchParams.get('invoice');
      const paymentIntent = reqUrl.searchParams.get('payment[payment_intent]');
      const sub = Array.from(subscriptions.values()).find(
        (s) => s.invoiceId === invoiceId || s.paymentIntentId === paymentIntent,
      );
      if (!sub) {
        sendJson({ has_more: false, data: [] });
        return;
      }
      sendJson({
        has_more: false,
        data: [
          {
            id: `inpay_${sub.invoiceId}`,
            status: 'paid',
            invoice: sub.invoiceId,
            payment: {
              type: 'payment_intent',
              payment_intent: sub.paymentIntentId,
            },
          },
        ],
      });
      return;
    }

    if (req.method === 'GET' && reqUrl.pathname === '/v1/refunds') {
      sendJson({ has_more: false, data: [] });
      return;
    }

    sendJson({ error: { message: 'Unhandled Stripe route' } }, 404);
  });

  const proxyServer = http.createServer((_req, res) => {
    res.writeHead(400);
    res.end();
  });

  proxyServer.on('connect', (_req, clientSocket, head) => {
    const stripeAddr = stripeServer.address();
    const stripePort = typeof stripeAddr === 'object' && stripeAddr ? stripeAddr.port : 0;
    const serverSocket = net.connect(stripePort, '127.0.0.1', () => {
      clientSocket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
      if (head && head.length > 0) {
        serverSocket.write(head);
      }
      serverSocket.pipe(clientSocket);
      clientSocket.pipe(serverSocket);
    });
    serverSocket.on('error', () => clientSocket.destroy());
    clientSocket.on('error', () => serverSocket.destroy());
  });

  function buildSignedWebhookEvent(checkoutIdOpt) {
    const checkoutId = checkoutIdOpt || latestCheckoutId;
    if (!checkoutId) {
      throw new Error('No checkout session created yet');
    }
    const cleanId = checkoutId.replace(/-/g, '').slice(0, 16);
    const subId = `sub_test_${cleanId}`;
    const cusId = `cus_test_${cleanId}`;
    const invId = `in_test_${cleanId}`;
    const piId = `pi_test_${cleanId}`;
    const nowSec = Math.floor(Date.now() / 1000);
    subscriptions.set(subId, {
      id: subId,
      customerId: cusId,
      checkoutId,
      priceId: 'price_monthly_test',
      status: 'active',
      cancelAtPeriodEnd: false,
      invoiceId: invId,
      paymentIntentId: piId,
      periodStart: nowSec - 60,
      periodEnd: nowSec + 28 * 86400,
    });

    const eventPayload = JSON.stringify({
      id: `evt_test_${crypto.randomUUID().replace(/-/g, '').slice(0, 18)}`,
      type: 'checkout.session.completed',
      api_version: '2026-09-30.endive',
      livemode: false,
      data: {
        object: {
          id: `cs_test_${checkoutId.replace(/-/g, '')}`,
          customer: cusId,
          subscription: subId,
          client_reference_id: checkoutId,
          metadata: {
            fvpn_checkout_id: checkoutId,
          },
        },
      },
    });
    const ts = Math.floor(Date.now() / 1000);
    const sig = crypto
      .createHmac('sha256', Buffer.from(webhookSecret, 'utf8'))
      .update(`${ts}.${eventPayload}`)
      .digest('hex');
    return {
      rawBody: eventPayload,
      signatureHeader: `t=${ts},v1=${sig}`,
      subscriptionId: subId,
    };
  }

  return {
    stripeServer,
    proxyServer,
    buildSignedWebhookEvent,
    getLatestCheckoutId: () => latestCheckoutId,
  };
}

async function httpsRequest(urlStr, options = {}) {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    const req = https.request(
      {
        hostname: u.hostname,
        port: u.port,
        path: u.pathname + u.search,
        method: options.method || 'GET',
        headers: options.headers || {},
        ca: options.ca,
        cert: options.cert,
        key: options.key,
        rejectUnauthorized: false,
      },
      (res) => {
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () => {
          const text = Buffer.concat(chunks).toString('utf8');
          resolve({
            status: res.statusCode || 0,
            headers: res.headers,
            text,
          });
        });
      },
    );
    req.on('error', reject);
    if (options.body) {
      req.write(options.body);
    }
    req.end();
  });
}

async function performOidcLogin(controlBaseUrl, oidcController, persona, caPem) {
  oidcController.setNextLoginPersona(persona);
  const loginRes = await httpsRequest(`${controlBaseUrl}/auth/login`, {
    method: 'GET',
    ca: caPem,
  });
  if (loginRes.status !== 302) {
    throw new Error(
      `Expected 302 from /auth/login, got ${loginRes.status}: ${loginRes.text}`,
    );
  }
  const setCookies = loginRes.headers['set-cookie'] || [];
  const flowCookie = setCookies
    .map((c) => c.split(';')[0])
    .find((c) => c.startsWith('__Host-fvpn-login='));
  const authUrl = loginRes.headers.location;
  if (!flowCookie || !authUrl) {
    throw new Error('Missing flow cookie or location from /auth/login');
  }

  const authorizeRes = await httpsRequest(authUrl, {
    method: 'GET',
    ca: caPem,
  });
  if (authorizeRes.status !== 302 || !authorizeRes.headers.location) {
    throw new Error(`Expected 302 from OIDC /authorize, got ${authorizeRes.status}`);
  }

  const cbUrl = new URL(authorizeRes.headers.location);
  const callbackRes = await httpsRequest(
    `${controlBaseUrl}${cbUrl.pathname}${cbUrl.search}`,
    {
      method: 'GET',
      headers: {
        Cookie: flowCookie,
      },
      ca: caPem,
    },
  );
  if (callbackRes.status !== 303) {
    throw new Error(
      `Expected 303 from /auth/callback, got ${callbackRes.status}: ${callbackRes.text}`,
    );
  }
  const sessionCookie = (callbackRes.headers['set-cookie'] || [])
    .map((c) => c.split(';')[0])
    .find(
      (c) =>
        c.startsWith('__Host-fvpn-session=') ||
        c.startsWith('__Host-fvpn-admin=') ||
        c.startsWith('__Host-fvpn-member='),
    );
  if (!sessionCookie) {
    throw new Error('Missing session cookie from /auth/callback');
  }

  const sessionRes = await httpsRequest(`${controlBaseUrl}/auth/session`, {
    method: 'GET',
    headers: { Cookie: sessionCookie },
    ca: caPem,
  });
  const sessionJson = JSON.parse(sessionRes.text);
  const csrfToken = sessionRes.headers['x-csrf-token'];
  return {
    sessionCookie,
    csrfToken,
    session: sessionJson,
    sub: persona.sub,
  };
}

async function createGroupViaApi(controlBaseUrl, adminAuth, caPem, code, name) {
  const res = await httpsRequest(`${controlBaseUrl}/api/v1/admin/groups`, {
    method: 'POST',
    headers: {
      Cookie: adminAuth.sessionCookie,
      'X-CSRF-Token': adminAuth.csrfToken,
      Origin: 'https://portal.example.invalid',
      'Idempotency-Key': crypto.randomUUID(),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ code, name }),
    ca: caPem,
  });
  if (res.status !== 201) {
    throw new Error(
      `POST /api/v1/admin/groups (${code}) failed (${res.status}): ${res.text}`,
    );
  }
  return JSON.parse(res.text);
}

async function enrollLiveNode(
  controlBaseUrl,
  agentBaseUrl,
  adminAuth,
  caPem,
  tmpDir,
  options = {},
) {
  const {
    name = 'tw-tpe-01',
    region = 'tw-tpe',
    arch = 'amd64',
    peerId = 'nb-peer-node-1',
    memberGroupIds = [],
    notes = 'Primary Taipei exit node',
    filePrefix = 'enrolled-node-1',
  } = options;

  const createRes = await httpsRequest(`${controlBaseUrl}/api/v1/admin/enrollments`, {
    method: 'POST',
    headers: {
      Cookie: adminAuth.sessionCookie,
      'X-CSRF-Token': adminAuth.csrfToken,
      Origin: 'https://portal.example.invalid',
      'Idempotency-Key': crypto.randomUUID(),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      delivery: 'preinstalled',
      expires_in_seconds: 3600,
      name,
      region,
      member_group_ids: memberGroupIds,
      notes,
    }),
    ca: caPem,
  });
  if (createRes.status !== 201) {
    throw new Error(`CreateEnrollment failed (${createRes.status}): ${createRes.text}`);
  }
  const created = JSON.parse(createRes.text);
  const token = created.token;
  const nodeId = created.enrollment.node_id;

  const nodeKeyPath = path.join(tmpDir, `${filePrefix}.key`);
  const nodeCsrPath = path.join(tmpDir, `${filePrefix}.csr`);
  execFileSync('openssl', [
    'ecparam',
    '-name',
    'prime256v1',
    '-genkey',
    '-noout',
    '-out',
    nodeKeyPath,
  ]);
  execFileSync('openssl', [
    'req',
    '-new',
    '-key',
    nodeKeyPath,
    '-out',
    nodeCsrPath,
    '-subj',
    '/',
  ]);
  const csrPem = fs.readFileSync(nodeCsrPath, 'utf8');
  const nodeKeyPem = fs.readFileSync(nodeKeyPath, 'utf8');

  const enrollmentRequestId = crypto.randomUUID();
  const nodeIncarnationId = crypto.randomUUID();
  const agentRelease = '1.1.0';
  const platform = { arch, os: 'ubuntu', os_version: '24.04' };
  const jcsProofInput = JSON.stringify({
    agent_release: agentRelease,
    csr_pem: csrPem,
    enrollment_request_id: enrollmentRequestId,
    node_incarnation_id: nodeIncarnationId,
    platform,
  });
  const proofB64 = crypto
    .sign('sha256', Buffer.from(jcsProofInput, 'utf8'), nodeKeyPem)
    .toString('base64');

  const enrollRes = await httpsRequest(`${controlBaseUrl}/api/v1/agent/enroll`, {
    method: 'POST',
    headers: {
      Authorization: `Enrollment ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      enrollment_request_id: enrollmentRequestId,
      node_incarnation_id: nodeIncarnationId,
      csr_pem: csrPem,
      platform,
      agent_release: agentRelease,
      proof: proofB64,
    }),
    ca: caPem,
  });
  if (enrollRes.status !== 201) {
    throw new Error(`EnrollNode failed (${enrollRes.status}): ${enrollRes.text}`);
  }
  const enrolled = JSON.parse(enrollRes.text);
  const clientCertPem = enrolled.certificate_pem;
  const incarnationId = enrolled.node_incarnation_id;

  const bootId = crypto.randomUUID();
  const ledgerEpoch = crypto.randomUUID();
  const meterEpochId = crypto.randomUUID();
  const leaseId = crypto.randomUUID();
  let lastObservedMs = 0;
  let currentReleaseId = RELEASE_100_ID;
  let lastGoodReleaseId = RELEASE_100_ID;
  let memberForwardingEnabled = true;
  let activeUpdateObs = {
    job_id: null,
    generation: null,
    state: null,
    stage: null,
  };

  function computeUpdateResultPayloadHash(bodyWithoutHash) {
    return crypto
      .createHash('sha256')
      .update(Buffer.from(canonicalizeJson(bodyWithoutHash), 'utf8'))
      .digest('hex');
  }

  async function submitObservation(fallbackGen = '1') {
    const desiredEnvelope = await fetchDesiredState();
    const desired = desiredEnvelope?.signed ?? null;
    const appliedGen = String(desired?.config_generation ?? fallbackGen);
    const nowMs = Math.max(Date.now(), lastObservedMs + 1);
    lastObservedMs = nowMs;
    const nowIso = new Date(nowMs).toISOString();
    const leaseExpiresIso = new Date(nowMs + 3600_000).toISOString();
    const obsBody = {
      report_id: crypto.randomUUID(),
      node_id: nodeId,
      node_incarnation_id: incarnationId,
      boot_id: bootId,
      observed_at: nowIso,
      agent_version: '1.1.0',
      current_release_id: currentReleaseId,
      last_good_release_id: lastGoodReleaseId,
      applied_config_generation: appliedGen,
      applied_policy_refs: {
        authorization_generation: String(
          desired?.policy_refs?.authorization_generation ?? '1',
        ),
        fup_generation: String(desired?.policy_refs?.fup_generation ?? '1'),
        dns_generation: String(desired?.policy_refs?.dns_generation ?? '1'),
      },
      applied_binding_revision: String(desired?.binding_revision ?? '1'),
      provisioning_state: 'ready',
      safety_status: 'passed',
      data_health: 'healthy',
      member_forwarding_enabled: memberForwardingEnabled,
      forwarding_block_reasons: [],
      lease: {
        lease_id: leaseId,
        expires_at: leaseExpiresIso,
      },
      netbird: {
        peer_id: peerId,
        version: '0.76.2',
        backend: 'kernel',
      },
      usage: {
        ledger_epoch: ledgerEpoch,
        meter_epoch_id: meterEpochId,
        last_durable_event_seq: '10',
        central_acked_through: '10',
        blocked_seq: null,
        unacked_event_count: 0,
        unacked_payload_bytes: '0',
        oldest_unacked_at: null,
        acked_archive_bytes: '4096',
      },
      update: activeUpdateObs,
      checks: [
        {
          name: 'internet_only',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
          detail: 'inet fvpn table verified; RFC1918 egress blocked',
        },
        {
          name: 'meter_path',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
          detail: 'forward_admission_v1 counters active on wg-nb0',
        },
        {
          name: 'tunnel_probe',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
          detail: 'NetBird WireGuard peer handshake verified',
        },
        {
          name: 'lease',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
          detail: 'serving lease valid and unexpired',
        },
        {
          name: 'qos',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
          detail: 'tc HTB + fq_codel classes installed',
        },
        {
          name: 'source_binding',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
          detail: 'member peer source IP binding verified',
        },
      ],
      errors: [],
      dns_policy_evidence: {
        document_id:
          desired?.dns?.resolver_document?.document_id ??
          '01900000-0000-7000-8000-000000000056',
        content_sha256:
          desired?.dns?.resolver_document?.content_sha256 ??
          '2bed2c3a6225090d221f86c8d597c2782a44149ad9bafa7f02783c4cbaf5f20d',
        generation: String(
          desired?.dns?.resolver_document?.generation ??
            desired?.policy_refs?.dns_generation ??
            '1',
        ),
        outcome: 'applied',
        applied_at: nowIso,
        feeds: [
          {
            feed_code: 'malware-domains-default',
            content_sha256:
              '7777777777777777777777777777777777777777777777777777777777777777',
            rule_count: 1420,
          },
        ],
        error_code: null,
      },
    };
    for (let attempt = 1; attempt <= 6; attempt += 1) {
      const attemptBody =
        attempt === 1
          ? obsBody
          : {
              ...obsBody,
              report_id: crypto.randomUUID(),
              observed_at: new Date().toISOString(),
            };
      const obsRes = await httpsRequest(`${agentBaseUrl}/api/v1/agent/observations`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(attemptBody),
        ca: caPem,
        cert: clientCertPem,
        key: nodeKeyPem,
      });
      if (obsRes.status === 200 || obsRes.status === 202) {
        return;
      }
      if (obsRes.status === 429 && attempt < 6) {
        const retryAfterHeader = obsRes.headers?.['retry-after'];
        const parsedSec = Number.parseInt(String(retryAfterHeader ?? '1'), 10);
        const waitMs = Math.min(
          5000,
          Math.max(200, (Number.isFinite(parsedSec) ? parsedSec : 1) * 1000),
        );
        await new Promise((r) => setTimeout(r, waitMs));
        continue;
      }
      throw new Error(
        `POST /api/v1/agent/observations failed (${obsRes.status}): ${obsRes.text}`,
      );
    }
  }

  async function fetchDesiredState() {
    const res = await httpsRequest(`${agentBaseUrl}/api/v1/agent/desired-state`, {
      method: 'GET',
      headers: { Accept: 'application/json' },
      ca: caPem,
      cert: clientCertPem,
      key: nodeKeyPem,
    });
    if (res.status !== 200) {
      throw new Error(
        `GET /api/v1/agent/desired-state failed (${res.status}): ${res.text}`,
      );
    }
    return JSON.parse(res.text);
  }

  async function submitUpdateResult(intent, compiledConfigGen) {
    const issuedMs = new Date(intent.issued_at).getTime();
    const startedIso = new Date(issuedMs + 1000).toISOString();
    const finishedIso = new Date(issuedMs + 2000).toISOString();
    const resultBodyWithoutHash = {
      result_id: crypto.randomUUID(),
      node_id: nodeId,
      incarnation_id: incarnationId,
      boot_id: bootId,
      batch_id: intent.batch_id,
      target_id: intent.target_id,
      job_id: intent.job_id,
      attempt: intent.attempt ?? 1,
      generation: String(intent.generation),
      state: 'succeeded',
      stage: null,
      requested_release_id: intent.release_id,
      previous_release_id: currentReleaseId,
      current_release_id: intent.release_id,
      manifest_sha256: intent.manifest_sha256,
      publication_intent: intent.publication_intent ?? 'published',
      member_forwarding_enabled: true,
      recovery_required: false,
      error_code: null,
      started_at: startedIso,
      finished_at: finishedIso,
      state_schema_version: 1,
      policy_generation: String(compiledConfigGen),
      health: {
        processes: 'passed',
        identity_and_state: 'passed',
        internet_only: 'passed',
        dns_policy: 'passed',
        tunnel_probe: 'passed',
      },
      usage_checkpoint: {
        last_durable_event_seq: '10',
        central_acked_through: '9',
        closed_meter_epoch_id: null,
        new_meter_epoch_id: null,
      },
    };
    const payloadHash = computeUpdateResultPayloadHash(resultBodyWithoutHash);
    const resultBody = {
      ...resultBodyWithoutHash,
      payload_hash: payloadHash,
    };
    const res = await httpsRequest(`${agentBaseUrl}/api/v1/agent/update-results`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(resultBody),
      ca: caPem,
      cert: clientCertPem,
      key: nodeKeyPem,
    });
    if (res.status !== 200 && res.status !== 202) {
      throw new Error(
        `POST /api/v1/agent/update-results failed (${res.status}): ${res.text}`,
      );
    }
    currentReleaseId = intent.release_id;
    lastGoodReleaseId = intent.release_id;
    memberForwardingEnabled = true;
    activeUpdateObs = {
      job_id: intent.job_id,
      generation: String(intent.generation),
      state: 'succeeded',
      stage: null,
    };
    return JSON.parse(res.text);
  }

  async function submitRolledBackUpdateResult(intent, compiledConfigGen) {
    const issuedMs = new Date(intent.issued_at).getTime();
    const startedIso = new Date(issuedMs + 1000).toISOString();
    const finishedIso = new Date(issuedMs + 2000).toISOString();
    const templateWithoutHash = structuredClone(AT26_DIRECT_ROLLBACK);
    delete templateWithoutHash.payload_hash;
    const resultBodyWithoutHash = {
      ...templateWithoutHash,
      result_id: crypto.randomUUID(),
      node_id: nodeId,
      incarnation_id: incarnationId,
      boot_id: bootId,
      batch_id: intent.batch_id,
      target_id: intent.target_id,
      job_id: intent.job_id,
      attempt: intent.attempt ?? 1,
      generation: String(intent.generation),
      requested_release_id: intent.release_id,
      previous_release_id: intent.release_id,
      current_release_id: RELEASE_100_ID,
      manifest_sha256: intent.manifest_sha256,
      publication_intent: intent.publication_intent ?? 'unpublished',
      started_at: startedIso,
      finished_at: finishedIso,
      policy_generation: String(compiledConfigGen),
    };
    const payloadHash = computeUpdateResultPayloadHash(resultBodyWithoutHash);
    const resultBody = {
      ...resultBodyWithoutHash,
      payload_hash: payloadHash,
    };
    const res = await httpsRequest(`${agentBaseUrl}/api/v1/agent/update-results`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(resultBody),
      ca: caPem,
      cert: clientCertPem,
      key: nodeKeyPem,
    });
    if (res.status !== 200 && res.status !== 202) {
      throw new Error(
        `POST /api/v1/agent/update-results (rolled_back) failed (${res.status}): ${res.text}`,
      );
    }
    currentReleaseId = RELEASE_100_ID;
    lastGoodReleaseId = RELEASE_100_ID;
    memberForwardingEnabled = Boolean(resultBodyWithoutHash.member_forwarding_enabled);
    activeUpdateObs = {
      job_id: intent.job_id,
      generation: String(intent.generation),
      state: 'rolled_back',
      stage: resultBodyWithoutHash.stage ?? 'report',
    };
    return JSON.parse(res.text);
  }

  return {
    nodeId,
    incarnationId,
    clientCertPem,
    nodeKeyPem,
    submitObservation,
    fetchDesiredState,
    submitUpdateResult,
    submitRolledBackUpdateResult,
  };
}

function lastLines(text, count = 50) {
  const lines = String(text || '')
    .trim()
    .split('\n')
    .filter(Boolean);
  return lines.slice(-count).join('\n') || '(empty)';
}

async function runSeedStep(stepName, fn) {
  try {
    if (process.env.PORTAL_DEV_STACK_INJECT_FAIL_STEP === stepName) {
      throw new Error(
        `Injected failure for diagnostics verification at step "${stepName}"`,
      );
    }
    return await fn();
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    throw new Error(`[seed step "${stepName}"] ${msg}`, { cause: err });
  }
}

async function patchMemberViaApi(controlBaseUrl, adminAuth, caPem, memberId, patchBody) {
  for (let attempt = 1; attempt <= 4; attempt += 1) {
    const getRes = await httpsRequest(
      `${controlBaseUrl}/api/v1/admin/members/${memberId}`,
      {
        method: 'GET',
        headers: {
          Cookie: adminAuth.sessionCookie,
          Accept: 'application/json',
        },
        ca: caPem,
      },
    );
    if (getRes.status !== 200) {
      throw new Error(
        `GET /api/v1/admin/members/${memberId} failed (${getRes.status}): ${getRes.text}`,
      );
    }
    const memberObj = JSON.parse(getRes.text);
    const etag = getRes.headers.etag || `"${memberObj.resource_version}"`;
    const patchRes = await httpsRequest(
      `${controlBaseUrl}/api/v1/admin/members/${memberId}`,
      {
        method: 'PATCH',
        headers: {
          Cookie: adminAuth.sessionCookie,
          'X-CSRF-Token': adminAuth.csrfToken,
          Origin: 'https://portal.example.invalid',
          'Idempotency-Key': crypto.randomUUID(),
          'If-Match': etag,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(patchBody),
        ca: caPem,
      },
    );
    if (patchRes.status === 200 || patchRes.status === 202) {
      return JSON.parse(patchRes.text);
    }
    if (patchRes.status === 412 && attempt < 4) {
      continue;
    }
    throw new Error(
      `PATCH /api/v1/admin/members/${memberId} failed (${patchRes.status}): ${patchRes.text}`,
    );
  }
  throw new Error(`PATCH /api/v1/admin/members/${memberId} exhausted ETag retries`);
}

async function grantMemberEntitlementViaApi(
  controlBaseUrl,
  adminAuth,
  caPem,
  memberId,
  planRevisionId,
  validFrom,
  validUntil,
  reason,
) {
  const grantRes = await httpsRequest(
    `${controlBaseUrl}/api/v1/admin/members/${memberId}/grants`,
    {
      method: 'POST',
      headers: {
        Cookie: adminAuth.sessionCookie,
        'X-CSRF-Token': adminAuth.csrfToken,
        Origin: 'https://portal.example.invalid',
        'Idempotency-Key': crypto.randomUUID(),
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        source: 'admin_grant',
        plan_revision_id: planRevisionId,
        valid_from: validFrom.toISOString(),
        valid_until: validUntil.toISOString(),
        reason,
      }),
      ca: caPem,
    },
  );
  if (grantRes.status !== 202) {
    throw new Error(
      `POST /api/v1/admin/members/${memberId}/grants failed (${grantRes.status}): ${grantRes.text}`,
    );
  }
  return JSON.parse(grantRes.text);
}

async function pollMemberEntitlementActive(pool, memberId, label) {
  let lastRow = null;
  for (let i = 0; i < 60; i += 1) {
    const res = await pool.query(
      `SELECT m.status AS member_status, e.state AS entitlement_state, e.device_limit::int AS device_limit
       FROM members m
       LEFT JOIN entitlements e ON e.member_id = m.id
       WHERE m.id = $1`,
      [memberId],
    );
    lastRow = res.rows[0] || null;
    if (
      lastRow &&
      lastRow.member_status === 'active' &&
      lastRow.entitlement_state === 'active' &&
      lastRow.device_limit === 3
    ) {
      return lastRow;
    }
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error(
    `Timed out waiting for ${label} (${memberId}) active entitlement projection; last=${JSON.stringify(lastRow)}`,
  );
}

async function pollMemberAuthorizedDevices(pool, memberId, expectedCount, label) {
  const deadline = Date.now() + 15_000;
  let lastIds = [];
  let lastBinding = null;
  while (Date.now() < deadline) {
    const bindRes = await pool.query(
      `SELECT external_user_id, mapping_proof, revision::text AS revision
       FROM netbird_user_bindings
       WHERE member_id = $1
         AND deployment_id = 'nb-dev'
         AND mapping_proof LIKE 'shared_issuer_public_subject:%'`,
      [memberId],
    );
    lastBinding = bindRes.rows[0] || null;

    const res = await pool.query(
      `SELECT d.id
       FROM member_devices d
       JOIN device_authorization_observations o ON o.device_id = d.id
       WHERE d.member_id = $1
         AND d.status = 'authorized'
         AND o.observed_authorized = true
       ORDER BY d.id`,
      [memberId],
    );
    lastIds = res.rows.map((r) => r.id);
    if (lastBinding && lastIds.length >= expectedCount) {
      return lastIds.slice(0, expectedCount);
    }
    await new Promise((r) => setTimeout(r, 200));
  }
  const diagRes = await pool.query(
    `SELECT d.id, d.peer_id, d.status, o.observed_authorized
     FROM member_devices d
     LEFT JOIN device_authorization_observations o ON o.device_id = d.id
     WHERE d.member_id = $1
     ORDER BY d.id`,
    [memberId],
  );
  throw new Error(
    `Timed out waiting for ${expectedCount} authorized+observed devices on ${label} (${memberId}); binding=${JSON.stringify(lastBinding)} current=${JSON.stringify(diagRes.rows)}`,
  );
}

const NODE_AFFECTING_OUTBOX_EVENTS = [
  'device.authorized',
  'device.revoked',
  'member.entitlement_changed',
  'member.suspended',
  'policy.activated',
  'node.peer.observe',
  'node.peer.discover',
  'node.provision.requested',
  'node.publication_changed',
  'node.updated',
  'fleet.update_created',
  'fleet.first_sync_created',
  'fleet.update_retry',
  'fleet.release_support_changed',
];

async function drainNodeAffectingOutbox(pool, label = 'outbox') {
  const deadline = Date.now() + 15_000;
  while (Date.now() < deadline) {
    const outboxRes = await pool.query(
      `SELECT EXISTS(
         SELECT 1 FROM outbox_events
         WHERE delivered_at IS NULL
           AND dead_lettered_at IS NULL
           AND event_type = ANY($1::text[])
       ) AS has_undelivered`,
      [NODE_AFFECTING_OUTBOX_EVENTS],
    );
    if (!outboxRes.rows[0]?.has_undelivered) {
      return;
    }
    await new Promise((r) => setTimeout(r, 50));
  }
  throw new Error(`Timed out draining node-affecting outbox events on ${label}`);
}

async function pollLiveNodeOnline(pool, liveNode, label = 'liveNodeOnline') {
  const deadline = Date.now() + 15_000;
  let lastRow = null;
  let lastSubmittedGen = null;
  let lastSubmittedAt = 0;
  while (Date.now() < deadline) {
    const res = await pool.query(
      `SELECT
         desired_config_generation::text AS desired_config_generation,
         applied_config_generation::text AS applied_config_generation,
         greatest(desired_config_generation, 1)::text AS target_gen,
         provisioning_state,
         safety_status,
         control_status,
         data_health,
         (applied_config_generation >= desired_config_generation) AS gen_caught_up,
         EXISTS(
           SELECT 1 FROM outbox_events
           WHERE delivered_at IS NULL
             AND dead_lettered_at IS NULL
             AND event_type = 'node.peer.observe'
         ) AS has_pending_observe
       FROM nodes
       WHERE id = $1`,
      [liveNode.nodeId],
    );
    lastRow = res.rows[0] || null;
    if (
      lastRow &&
      lastRow.provisioning_state === 'ready' &&
      lastRow.safety_status === 'passed' &&
      lastRow.control_status === 'online' &&
      lastRow.gen_caught_up === true &&
      !lastRow.has_pending_observe
    ) {
      return lastRow;
    }
    const targetGen = lastRow?.target_gen || '1';
    const now = Date.now();
    if (
      !lastRow?.gen_caught_up ||
      lastSubmittedGen !== targetGen ||
      (!lastRow?.has_pending_observe && now - lastSubmittedAt >= 1000)
    ) {
      await liveNode.submitObservation(targetGen);
      lastSubmittedGen = targetGen;
      lastSubmittedAt = Date.now();
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error(
    `Timed out waiting for live node online projection on ${label} (${liveNode.nodeId}); last=${JSON.stringify(lastRow)}`,
  );
}

async function pollLiveNodeReady(
  pool,
  liveNode,
  label = 'liveNode',
  expectedReleaseId = RELEASE_110_ID,
) {
  const deadline = Date.now() + 15_000;
  let lastRow = null;
  let lastSubmittedGen = null;
  let lastSubmittedAt = 0;
  while (Date.now() < deadline) {
    const res = await pool.query(
      `SELECT
         desired_config_generation::text AS desired_config_generation,
         applied_config_generation::text AS applied_config_generation,
         greatest(desired_config_generation, 1)::text AS target_gen,
         provisioning_state,
         safety_status,
         control_status,
         data_health,
         current_release_id,
         last_good_release_id,
         (applied_config_generation >= desired_config_generation) AS gen_caught_up,
         EXISTS(
           SELECT 1 FROM outbox_events
           WHERE delivered_at IS NULL
             AND dead_lettered_at IS NULL
             AND event_type = 'node.peer.observe'
         ) AS has_pending_observe
       FROM nodes
       WHERE id = $1`,
      [liveNode.nodeId],
    );
    lastRow = res.rows[0] || null;
    if (
      lastRow &&
      lastRow.provisioning_state === 'ready' &&
      lastRow.safety_status === 'passed' &&
      lastRow.control_status === 'online' &&
      lastRow.gen_caught_up === true &&
      lastRow.current_release_id === expectedReleaseId &&
      lastRow.last_good_release_id === expectedReleaseId &&
      !lastRow.has_pending_observe
    ) {
      return lastRow;
    }
    const targetGen = lastRow?.target_gen || '1';
    const now = Date.now();
    if (
      !lastRow?.gen_caught_up ||
      lastRow?.current_release_id !== expectedReleaseId ||
      lastRow?.last_good_release_id !== expectedReleaseId ||
      lastSubmittedGen !== targetGen ||
      (!lastRow?.has_pending_observe && now - lastSubmittedAt >= 1000)
    ) {
      await liveNode.submitObservation(targetGen);
      lastSubmittedGen = targetGen;
      lastSubmittedAt = Date.now();
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error(
    `Timed out waiting for live node observation projection on ${label} (${liveNode.nodeId}); last=${JSON.stringify(lastRow)}`,
  );
}

async function publishLiveNodeExitViaApi(
  controlBaseUrl,
  adminAuth,
  caPem,
  pool,
  liveNode,
  exitGroupId,
) {
  let op = null;
  for (let attempt = 1; attempt <= 4; attempt += 1) {
    const getRes = await httpsRequest(
      `${controlBaseUrl}/api/v1/admin/nodes/${liveNode.nodeId}`,
      {
        method: 'GET',
        headers: {
          Cookie: adminAuth.sessionCookie,
          Accept: 'application/json',
        },
        ca: caPem,
      },
    );
    if (getRes.status !== 200) {
      throw new Error(
        `GET /api/v1/admin/nodes/${liveNode.nodeId} failed (${getRes.status}): ${getRes.text}`,
      );
    }
    const nodeObj = JSON.parse(getRes.text);
    const etag = getRes.headers.etag || `"${nodeObj.resource_version}"`;
    const pubRes = await httpsRequest(
      `${controlBaseUrl}/api/v1/admin/nodes/${liveNode.nodeId}/publication`,
      {
        method: 'PUT',
        headers: {
          Cookie: adminAuth.sessionCookie,
          'X-CSRF-Token': adminAuth.csrfToken,
          Origin: 'https://portal.example.invalid',
          'Idempotency-Key': crypto.randomUUID(),
          'If-Match': etag,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          publication_intent: 'published',
          reason: 'Initial dev-stack exit publication',
          group_ids: [exitGroupId],
        }),
        ca: caPem,
      },
    );
    if (pubRes.status === 200 || pubRes.status === 202) {
      op = JSON.parse(pubRes.text);
      break;
    }
    if (pubRes.status === 412 && attempt < 4) {
      continue;
    }
    throw new Error(
      `PUT /api/v1/admin/nodes/${liveNode.nodeId}/publication failed (${pubRes.status}): ${pubRes.text}`,
    );
  }
  if (!op) {
    throw new Error(
      `PUT /api/v1/admin/nodes/${liveNode.nodeId}/publication exhausted ETag retries`,
    );
  }

  await liveNode.submitObservation();

  const deadline = Date.now() + 15_000;
  let lastRow = null;
  let lastSubmittedGen = null;
  let lastSubmittedAt = Date.now();
  while (Date.now() < deadline) {
    const res = await pool.query(
      `SELECT
         n.desired_config_generation::text AS desired_config_generation,
         n.applied_config_generation::text AS applied_config_generation,
         greatest(n.desired_config_generation, 1)::text AS target_gen,
         (n.applied_config_generation >= n.desired_config_generation) AS gen_caught_up,
         e.publication_intent,
         e.publication_observed,
         EXISTS(
           SELECT 1 FROM outbox_events
           WHERE delivered_at IS NULL
             AND dead_lettered_at IS NULL
             AND event_type = 'node.peer.observe'
         ) AS has_pending_observe
       FROM nodes n
       JOIN exits e ON e.node_id = n.id
       WHERE n.id = $1`,
      [liveNode.nodeId],
    );
    lastRow = res.rows[0] || null;
    if (
      lastRow &&
      lastRow.publication_intent === 'published' &&
      lastRow.publication_observed === 'active' &&
      lastRow.gen_caught_up === true &&
      !lastRow.has_pending_observe
    ) {
      return lastRow;
    }
    const targetGen = lastRow?.target_gen || '1';
    const now = Date.now();
    if (
      !lastRow?.gen_caught_up ||
      lastSubmittedGen !== targetGen ||
      (!lastRow?.has_pending_observe && now - lastSubmittedAt >= 1000)
    ) {
      await liveNode.submitObservation(targetGen);
      lastSubmittedGen = targetGen;
      lastSubmittedAt = Date.now();
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error(
    `Timed out waiting for live node exit publication active on ${liveNode.nodeId}; last=${JSON.stringify(lastRow)}`,
  );
}

async function seedDatabase(
  pool,
  adminId,
  adminAuth,
  controlBaseUrl,
  caPem,
  defaultMemberId,
  liveNode,
  secondaryNode,
  rollbackNode,
  exitGroupId,
  getWorkerLogs,
) {
  const enrolledNodeId = liveNode.nodeId;
  const secondaryNodeId = secondaryNode.nodeId;
  const rollbackNodeId = rollbackNode.nodeId;
  const now = new Date();
  // Valid < 30d grant window (14d + 1h < MaxGrantDuration 2592000s, D-050)
  const periodStart = new Date(now.getTime() - 3600_000);
  const periodEnd = new Date(now.getTime() + 14 * 86400_000);
  // UTC calendar month window for usage_rollups (§10.2)
  const monthStart = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));
  const monthEnd = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1));
  const currentMonth = `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, '0')}`;

  await runSeedStep('seed-releases-and-fleet-config', async () => {
    const r100 = buildSignedRelease100Record();
    await pool.query(
      `INSERT INTO releases(id, version, release_sequence, manifest_schema_version, manifest, manifest_hash, signature, signing_key_ref, component_versions)
       VALUES ($1, '1.0.0', 41, 1, $2::jsonb, $3, $4, $5, $6::jsonb)
       ON CONFLICT (id) DO NOTHING`,
      [
        RELEASE_100_ID,
        r100.envelopeJson,
        Buffer.from(r100.envelopeSha256, 'hex'),
        Buffer.from(r100.signatureB64, 'base64'),
        RELEASE_VECTOR.key_id,
        JSON.stringify(r100.envelope.signed.components),
      ],
    );
    for (const art of r100.envelope.signed.artifacts) {
      await pool.query(
        `INSERT INTO artifacts(release_id, platform, arch, digest, size_bytes, installed_size_bytes, storage_ref)
         VALUES ($1, 'ubuntu/24.04', $2, $3, $4::numeric, $5::numeric, $6)
         ON CONFLICT (release_id, platform, arch) DO NOTHING`,
        [
          RELEASE_100_ID,
          art.architecture,
          Buffer.from(art.sha256, 'hex'),
          art.size_bytes,
          art.installed_size_bytes,
          art.url,
        ],
      );
    }
    await pool.query(
      `INSERT INTO release_verifications(release_id, manifest_url, state, verified_at)
       VALUES ($1, 'https://releases.example.invalid/r41/manifest.json', 'verified', clock_timestamp())
       ON CONFLICT (release_id) DO NOTHING`,
      [RELEASE_100_ID],
    );

    const manifestEnvelope = JSON.stringify(RELEASE_VECTOR.envelope);
    const manifestHash = Buffer.from(RELEASE_VECTOR.envelope_sha256, 'hex');
    const signatureBytes = Buffer.from(RELEASE_VECTOR.signature_b64, 'base64');

    await pool.query(
      `INSERT INTO releases(id, version, release_sequence, manifest_schema_version, manifest, manifest_hash, signature, signing_key_ref, component_versions)
       VALUES ($1, '1.1.0', 42, 1, $2::jsonb, $3, $4, $5, $6::jsonb)
       ON CONFLICT (id) DO NOTHING`,
      [
        RELEASE_110_ID,
        manifestEnvelope,
        manifestHash,
        signatureBytes,
        RELEASE_VECTOR.key_id,
        JSON.stringify(RELEASE_VECTOR.envelope.signed.components),
      ],
    );

    for (const art of RELEASE_VECTOR.envelope.signed.artifacts) {
      await pool.query(
        `INSERT INTO artifacts(release_id, platform, arch, digest, size_bytes, installed_size_bytes, storage_ref)
         VALUES ($1, 'ubuntu/24.04', $2, $3, $4::numeric, $5::numeric, $6)
         ON CONFLICT (release_id, platform, arch) DO NOTHING`,
        [
          RELEASE_110_ID,
          art.architecture,
          Buffer.from(art.sha256, 'hex'),
          art.size_bytes,
          art.installed_size_bytes,
          art.url,
        ],
      );
    }

    await pool.query(
      `INSERT INTO release_verifications(release_id, manifest_url, state, verified_at)
       VALUES ($1, 'https://releases.example.invalid/r42/manifest.json', 'verified', clock_timestamp())
       ON CONFLICT (release_id) DO NOTHING`,
      [RELEASE_110_ID],
    );

    await pool.query(
      `UPDATE fleet_config
       SET desired_release_id = $1,
           supported_release_policy = 'allowlist',
           approved_release_ids = ARRAY[$1::uuid, $2::uuid],
           supported_release_ids = ARRAY[$1::uuid, $2::uuid],
           support_generation = 1,
           resource_version = resource_version + 1
       WHERE id = '00000000-0000-0000-0000-000000000001'`,
      [RELEASE_100_ID, RELEASE_110_ID],
    );
  });

  const policiesToSeed = [
    {
      defId: '0195f100-0000-7000-8000-000000000301',
      revId: '0195f100-0000-7000-8000-000000000401',
      type: 'fair_use',
      code: 'standard-fup-default',
      reason: 'Initial Fair Use Policy',
      payload: {
        mode: 'enforce',
        effective_at: '2026-10-01T00:00:00Z',
        effective_period: 'next_full_period',
        accounting_basis: {
          period_type: 'utc_calendar_month',
          metering_version: 'forward_admission_v1',
          measure: 'total_bytes',
        },
        tiers: [
          {
            threshold_bytes: '0',
            down_bps: 100000000,
            up_bps: 50000000,
            priority: 1,
            display_name: { 'zh-TW': '標準速率', en: 'Standard Speed' },
          },
          {
            threshold_bytes: '536870912000',
            down_bps: 10000000,
            up_bps: 5000000,
            priority: 3,
            display_name: { 'zh-TW': '尖峰保護降速', en: 'Fair-Use Adjusted' },
          },
        ],
        minimum_rate: { down_bps: 2000000, up_bps: 1000000 },
        overrides: [],
      },
    },
    {
      defId: '0195f100-0000-7000-8000-000000000303',
      revId: '0195f100-0000-7000-8000-000000000403',
      type: 'dns_profile',
      code: 'standard-protective-default',
      reason: 'Initial Protective DNS Profile',
      payload: {
        profile_code: 'standard',
        display_name: { 'zh-TW': '標準防護 DNS', en: 'Standard Protective DNS' },
        domain_rule_set_revision_ids: ['0195f100-0000-7000-8000-000000000404'],
        threat_feed_revision_ids: [],
        block_response: 'nxdomain',
        blocked_ttl_seconds: 60,
      },
    },
    {
      defId: '0195f100-0000-7000-8000-000000000304',
      revId: '0195f100-0000-7000-8000-000000000404',
      type: 'domain_rule_set',
      code: 'operator-overrides-default',
      reason: 'Initial Domain Rule Set',
      payload: {
        rules: [
          {
            rule_id: '0195f100-0000-7000-8000-000000000411',
            profile_id: null,
            domain: 'partner.example.com',
            include_subdomains: true,
            action: 'allow',
            reason: 'false_positive',
            revision: '1',
          },
          {
            rule_id: '0195f100-0000-7000-8000-000000000412',
            profile_id: 'standard',
            domain: 'phishing-test.example.invalid',
            include_subdomains: true,
            action: 'block',
            reason: 'admin_rule',
            revision: '1',
          },
        ],
      },
    },
    {
      defId: '0195f100-0000-7000-8000-000000000305',
      revId: '0195f100-0000-7000-8000-000000000405',
      type: 'threat_feed',
      code: 'malware-domains-default',
      reason: 'Initial Malware Threat Feed',
      payload: {
        provider: 'ExampleThreatIntel',
        feed_code: 'malware-domains-default',
        license_name: 'CC-BY-4.0',
        license_url: 'https://creativecommons.org/licenses/by/4.0/',
        commercial_use_basis: 'Permitted under CC-BY-4.0 with attribution in portal',
        attribution: 'ExampleThreatIntel Community Feed',
        source_url: 'https://feeds.example.com/malware-domains.txt',
        format: 'domain_list',
        categories: ['malware', 'phishing'],
        content_sha256:
          'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
        max_size_bytes: '10485760',
        valid_from: '2026-10-01T00:00:00Z',
        valid_until: '2026-12-31T00:00:00Z',
        update_interval_seconds: 86400,
      },
    },
    {
      defId: '0195f100-0000-7000-8000-000000000306',
      revId: '0195f100-0000-7000-8000-000000000406',
      type: 'release_support',
      code: 'fleet-release-support',
      reason: 'Initial Release Support Policy',
      payload: {
        approved_release_ids: [RELEASE_100_ID, RELEASE_110_ID],
        revoked_release_ids: [],
        minimum_serving_version: {
          serving_release_ids: [RELEASE_100_ID, RELEASE_110_ID],
        },
      },
    },
  ];

  await runSeedStep('seed-policies', async () => {
    for (const pol of policiesToSeed) {
      const payloadStr = JSON.stringify(pol.payload);
      const payloadHash = crypto.createHash('sha256').update(payloadStr).digest();
      const initialGen = pol.type === 'dns_profile' ? 0 : 1;
      await pool.query(
        `INSERT INTO policy_definitions(id, type, code, generation, resource_version)
         VALUES ($1, $2, $3, $4, 1)
         ON CONFLICT (id) DO NOTHING`,
        [pol.defId, pol.type, pol.code, initialGen],
      );
      await pool.query(
        `INSERT INTO policy_revisions(id, definition_id, type, revision, schema_version, payload, payload_hash, signing_key_ref, created_by, reason)
         VALUES ($1, $2, $3, 1, 1, $4::jsonb, $5, 'desired-state-active', $6, $7)
         ON CONFLICT (id) DO NOTHING`,
        [pol.revId, pol.defId, pol.type, payloadStr, payloadHash, adminId, pol.reason],
      );
      if (pol.type !== 'dns_profile') {
        await pool.query(
          `UPDATE policy_definitions SET active_revision_id = $2, resource_version = resource_version + 1 WHERE id = $1`,
          [pol.defId, pol.revId],
        );
      }
    }

    // Activate dns_profile via POST /api/v1/admin/policies/{definition_id}/activate so fvpn-worker's
    // dns.publish job compiles and publishes the canonical resolver document (D-050, D-056).
    const dnsProfilePol = policiesToSeed.find((p) => p.type === 'dns_profile');
    if (dnsProfilePol) {
      const actRes = await httpsRequest(
        `${controlBaseUrl}/api/v1/admin/policies/${dnsProfilePol.defId}/activate`,
        {
          method: 'POST',
          headers: {
            Cookie: adminAuth.sessionCookie,
            'X-CSRF-Token': adminAuth.csrfToken,
            Origin: 'https://portal.example.invalid',
            'Idempotency-Key': crypto.randomUUID(),
            'If-Match': '"1"',
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            revision_id: dnsProfilePol.revId,
            reason: dnsProfilePol.reason,
          }),
          ca: caPem,
        },
      );
      if (actRes.status !== 202 && actRes.status !== 200) {
        throw new Error(
          `POST /api/v1/admin/policies/${dnsProfilePol.defId}/activate failed (${actRes.status}): ${actRes.text}`,
        );
      }
      const deadline = Date.now() + 15_000;
      let published = false;
      while (Date.now() < deadline) {
        const headRes = await pool.query(
          `SELECT status, document_sha256, policy_generation::text AS policy_generation
           FROM dns_publication_heads
           WHERE definition_id = $1 AND status = 'active'`,
          [dnsProfilePol.defId],
        );
        if (headRes.rows.length > 0) {
          published = true;
          break;
        }
        await new Promise((r) => setTimeout(r, 100));
      }
      if (!published) {
        throw new Error(
          `Timed out waiting for dns.publish worker projection on ${dnsProfilePol.defId}`,
        );
      }
    }
  });

  // Seed and activate the `plan` policy with a rate through the product API
  // (`POST /api/v1/admin/policies` + `POST /api/v1/admin/policies/{definition_id}/activate`)
  // so `plans` and `plan_revisions` carry `rate_limit` and the seeded member is included in desired state.
  const seededPlan = await runSeedStep('seed-plan-policy-via-api', async () => {
    const createRes = await httpsRequest(`${controlBaseUrl}/api/v1/admin/policies`, {
      method: 'POST',
      headers: {
        Cookie: adminAuth.sessionCookie,
        'X-CSRF-Token': adminAuth.csrfToken,
        Origin: 'https://portal.example.invalid',
        'Idempotency-Key': crypto.randomUUID(),
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        type: 'plan',
        code: 'standard-monthly',
        schema_version: 1,
        reason: 'Initial Standard Monthly Plan with rate_limit',
        payload: {
          display_name: { 'zh-TW': '標準月費方案', en: 'Standard Monthly Plan' },
          device_limit: 3,
          exit_group_ids: [exitGroupId],
          price: {
            amount_minor: 19900,
            currency: 'TWD',
            billing_interval: 'month',
            provider: 'stripe',
            provider_price_ref: 'price_monthly_test',
          },
          rate_limit: {
            down_bps: 100000000,
            up_bps: 50000000,
          },
          fair_use_revision_id: '0195f100-0000-7000-8000-000000000401',
          dns_profile_revision_id: '0195f100-0000-7000-8000-000000000403',
        },
      }),
      ca: caPem,
    });
    if (createRes.status !== 201) {
      throw new Error(
        `POST /api/v1/admin/policies (type=plan) failed (${createRes.status}): ${createRes.text}`,
      );
    }
    const createdRev = JSON.parse(createRes.text);
    const planDefId = createdRev.definition_id;
    const planRevId = createdRev.revision_id;

    const actRes = await httpsRequest(
      `${controlBaseUrl}/api/v1/admin/policies/${planDefId}/activate`,
      {
        method: 'POST',
        headers: {
          Cookie: adminAuth.sessionCookie,
          'X-CSRF-Token': adminAuth.csrfToken,
          Origin: 'https://portal.example.invalid',
          'Idempotency-Key': crypto.randomUUID(),
          'If-Match': '"1"',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          revision_id: planRevId,
          reason: 'Activate Standard Monthly Plan with rate_limit',
        }),
        ca: caPem,
      },
    );
    if (actRes.status !== 202 && actRes.status !== 200) {
      throw new Error(
        `POST /api/v1/admin/policies/${planDefId}/activate (type=plan) failed (${actRes.status}): ${actRes.text}`,
      );
    }
    await drainNodeAffectingOutbox(pool, 'seed-plan-policy-via-api');
    return { definitionId: planDefId, revisionId: planRevId };
  });

  await runSeedStep('seed-member-billing-customer', async () => {
    await pool.query(
      `INSERT INTO billing_customers(id, member_id, provider, external_customer_id)
       VALUES ($1, $2, 'stripe', 'cus_test_default_member')
       ON CONFLICT (member_id, provider) DO NOTHING`,
      [crypto.randomUUID(), defaultMemberId],
    );
  });

  await runSeedStep('patch-default-member-via-api', async () => {
    await patchMemberViaApi(controlBaseUrl, adminAuth, caPem, defaultMemberId, {
      display_name: '陳小華 (Active Member)',
      member_group_ids: [exitGroupId],
      notes: 'Default seeded active member',
    });
  });

  await runSeedStep('grant-default-member-entitlement-via-api', async () => {
    await grantMemberEntitlementViaApi(
      controlBaseUrl,
      adminAuth,
      caPem,
      defaultMemberId,
      seededPlan.revisionId,
      periodStart,
      periodEnd,
      'TEST ONLY initial active entitlement grant',
    );
  });

  await runSeedStep('await-default-member-entitlement-projection', async () => {
    await pollMemberEntitlementActive(pool, defaultMemberId, 'defaultMemberId');
  });

  await runSeedStep('await-default-member-device-discovery', async () => {
    await pollMemberAuthorizedDevices(pool, defaultMemberId, 3, 'defaultMemberId');
  });

  await runSeedStep('seed-node-and-usage-fixtures', async () => {
    await pollLiveNodeOnline(pool, liveNode, 'seed-node-and-usage-fixtures-tpe01');
    await pollLiveNodeOnline(pool, rollbackNode, 'seed-node-and-usage-fixtures-tpe02');

    await pool.query(
      `INSERT INTO usage_rollups(id, member_id, period, granularity, projection_generation, metering_version, upload_bytes, download_bytes, last_observed_at, aggregate_as_of, period_start, period_end, estimated, gap, reconciliation_in_progress)
       VALUES ($1, $2, $3, 'month', 1, 'forward_admission_v1', 9007199254740993, 9007199254740995, clock_timestamp(), clock_timestamp(), $4, $5, true, true, false)
       ON CONFLICT (member_id, period, granularity, projection_generation) DO NOTHING`,
      [crypto.randomUUID(), defaultMemberId, currentMonth, monthStart, monthEnd],
    );
  });

  await runSeedStep('publish-live-node-exit-via-api', async () => {
    await publishLiveNodeExitViaApi(
      controlBaseUrl,
      adminAuth,
      caPem,
      pool,
      liveNode,
      exitGroupId,
    );
    await drainNodeAffectingOutbox(pool, 'publish-live-node-exit-via-api');

    const desiredEnvelope = await liveNode.fetchDesiredState();
    const desired = desiredEnvelope?.signed ?? {};
    const memberRatePolicy = (desired.member_rate_policies || []).find(
      (m) => m.member_id === defaultMemberId,
    );
    const memberDeviceBindings = (desired.device_bindings || []).filter(
      (b) => b.member_id === defaultMemberId,
    );
    if (
      !memberRatePolicy ||
      memberRatePolicy.down_bps !== 100000000 ||
      memberRatePolicy.up_bps !== 50000000 ||
      memberDeviceBindings.length < 3
    ) {
      throw new Error(
        `Expected defaultMemberId (${defaultMemberId}) in desired.member_rate_policies with rate 100M/50M and 3 device_bindings; got ratePolicy=${JSON.stringify(memberRatePolicy)}, bindings=${memberDeviceBindings.length}`,
      );
    }
    if (getWorkerLogs().includes('plan_missing_rate')) {
      throw new Error(
        `Unexpected plan_missing_rate warning in fvpn-worker logs:\n${lastLines(getWorkerLogs(), 50)}`,
      );
    }
  });

  await runSeedStep('create-update-batch-and-report-result-via-api', async () => {
    const relRes = await httpsRequest(`${controlBaseUrl}/api/v1/admin/releases`, {
      method: 'GET',
      headers: {
        Cookie: adminAuth.sessionCookie,
        Accept: 'application/json',
      },
      ca: caPem,
    });
    if (relRes.status !== 200) {
      throw new Error(
        `GET /api/v1/admin/releases failed (${relRes.status}): ${relRes.text}`,
      );
    }
    const relData = JSON.parse(relRes.text);
    const expectedFleetGen = String(relData.fleet?.generation ?? '0');

    const batchRes = await httpsRequest(`${controlBaseUrl}/api/v1/admin/update-batches`, {
      method: 'POST',
      headers: {
        Cookie: adminAuth.sessionCookie,
        'X-CSRF-Token': adminAuth.csrfToken,
        Origin: 'https://portal.example.invalid',
        'Idempotency-Key': crypto.randomUUID(),
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        release_id: RELEASE_110_ID,
        scope: 'all_nodes',
        expected_generation: expectedFleetGen,
        set_fleet_default: true,
        reason: 'Scheduled fleet rollout to 1.1.0',
      }),
      ca: caPem,
    });
    if (batchRes.status !== 202) {
      throw new Error(
        `POST /api/v1/admin/update-batches failed (${batchRes.status}): ${batchRes.text}`,
      );
    }
    const createdBatch = JSON.parse(batchRes.text);
    const batchId = createdBatch.batch_id;

    // 1. Primary live node (tw-tpe-01) reports `succeeded` on RELEASE_110_ID
    const desiredEnvelope = await liveNode.fetchDesiredState();
    const updateIntent = desiredEnvelope?.signed?.update;
    const compiledConfigGen = String(desiredEnvelope?.signed?.config_generation ?? '1');
    if (!updateIntent || !updateIntent.job_id) {
      throw new Error(
        `Expected update intent from GET /api/v1/agent/desired-state on liveNode; got ${JSON.stringify(desiredEnvelope?.signed)}`,
      );
    }

    await liveNode.submitUpdateResult(updateIntent, compiledConfigGen);
    await new Promise((r) => setTimeout(r, 10));
    await pollLiveNodeReady(
      pool,
      liveNode,
      'post-update-result-observation-tpe01',
      RELEASE_110_ID,
    );

    // 2. Rollback node (tw-tpe-02) reports `rolled_back` modeled on internal/fleetupdate/testdata/at26-direct-rollback.json
    const rbDesiredEnvelope = await rollbackNode.fetchDesiredState();
    const rbUpdateIntent = rbDesiredEnvelope?.signed?.update;
    const rbCompiledConfigGen = String(
      rbDesiredEnvelope?.signed?.config_generation ?? '1',
    );
    if (!rbUpdateIntent || !rbUpdateIntent.job_id) {
      throw new Error(
        `Expected update intent from GET /api/v1/agent/desired-state on rollbackNode; got ${JSON.stringify(rbDesiredEnvelope?.signed)}`,
      );
    }

    await rollbackNode.submitRolledBackUpdateResult(rbUpdateIntent, rbCompiledConfigGen);
    await new Promise((r) => setTimeout(r, 10));
    await pollLiveNodeReady(
      pool,
      rollbackNode,
      'post-rollback-result-observation-tpe02',
      RELEASE_100_ID,
    );

    const pollDeadline = Date.now() + 15_000;
    let lastBatch = null;
    while (Date.now() < pollDeadline) {
      const getBatchRes = await httpsRequest(
        `${controlBaseUrl}/api/v1/admin/update-batches/${batchId}`,
        {
          method: 'GET',
          headers: {
            Cookie: adminAuth.sessionCookie,
            Accept: 'application/json',
          },
          ca: caPem,
        },
      );
      if (getBatchRes.status === 200) {
        lastBatch = JSON.parse(getBatchRes.text);
        const liveTarget = (lastBatch.targets || []).find(
          (t) => t.node_id === enrolledNodeId,
        );
        const rbTarget = (lastBatch.targets || []).find(
          (t) => t.node_id === rollbackNodeId,
        );
        const offlineTarget = (lastBatch.targets || []).find(
          (t) => t.node_id === secondaryNodeId,
        );
        if (
          liveTarget &&
          liveTarget.state === 'succeeded' &&
          liveTarget.unconfirmed === false &&
          liveTarget.policy_generation === compiledConfigGen &&
          Array.isArray(liveTarget.policy_refs) &&
          liveTarget.policy_refs.includes('0195f100-0000-7000-8000-000000000401') &&
          rbTarget &&
          rbTarget.state === 'rolled_back' &&
          rbTarget.unconfirmed === false &&
          rbTarget.previous_release_id === RELEASE_110_ID &&
          rbTarget.current_release_id === RELEASE_100_ID &&
          rbTarget.health?.tunnel_probe === 'passed' &&
          rbTarget.candidate_health?.tunnel_probe === 'failed' &&
          offlineTarget &&
          offlineTarget.state === 'waiting_online' &&
          offlineTarget.unconfirmed === true &&
          offlineTarget.policy_generation === null &&
          offlineTarget.policy_refs === null &&
          offlineTarget.previous_release_id === null &&
          offlineTarget.health === null &&
          offlineTarget.candidate_health === null
        ) {
          return lastBatch;
        }
      }
      await new Promise((r) => setTimeout(r, 100));
    }
    throw new Error(
      `Timed out waiting for update batch ${batchId} target projections; last=${JSON.stringify(lastBatch)}`,
    );
  });

  return { seededPlan };
}

const MIME_TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.ico': 'image/x-icon',
};

async function main() {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'fvpn-w3-portal-live-'));
  const dbName = `fvpn_w3_portal_live_${process.pid}_${crypto.randomBytes(4).toString('hex')}`;
  const testDbDsn = replaceDbName(ADMIN_DSN, dbName);

  let controlProc = null;
  let workerProc = null;
  let controlLogs = '';
  let workerLogs = '';
  let heartbeatTimer = null;
  let heartbeatInFlight = false;
  let cleanedUp = false;

  const adminClient = new pg.Client({ connectionString: ADMIN_DSN });
  adminClient.on('error', () => {
    // Suppress unhandled client error events during teardown / backend termination
  });
  await adminClient.connect();
  await adminClient.query(`CREATE DATABASE ${dbName}`);

  let pool = null;

  async function cleanup() {
    if (cleanedUp) return;
    cleanedUp = true;
    if (heartbeatTimer) clearInterval(heartbeatTimer);
    for (let i = 0; i < 10 && heartbeatInFlight; i += 1) {
      await new Promise((r) => setTimeout(r, 25));
    }
    if (workerProc && !workerProc.killed) {
      workerProc.kill('SIGTERM');
    }
    if (controlProc && !controlProc.killed) {
      controlProc.kill('SIGTERM');
    }
    if (pool) {
      await pool.end().catch(() => {});
    }
    try {
      await adminClient.query(
        `SELECT pg_terminate_backend(pid)
         FROM pg_stat_activity
         WHERE datname = $1 AND pid <> pg_backend_pid()`,
        [dbName],
      );
    } catch {
      // Tolerate backends that are already gone
    }
    try {
      await adminClient.query(`DROP DATABASE IF EXISTS ${dbName} WITH (FORCE)`);
    } catch {
      // Ignore drop errors on exit or if backends already terminated
    }
    await adminClient.end().catch(() => {});
    try {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    } catch {
      // Ignore
    }
  }

  process.on('SIGINT', async () => {
    await cleanup();
    process.exit(0);
  });
  process.on('SIGTERM', async () => {
    await cleanup();
    process.exit(0);
  });
  process.on('exit', () => {
    if (workerProc && !workerProc.killed) workerProc.kill('SIGKILL');
    if (controlProc && !controlProc.killed) controlProc.kill('SIGKILL');
  });

  try {
    const distDir = path.join(PORTAL_DIR, 'dist');
    if (!fs.existsSync(path.join(distDir, 'index.html'))) {
      console.log('[portal-dev-stack] Building portal static bundle...');
      execFileSync('npm', ['run', 'build'], { cwd: PORTAL_DIR, stdio: 'inherit' });
    }

    console.log('[portal-dev-stack] Building fvpn-control and fvpn-worker...');
    const controlBin = path.join(tmpDir, 'fvpn-control');
    const workerBin = path.join(tmpDir, 'fvpn-worker');
    const goEnv = { ...process.env, CGO_ENABLED: '0', GOTOOLCHAIN: 'local' };
    execFileSync('go', ['build', '-o', controlBin, './apps/control/cmd/fvpn-control'], {
      cwd: REPO_ROOT,
      env: goEnv,
      stdio: 'inherit',
    });
    execFileSync('go', ['build', '-o', workerBin, './apps/worker/cmd/fvpn-worker'], {
      cwd: REPO_ROOT,
      env: goEnv,
      stdio: 'inherit',
    });

    const certs = generateCertsAndVault(tmpDir);
    const caPem = fs.readFileSync(certs.caCertPath);
    const tlsOptions = {
      key: fs.readFileSync(certs.serverKeyPath),
      cert: fs.readFileSync(certs.serverCertPath),
    };

    const [
      oidcPort,
      nbPort,
      stripePort,
      proxyPort,
      publicPort,
      agentPort,
      metricsPort,
      workerMetricsPort,
    ] = await Promise.all([
      getFreePort(),
      getFreePort(),
      getFreePort(),
      getFreePort(),
      getFreePort(),
      getFreePort(),
      getFreePort(),
      getFreePort(),
    ]);

    const portalBaseUrl = `http://127.0.0.1:${PORTAL_PORT}`;
    const controlBaseUrl = `https://127.0.0.1:${publicPort}`;
    const agentBaseUrl = `https://127.0.0.1:${agentPort}`;

    const oidcController = createMockOidcServer(tlsOptions, oidcPort, portalBaseUrl);
    await new Promise((resolve) =>
      oidcController.server.listen(oidcPort, '127.0.0.1', resolve),
    );

    const nbController = createFakeNetBirdServer(tlsOptions);
    await new Promise((resolve) =>
      nbController.server.listen(nbPort, '127.0.0.1', resolve),
    );

    const stripeWebhookSecret = 'whsec_test_portal_live_secret';
    const stripeController = createFakeStripeAndProxy(tlsOptions, stripeWebhookSecret);
    await new Promise((resolve) =>
      stripeController.stripeServer.listen(stripePort, '127.0.0.1', resolve),
    );
    await new Promise((resolve) =>
      stripeController.proxyServer.listen(proxyPort, '127.0.0.1', resolve),
    );

    const dsnFile = path.join(tmpDir, 'db.dsn');
    fs.writeFileSync(dsnFile, testDbDsn, { mode: 0o600 });

    const stripeKeyFile = path.join(tmpDir, 'stripe.key');
    fs.writeFileSync(stripeKeyFile, 'sk_test_synthetic_portal_live', { mode: 0o600 });

    const stripeWebhookSecretFile = path.join(tmpDir, 'stripe.whsec');
    fs.writeFileSync(stripeWebhookSecretFile, stripeWebhookSecret, { mode: 0o600 });

    const billingVaultKeyFile = path.join(tmpDir, 'billing.vault');
    fs.writeFileSync(billingVaultKeyFile, crypto.randomBytes(32), { mode: 0o600 });

    const nbTokenFile = path.join(tmpDir, 'netbird.token');
    fs.writeFileSync(nbTokenFile, 'TEST ONLY synthetic token', { mode: 0o600 });

    const membersConfigPath = path.join(tmpDir, 'members.json');
    fs.writeFileSync(
      membersConfigPath,
      JSON.stringify(
        {
          provider: 'stripe',
          api_key_file: stripeKeyFile,
          webhook_secret_files: [stripeWebhookSecretFile],
          vault_active_key: 'v1',
          vault_key_files: { v1: billingVaultKeyFile },
          live_mode: false,
          price_ids: ['price_monthly_test'],
          return_urls: [
            'https://portal.example.com/member/billing',
            'https://portal.example.invalid/member/billing',
          ],
          full_refund_policy: 'revoke_original_period',
          grace_seconds: 259200,
          grace_reason: 'payment_past_due',
          trial_max_seconds: 0,
          grant_max_seconds: 2592000,
          device_limit: 3,
          portal_ttl_seconds: 60,
          billing_lease_seconds: 60,
          max_webhook_bytes: 65536,
          reconcile_seconds: 1,
          device_sync_seconds: 1,
          netbird_url: `https://127.0.0.1:${nbPort}/api`,
          netbird_deployment: 'nb-dev',
          netbird_token_file: nbTokenFile,
          overlay_scope: 'nb-overlay',
        },
        null,
        2,
      ),
    );

    const nodesConfigPath = path.join(tmpDir, 'nodes.json');
    fs.writeFileSync(
      nodesConfigPath,
      JSON.stringify(
        {
          provider_directory: certs.vaultDir,
          ca_certificate_file: certs.nodeCaCertPath,
          ca_key_id: certs.caKeyId,
          certificate_seconds: 86400,
          renewal_overlap_seconds: 300,
          bootstrap_seconds: 900,
          installer_seconds: 900,
          preinstalled_seconds: 604800,
          authorization_seconds: 3600,
          online_seconds: 180,
          stuck_seconds: 600,
          reconcile_seconds: 1,
          fetch_seconds: 10,
          desired_state_seconds: 600,
          release_origins: [
            'https://releases.example.invalid',
            'https://releases.example.com',
          ],
          dns_resolver_ipv4: ['100.64.0.53'],
          dns_resolver_ipv6: ['2001:db8::53'],
          netbird_url: `https://127.0.0.1:${nbPort}/api`,
          netbird_deployment: 'nb-dev',
          netbird_token_file: nbTokenFile,
        },
        null,
        2,
      ),
    );

    const dnsConfigPath = path.join(tmpDir, 'dns.json');
    fs.writeFileSync(
      dnsConfigPath,
      JSON.stringify({ upstreams: ['192.0.2.53:53'] }, null, 2),
    );

    const policiesConfigPath = path.join(tmpDir, 'policies.json');
    fs.writeFileSync(
      policiesConfigPath,
      JSON.stringify(
        {
          // TEST ONLY settings for local/E2E control-plane and worker composition (D-045)
          fup_mode: 'shadow',
          fup_sweep_seconds: 60,
          fup_batch_size: 256,
          dns_upstream_profile: 'test-recursive-v1',
        },
        null,
        2,
      ),
    );

    const usageConfigPath = path.join(tmpDir, 'usage.json');
    fs.writeFileSync(
      usageConfigPath,
      JSON.stringify(
        {
          // TEST ONLY settings referencing per-run ephemeral signing vault (D-045)
          signing_config_file: certs.vaultDir,
          raw_usage_retention_days: 90,
          rollup_retention_months: 24,
          audit_retention_days: 365,
          projection_batch_size: 256,
        },
        null,
        2,
      ),
    );

    const centralConfig = {
      public_address: `127.0.0.1:${publicPort}`,
      agent_address: `127.0.0.1:${agentPort}`,
      metrics_address: `127.0.0.1:${metricsPort}`,
      public_origin: 'https://portal.example.invalid',
      tls_certificate_file: certs.serverCertPath,
      tls_key_file: certs.serverKeyPath,
      agent_ca_file: certs.nodeCaCertPath,
      database_url_file: dsnFile,
      oidc_issuer: oidcController.issuer,
      oidc_client_id: 'portal-web',
      admin_claim: 'roles',
      admin_value: 'admin',
      mfa_amr: ['mfa'],
      members_config_file: membersConfigPath,
      nodes_config_file: nodesConfigPath,
      dns_config_file: dnsConfigPath,
      policies_config_file: policiesConfigPath,
      usage_config_file: usageConfigPath,
      ingress_rate: { per_second: 1000, burst: 1000 },
      browser_rate: { per_second: 1000, burst: 1000 },
      login_rate: { per_second: 1000, burst: 1000 },
      agent_rate: { per_second: 1000, burst: 1000 },
      heartbeat_rate: { per_second: 1000, burst: 1000 },
      usage_rate: { per_second: 1000, burst: 1000 },
      lease_rate: { per_second: 1000, burst: 1000 },
      anonymous_rate: { per_second: 1000, burst: 1000 },
      worker_poll_seconds: 1,
      worker_backoff_seconds: 1,
      worker_backoff_max_seconds: 2,
      migrate_on_start: true,
    };
    const centralConfigPath = path.join(tmpDir, 'central.json');
    fs.writeFileSync(centralConfigPath, JSON.stringify(centralConfig, null, 2));

    const workerCentralConfigPath = path.join(tmpDir, 'worker-central.json');
    fs.writeFileSync(
      workerCentralConfigPath,
      JSON.stringify(
        {
          ...centralConfig,
          metrics_address: `127.0.0.1:${workerMetricsPort}`,
          migrate_on_start: false,
        },
        null,
        2,
      ),
    );

    const childEnv = {
      ...process.env,
      SSL_CERT_FILE: certs.caCertPath,
      HTTPS_PROXY: `http://127.0.0.1:${proxyPort}`,
      NO_PROXY: '127.0.0.1,localhost',
    };

    console.log('[portal-dev-stack] Starting fvpn-control (with migrations)...');
    let controlExitCode = null;
    controlProc = spawn(controlBin, ['-config', centralConfigPath, '-migrate-on-start'], {
      cwd: REPO_ROOT,
      env: childEnv,
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    controlProc.stdout.on('data', (d) => {
      controlLogs += d.toString();
      if (process.env.DEBUG_STACK) process.stdout.write(`[fvpn-control] ${d}`);
    });
    controlProc.stderr.on('data', (d) => {
      controlLogs += d.toString();
      if (process.env.DEBUG_STACK) process.stderr.write(`[fvpn-control] ${d}`);
    });
    controlProc.on('exit', (code) => {
      controlExitCode = code;
    });

    let controlReady = false;
    for (let i = 0; i < 100; i += 1) {
      if (controlExitCode !== null) {
        throw new Error(
          `fvpn-control exited early (${controlExitCode}):\n${controlLogs}`,
        );
      }
      try {
        const r = await httpsRequest(`${controlBaseUrl}/readyz`, { ca: caPem });
        if (r.status === 200) {
          controlReady = true;
          break;
        }
      } catch {
        // Wait
      }
      await new Promise((r) => setTimeout(r, 200));
    }
    if (!controlReady) {
      throw new Error(`fvpn-control did not become ready:\n${controlLogs}`);
    }

    console.log('[portal-dev-stack] Starting fvpn-worker...');
    workerProc = spawn(workerBin, ['-config', workerCentralConfigPath], {
      cwd: REPO_ROOT,
      env: childEnv,
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    workerProc.stdout.on('data', (d) => {
      workerLogs += d.toString();
      if (process.env.DEBUG_STACK) process.stdout.write(`[fvpn-worker] ${d}`);
    });
    workerProc.stderr.on('data', (d) => {
      workerLogs += d.toString();
      if (process.env.DEBUG_STACK) process.stderr.write(`[fvpn-worker] ${d}`);
    });

    pool = new pg.Pool({ connectionString: testDbDsn });
    pool.on('error', () => {
      // Prevent unhandled pg-pool error events (e.g. 57P01 administrator command) from crashing teardown
    });
    pool.on('connect', (client) => {
      client.on('error', () => {
        // Prevent checked-out client error events during teardown from crashing process
      });
    });

    const adminAuth = await runSeedStep('oidc-login-admin', async () =>
      performOidcLogin(
        controlBaseUrl,
        oidcController,
        {
          role: 'admin',
          sub: 'oidc-sub-admin-1',
          name: 'Operator Admin (tw-ops)',
        },
        caPem,
      ),
    );
    const adminRowRes = await pool.query(
      `SELECT id FROM admin_identities WHERE subject = 'oidc-sub-admin-1' LIMIT 1`,
    );
    const adminId = adminRowRes.rows[0]?.id || '00000000-0000-0000-0000-000000000003';

    const memberAuth = await runSeedStep('oidc-login-default-member', async () =>
      performOidcLogin(
        controlBaseUrl,
        oidcController,
        {
          role: 'member',
          sub: 'oidc-sub-member-1',
          name: '陳小華 (Active Member)',
        },
        caPem,
      ),
    );
    const defaultMemberId = memberAuth.session.member_id;

    const { exitGroupId } = await runSeedStep(
      'create-member-groups-via-api',
      async () => {
        const exitGrp = await createGroupViaApi(
          controlBaseUrl,
          adminAuth,
          caPem,
          'standard-exits',
          'Standard Exit Group',
        );
        await createGroupViaApi(
          controlBaseUrl,
          adminAuth,
          caPem,
          'default-members',
          'Default Member Group',
        );
        return { exitGroupId: exitGrp.group_id };
      },
    );

    nbController.setNextSetupKeyPeer({ id: 'nb-peer-node-1', name: 'tw-tpe-01' });
    const liveNode = await runSeedStep('enroll-live-node-1', async () => {
      const node = await enrollLiveNode(
        controlBaseUrl,
        agentBaseUrl,
        adminAuth,
        caPem,
        tmpDir,
        {
          name: 'tw-tpe-01',
          region: 'tw-tpe',
          arch: 'amd64',
          peerId: 'nb-peer-node-1',
          memberGroupIds: [exitGroupId],
          notes: 'Primary Taipei exit node',
          filePrefix: 'enrolled-node-1',
        },
      );
      await drainNodeAffectingOutbox(pool, 'enroll-live-node-1');
      return node;
    });

    nbController.setNextSetupKeyPeer({ id: 'nb-peer-node-2', name: 'jp-tyo-01' });
    const secondaryNode = await runSeedStep('enroll-live-node-2', async () => {
      const node = await enrollLiveNode(
        controlBaseUrl,
        agentBaseUrl,
        adminAuth,
        caPem,
        tmpDir,
        {
          name: 'jp-tyo-01',
          region: 'jp-tyo',
          arch: 'arm64',
          peerId: 'nb-peer-node-2',
          memberGroupIds: [exitGroupId],
          notes: 'Secondary Tokyo exit node',
          filePrefix: 'enrolled-node-2',
        },
      );
      await drainNodeAffectingOutbox(pool, 'enroll-live-node-2');
      return node;
    });

    nbController.setNextSetupKeyPeer({ id: 'nb-peer-node-3', name: 'tw-tpe-02' });
    const rollbackNode = await runSeedStep('enroll-live-node-3', async () => {
      const node = await enrollLiveNode(
        controlBaseUrl,
        agentBaseUrl,
        adminAuth,
        caPem,
        tmpDir,
        {
          name: 'tw-tpe-02',
          region: 'tw-tpe',
          arch: 'amd64',
          peerId: 'nb-peer-node-3',
          memberGroupIds: [exitGroupId],
          notes: 'Secondary Taipei rollback test node',
          filePrefix: 'enrolled-node-3',
        },
      );
      await drainNodeAffectingOutbox(pool, 'enroll-live-node-3');
      return node;
    });

    const { seededPlan } = await seedDatabase(
      pool,
      adminId,
      adminAuth,
      controlBaseUrl,
      caPem,
      defaultMemberId,
      liveNode,
      secondaryNode,
      rollbackNode,
      exitGroupId,
      () => workerLogs,
    );
    await runSeedStep('submit-initial-node-observation', async () => {
      await pollLiveNodeReady(
        pool,
        liveNode,
        'submit-initial-node-observation',
        RELEASE_110_ID,
      );
    });

    let holdNodeObservations = false;
    heartbeatTimer = setInterval(async () => {
      if (heartbeatInFlight || holdNodeObservations) return;
      heartbeatInFlight = true;
      try {
        const stateRes = await pool.query(
          `SELECT
             greatest(n.desired_config_generation, 1)::text AS desired_gen,
             (n.applied_config_generation < n.desired_config_generation) AS gen_behind,
             (n.provisioning_state <> 'ready'
              OR n.control_status <> 'online'
              OR n.safety_status <> 'passed') AS state_not_ready,
             EXISTS(
               SELECT 1 FROM operations
               WHERE kind IN ('node.publish', 'node.unpublish', 'node.disable')
                 AND status IN ('pending', 'running', 'partially_succeeded')
             ) AS has_pending_op,
             EXISTS(
               SELECT 1 FROM outbox_events
               WHERE delivered_at IS NULL
                 AND dead_lettered_at IS NULL
                 AND event_type = 'node.peer.observe'
             ) AS has_pending_observe
           FROM nodes n
           WHERE n.id = $1`,
          [liveNode.nodeId],
        );
        const row = stateRes.rows[0];
        if (
          row &&
          !holdNodeObservations &&
          (row.gen_behind ||
            (!row.has_pending_observe && (row.state_not_ready || row.has_pending_op)))
        ) {
          await liveNode.submitObservation(row.desired_gen || '1');
        }
      } catch {
        // Ignore transient errors during shutdown
      } finally {
        heartbeatInFlight = false;
      }
    }, 250);

    const roleSessions = {
      admin: adminAuth,
      member: memberAuth,
    };
    let currentInteractiveRole = 'member';

    const portalServer = http.createServer(async (req, res) => {
      try {
        const reqUrl = new URL(req.url || '/', portalBaseUrl);

        if (reqUrl.pathname.startsWith('/__e2e/')) {
          let body = '';
          req.on('data', (c) => {
            body += c;
          });
          req.on('end', async () => {
            try {
              if (
                req.method === 'POST' &&
                reqUrl.pathname === '/__e2e/reset-member-journey'
              ) {
                const sub = `oidc-sub-journey-${crypto.randomUUID()}`;
                const journeyAuth = await performOidcLogin(
                  controlBaseUrl,
                  oidcController,
                  {
                    role: 'member',
                    sub,
                    name: '林測試 (Journey Member)',
                  },
                  caPem,
                );
                const journeyMemberId = journeyAuth.session.member_id;
                await patchMemberViaApi(
                  controlBaseUrl,
                  adminAuth,
                  caPem,
                  journeyMemberId,
                  {
                    display_name: '林測試 (Journey Member)',
                    member_group_ids: [exitGroupId],
                  },
                );
                nbController.objects.users.set(sub, {
                  id: sub,
                  auto_groups: [],
                  is_service_user: false,
                  is_blocked: false,
                });
                roleSessions.member = journeyAuth;
                currentInteractiveRole = 'member';
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(
                  JSON.stringify({
                    member_id: journeyMemberId,
                    role: 'member',
                  }),
                );
                return;
              }

              if (
                req.method === 'POST' &&
                reqUrl.pathname === '/__e2e/restore-default-member'
              ) {
                roleSessions.member = memberAuth;
                currentInteractiveRole = 'member';
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ member_id: defaultMemberId }));
                return;
              }

              if (req.method === 'POST' && reqUrl.pathname === '/__e2e/stripe-webhook') {
                const parsed = body ? JSON.parse(body) : {};
                const signed = stripeController.buildSignedWebhookEvent(
                  parsed.checkout_id,
                );
                const whRes = await httpsRequest(
                  `${controlBaseUrl}/api/v1/webhooks/stripe`,
                  {
                    method: 'POST',
                    headers: {
                      'Content-Type': 'application/json',
                      'Stripe-Signature': signed.signatureHeader,
                    },
                    body: signed.rawBody,
                    ca: caPem,
                  },
                );
                const targetMemberId = roleSessions.member.session.member_id;
                await pollMemberEntitlementActive(pool, targetMemberId, 'stripe-webhook');
                res.writeHead(whRes.status, { 'Content-Type': 'application/json' });
                res.end(whRes.text);
                return;
              }

              if (
                req.method === 'POST' &&
                reqUrl.pathname === '/__e2e/seed-member-devices'
              ) {
                const targetMemberId = roleSessions.member.session.member_id;
                const entRes = await pool.query(
                  `SELECT m.status AS member_status, e.state AS entitlement_state
                   FROM members m
                   LEFT JOIN entitlements e ON e.member_id = m.id
                   WHERE m.id = $1`,
                  [targetMemberId],
                );
                if (
                  !entRes.rows[0] ||
                  entRes.rows[0].member_status !== 'active' ||
                  entRes.rows[0].entitlement_state !== 'active'
                ) {
                  const now = new Date();
                  const vFrom = new Date(now.getTime() - 3600_000);
                  const vUntil = new Date(now.getTime() + 14 * 86400_000);
                  await grantMemberEntitlementViaApi(
                    controlBaseUrl,
                    adminAuth,
                    caPem,
                    targetMemberId,
                    seededPlan.revisionId,
                    vFrom,
                    vUntil,
                    'TEST ONLY E2E device test grant',
                  );
                  await pollMemberEntitlementActive(
                    pool,
                    targetMemberId,
                    'seed-member-devices',
                  );
                }

                const existingRes = await pool.query(
                  `SELECT d.id
                   FROM member_devices d
                   JOIN device_authorization_observations o ON o.device_id = d.id
                   WHERE d.member_id = $1
                     AND d.status = 'authorized'
                     AND o.observed_authorized = true
                   ORDER BY d.id`,
                  [targetMemberId],
                );
                if (existingRes.rows.length < 3) {
                  const netbirdUserId = roleSessions.member.sub || 'oidc-sub-member-1';
                  nbController.objects.users.set(netbirdUserId, {
                    id: netbirdUserId,
                    auto_groups: [],
                    is_service_user: false,
                    is_blocked: false,
                  });
                  const subnetOctet = Math.floor(Math.random() * 200) + 10;
                  for (let idx = 1; idx <= 3; idx += 1) {
                    const suffix = crypto.randomUUID().slice(0, 6);
                    const peerId = `nb-peer-journey-${idx}-${suffix}`;
                    const ip = `100.64.${subnetOctet}.${20 + idx}`;
                    nbController.objects.peers.set(peerId, {
                      id: peerId,
                      name: `Journey Device ${idx}`,
                      user_id: netbirdUserId,
                      ephemeral: false,
                      ip,
                    });
                  }
                }
                const deviceIds = await pollMemberAuthorizedDevices(
                  pool,
                  targetMemberId,
                  3,
                  'seed-member-devices',
                );
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ device_ids: deviceIds }));
                return;
              }

              if (
                req.method === 'POST' &&
                reqUrl.pathname === '/__e2e/hold-node-observations'
              ) {
                const parsed = JSON.parse(body || '{}');
                if (parsed.hold) {
                  holdNodeObservations = true;
                  while (heartbeatInFlight) {
                    await new Promise((r) => setTimeout(r, 25));
                  }
                  await drainNodeAffectingOutbox(pool, 'hold-node-observations-pre');
                  await pollLiveNodeReady(pool, liveNode, 'hold-node-observations');
                  await drainNodeAffectingOutbox(pool, 'hold-node-observations-post');
                } else {
                  holdNodeObservations = false;
                }
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ held: holdNodeObservations }));
                return;
              }

              if (
                req.method === 'POST' &&
                reqUrl.pathname === '/__e2e/bump-node-version'
              ) {
                const parsed = JSON.parse(body || '{}');
                const nodeId = parsed.node_id || liveNode.nodeId;
                const updated = await pool.query(
                  `UPDATE nodes SET notes = 'Concurrent operator edit', resource_version = resource_version + 1 WHERE id = $1 RETURNING resource_version::text`,
                  [nodeId],
                );
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(
                  JSON.stringify({
                    resource_version: updated.rows[0]?.resource_version,
                  }),
                );
                return;
              }

              res.writeHead(404);
              res.end();
            } catch (e) {
              res.writeHead(500, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify({ error: String(e) }));
            }
          });
          return;
        }

        if (
          reqUrl.pathname.startsWith('/auth/') ||
          reqUrl.pathname.startsWith('/api/v1/')
        ) {
          const requestedRole = reqUrl.searchParams.get('role');
          if (requestedRole === 'admin' || requestedRole === 'member') {
            currentInteractiveRole = requestedRole;
          } else if (reqUrl.pathname.startsWith('/api/v1/admin/')) {
            currentInteractiveRole = 'admin';
          } else if (reqUrl.pathname.startsWith('/api/v1/member/')) {
            currentInteractiveRole = 'member';
          }

          if (req.method === 'GET' && reqUrl.pathname === '/auth/login') {
            const roleToLogin = requestedRole === 'admin' ? 'admin' : 'member';
            currentInteractiveRole = roleToLogin;
            oidcController.setNextLoginPersona(
              roleToLogin === 'admin'
                ? {
                    role: 'admin',
                    sub: 'oidc-sub-admin-1',
                    name: 'Operator Admin (tw-ops)',
                  }
                : {
                    role: 'member',
                    sub: roleSessions.member.sub || 'oidc-sub-member-1',
                    name: roleSessions.member.session.display_label || 'Member',
                  },
            );
            const loginRes = await httpsRequest(`${controlBaseUrl}/auth/login`, {
              method: 'GET',
              ca: caPem,
            });
            const flowCookie = (loginRes.headers['set-cookie'] || [])
              .map((c) => c.split(';')[0])
              .find((c) => c.startsWith('__Host-fvpn-login='));
            const flowVal = flowCookie ? flowCookie.split('=')[1] : '';
            res.writeHead(302, {
              Location: loginRes.headers.location,
              'Set-Cookie': `fvpn_login=${flowVal}; Path=/; HttpOnly; SameSite=Lax`,
            });
            res.end();
            return;
          }

          if (req.method === 'GET' && reqUrl.pathname === '/auth/callback') {
            const rawCookies = req.headers.cookie || '';
            const match = rawCookies.match(/fvpn_login=([^;]+)/);
            const flowVal = match ? match[1] : '';
            const cbRes = await httpsRequest(
              `${controlBaseUrl}${reqUrl.pathname}${reqUrl.search}`,
              {
                method: 'GET',
                headers: {
                  Cookie: `__Host-fvpn-login=${flowVal}`,
                },
                ca: caPem,
              },
            );
            const sessionCookie = (cbRes.headers['set-cookie'] || [])
              .map((c) => c.split(';')[0])
              .find(
                (c) =>
                  c.startsWith('__Host-fvpn-session=') ||
                  c.startsWith('__Host-fvpn-admin=') ||
                  c.startsWith('__Host-fvpn-member='),
              );
            if (sessionCookie) {
              const sessCheck = await httpsRequest(`${controlBaseUrl}/auth/session`, {
                method: 'GET',
                headers: { Cookie: sessionCookie },
                ca: caPem,
              });
              const sessData = JSON.parse(sessCheck.text);
              const csrfToken = sessCheck.headers['x-csrf-token'];
              roleSessions[currentInteractiveRole] = {
                sessionCookie,
                csrfToken,
                session: sessData,
                sub:
                  currentInteractiveRole === 'admin'
                    ? 'oidc-sub-admin-1'
                    : roleSessions.member.sub || 'oidc-sub-member-1',
              };
            }
            const redirectTarget =
              currentInteractiveRole === 'admin' ? '/admin/overview' : '/member/service';
            res.writeHead(303, { Location: redirectTarget });
            res.end();
            return;
          }

          const activeAuth = roleSessions[currentInteractiveRole] || roleSessions.member;

          let reqBody = '';
          req.on('data', (c) => {
            reqBody += c;
          });
          req.on('end', async () => {
            try {
              const forwardHeaders = {
                Accept: req.headers.accept || 'application/json',
                Cookie: activeAuth.sessionCookie,
                Origin: 'https://portal.example.invalid',
                'X-CSRF-Token': activeAuth.csrfToken,
              };
              if (req.headers['content-type']) {
                forwardHeaders['Content-Type'] = req.headers['content-type'];
              }
              if (req.headers['idempotency-key']) {
                forwardHeaders['Idempotency-Key'] = req.headers['idempotency-key'];
              }
              if (req.headers['if-match']) {
                forwardHeaders['If-Match'] = req.headers['if-match'];
              }

              let upstreamBody = reqBody ? reqBody : undefined;
              if (
                req.method === 'POST' &&
                reqUrl.pathname === '/api/v1/member/checkout-sessions' &&
                reqBody
              ) {
                try {
                  const parsed = JSON.parse(reqBody);
                  if (parsed.plan_id === PLAN_ID && seededPlan?.definitionId) {
                    parsed.plan_id = seededPlan.definitionId;
                    upstreamBody = JSON.stringify(parsed);
                  }
                } catch {
                  // Keep original body if not valid JSON
                }
              }

              const upstreamRes = await httpsRequest(
                `${controlBaseUrl}${reqUrl.pathname}${reqUrl.search}`,
                {
                  method: req.method,
                  headers: forwardHeaders,
                  body: upstreamBody,
                  ca: caPem,
                },
              );

              if (
                !holdNodeObservations &&
                req.method === 'PUT' &&
                reqUrl.pathname.endsWith('/publication') &&
                (upstreamRes.status === 200 || upstreamRes.status === 202)
              ) {
                await liveNode.submitObservation();
              }

              const respHeaders = {};
              for (const [k, v] of Object.entries(upstreamRes.headers)) {
                if (
                  k.toLowerCase() === 'set-cookie' ||
                  k.toLowerCase() === 'content-length'
                ) {
                  continue;
                }
                if (v !== undefined) {
                  respHeaders[k] = v;
                }
              }
              res.writeHead(upstreamRes.status, respHeaders);
              res.end(upstreamRes.text);
            } catch (err) {
              if (!res.headersSent) {
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: String(err) }));
              }
            }
          });
          return;
        }

        if (reqUrl.pathname.startsWith('/admin')) {
          currentInteractiveRole = 'admin';
        } else if (reqUrl.pathname.startsWith('/member')) {
          currentInteractiveRole = 'member';
        }

        let filePath = path.join(
          distDir,
          reqUrl.pathname === '/' ? 'index.html' : reqUrl.pathname,
        );
        if (
          !filePath.startsWith(distDir) ||
          !fs.existsSync(filePath) ||
          fs.statSync(filePath).isDirectory()
        ) {
          filePath = path.join(distDir, 'index.html');
        }
        const ext = path.extname(filePath);
        const contentType = MIME_TYPES[ext] || 'application/octet-stream';
        res.writeHead(200, { 'Content-Type': contentType });
        fs.createReadStream(filePath).pipe(res);
      } catch (err) {
        res.writeHead(500, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: String(err) }));
      }
    });

    await new Promise((resolve) =>
      portalServer.listen(PORTAL_PORT, '127.0.0.1', resolve),
    );
    console.log(
      `[portal-dev-stack] Ready at ${portalBaseUrl} (control=${controlBaseUrl}, db=${dbName})`,
    );
  } catch (err) {
    console.error('[portal-dev-stack] Fatal error:', err);
    console.error(
      `[portal-dev-stack] Diagnostic tail — last 50 lines of fvpn-control:\n${lastLines(controlLogs, 50)}\n` +
        `[portal-dev-stack] Diagnostic tail — last 50 lines of fvpn-worker:\n${lastLines(workerLogs, 50)}`,
    );
    await cleanup();
    process.exit(1);
  }
}

main().catch((err) => {
  console.error('[portal-dev-stack] Fatal error:', err);
  process.exit(1);
});
