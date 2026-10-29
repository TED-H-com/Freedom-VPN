#!/usr/bin/env node
/**
 * Freedom VPN Portal Live Development & E2E Stack (`portal-dev-stack`).
 *
 * Responsibilities (AGENTS.md, Project Spec §18, D-026, D-031, D-032):
 * 1. Creates an isolated PostgreSQL database from `$FVPN_TEST_DATABASE_URL` and drops it on exit.
 * 2. Generates ephemeral TLS certificates, mTLS node CA, and FileProvider vault.
 * 3. Runs local HTTPS mock OIDC provider, fake NetBird management API, and fake Stripe API + HTTPS CONNECT proxy.
 * 4. Builds and launches real `fvpn-control -migrate-on-start` and `fvpn-worker` binaries.
 * 5. Enrolls a live node via `/api/v1/agent/enroll` and runs an mTLS agent observation loop so
 *    multi-step operations (`netbird` + `nodes`) reach `succeeded` through real control/worker code.
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

// Deterministic UUID constants for seeded test data
const PLAN_ID = 'b4fe029a-ecbf-4b1a-9ee7-de399791ae6e';
const PLAN_REVISION_ID = '0195f100-0000-7000-8000-000000000101';
const EXIT_GROUP_ID = '11111111-1111-4111-8111-111111111111';
const RELEASE_110_ID = RELEASE_VECTOR.envelope.signed.release_id; // 6d2797d0-b9a5-4724-96af-9b3039d351b1
const SEEDED_BATCH_ID = '85ef720a-31a0-4250-90d2-4b92c58695bc';
const SEEDED_NODE_2_ID = '0195f100-0000-7000-8000-000000000202';
const SEEDED_NODE_2_INC_ID = '0195f100-0000-7000-8000-000000000203';

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
        'nb-peer-dev-1',
        {
          id: 'nb-peer-dev-1',
          name: 'Member MacBook Pro',
          user_id: 'nb-user-member-1',
          ephemeral: false,
          ip: '100.64.0.21',
        },
      ],
      [
        'nb-peer-dev-2',
        {
          id: 'nb-peer-dev-2',
          name: 'Member iPhone 16',
          user_id: 'nb-user-member-1',
          ephemeral: false,
          ip: '100.64.0.22',
        },
      ],
      [
        'nb-peer-dev-3',
        {
          id: 'nb-peer-dev-3',
          name: 'Member Linux Workstation',
          user_id: 'nb-user-member-1',
          ephemeral: false,
          ip: '100.64.0.23',
        },
      ],
    ]),
    'setup-keys': new Map(),
    users: new Map([
      [
        'nb-user-member-1',
        {
          id: 'nb-user-member-1',
          auto_groups: [],
          is_service_user: false,
          is_blocked: false,
        },
      ],
    ]),
    'dns/nameservers': new Map(),
  };

  let serial = 1;

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

  return { server, objects };
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
      periodEnd: nowSec + 30 * 86400,
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
    .find((c) => c.startsWith('__Host-fvpn-session='));
  if (!sessionCookie) {
    throw new Error('Missing __Host-fvpn-session cookie from /auth/callback');
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
  };
}

async function enrollLiveNode(
  controlBaseUrl,
  agentBaseUrl,
  adminAuth,
  caPem,
  tmpDir,
  pool,
) {
  await pool.query(
    `INSERT INTO member_groups(id, code, name)
     VALUES ($1, 'standard-exits', 'Standard Exit Group'),
            ('63575bae-2d23-45c0-ae11-ba6bb9ace34a', 'default-members', 'Default Member Group')
     ON CONFLICT (id) DO NOTHING`,
    [EXIT_GROUP_ID],
  );

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
      name: 'tw-tpe-01',
      region: 'tw-tpe',
      member_group_ids: [EXIT_GROUP_ID],
      notes: 'Primary Taipei exit node',
    }),
    ca: caPem,
  });
  if (createRes.status !== 201) {
    throw new Error(`CreateEnrollment failed (${createRes.status}): ${createRes.text}`);
  }
  const created = JSON.parse(createRes.text);
  const token = created.token;
  const nodeId = created.enrollment.node_id;

  const nodeKeyPath = path.join(tmpDir, 'enrolled-node.key');
  const nodeCsrPath = path.join(tmpDir, 'enrolled-node.csr');
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
  const platform = { arch: 'amd64', os: 'ubuntu', os_version: '24.04' };
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

  await pool.query(
    `UPDATE node_incarnations SET deployment_id = 'nb-dev', peer_id = 'nb-peer-node-1' WHERE id = $1`,
    [incarnationId],
  );

  const bootId = crypto.randomUUID();
  const ledgerEpoch = crypto.randomUUID();
  const meterEpochId = crypto.randomUUID();

  async function submitObservation(appliedGen = '1') {
    const nowIso = new Date().toISOString();
    const obsBody = {
      report_id: crypto.randomUUID(),
      node_id: nodeId,
      node_incarnation_id: incarnationId,
      boot_id: bootId,
      observed_at: nowIso,
      agent_version: '1.1.0',
      current_release_id: RELEASE_110_ID,
      last_good_release_id: RELEASE_110_ID,
      applied_config_generation: String(appliedGen),
      applied_policy_refs: {
        authorization_generation: '1',
        fup_generation: '1',
        dns_generation: '1',
      },
      applied_binding_revision: '1',
      provisioning_state: 'ready',
      safety_status: 'passed',
      data_health: 'healthy',
      member_forwarding_enabled: true,
      forwarding_block_reasons: [],
      lease: {
        lease_id: null,
        expires_at: null,
      },
      netbird: {
        peer_id: 'nb-peer-node-1',
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
      update: {
        job_id: null,
        generation: null,
        state: null,
        stage: null,
      },
      checks: [
        {
          name: 'internet_only',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
        },
        {
          name: 'meter_path',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
        },
        {
          name: 'tunnel_probe',
          status: 'passed',
          observed_at: nowIso,
          error_code: null,
        },
      ],
      errors: [],
    };
    await httpsRequest(`${agentBaseUrl}/api/v1/agent/observations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(obsBody),
      ca: caPem,
      cert: clientCertPem,
      key: nodeKeyPem,
    });
  }

  return {
    nodeId,
    incarnationId,
    clientCertPem,
    nodeKeyPem,
    submitObservation,
  };
}

async function seedDatabase(pool, adminId, defaultMemberId, enrolledNodeId) {
  const now = new Date();
  const periodStart = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));
  const periodEnd = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1));
  const currentMonth = `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, '0')}`;

  await pool.query(
    `INSERT INTO member_groups(id, code, name)
     VALUES ($1, 'standard-exits', 'Standard Exit Group'),
            ('63575bae-2d23-45c0-ae11-ba6bb9ace34a', 'default-members', 'Default Member Group')
     ON CONFLICT (id) DO NOTHING`,
    [EXIT_GROUP_ID],
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
         approved_release_ids = ARRAY[$1::uuid],
         supported_release_ids = ARRAY[$1::uuid],
         support_generation = 1,
         resource_version = resource_version + 1
     WHERE id = '00000000-0000-0000-0000-000000000001'`,
    [RELEASE_110_ID],
  );

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
      defId: '0195f100-0000-7000-8000-000000000302',
      revId: '0195f100-0000-7000-8000-000000000402',
      type: 'plan',
      code: 'standard-monthly',
      reason: 'Initial Standard Monthly Plan',
      payload: {
        display_name: { 'zh-TW': '標準月費方案', en: 'Standard Monthly Plan' },
        device_limit: 3,
        exit_group_ids: [EXIT_GROUP_ID],
        price: {
          amount_minor: 19900,
          currency: 'TWD',
          billing_interval: 'month',
          provider: 'stripe',
          provider_price_ref: 'price_monthly_test',
        },
        rate_limit: null,
        fair_use_revision_id: '0195f100-0000-7000-8000-000000000401',
        dns_profile_revision_id: '0195f100-0000-7000-8000-000000000403',
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
        threat_feed_revision_ids: ['0195f100-0000-7000-8000-000000000405'],
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
        approved_release_ids: [RELEASE_110_ID],
        revoked_release_ids: [],
        minimum_serving_version: {
          serving_release_ids: [RELEASE_110_ID],
        },
      },
    },
  ];

  for (const pol of policiesToSeed) {
    const payloadStr = JSON.stringify(pol.payload);
    const payloadHash = crypto.createHash('sha256').update(payloadStr).digest();
    await pool.query(
      `INSERT INTO policy_definitions(id, type, code, generation, resource_version)
       VALUES ($1, $2, $3, 1, 1)
       ON CONFLICT (id) DO NOTHING`,
      [pol.defId, pol.type, pol.code],
    );
    await pool.query(
      `INSERT INTO policy_revisions(id, definition_id, type, revision, schema_version, payload, payload_hash, signing_key_ref, created_by, reason)
       VALUES ($1, $2, $3, 1, 1, $4::jsonb, $5, 'desired-state-active', $6, $7)
       ON CONFLICT (id) DO NOTHING`,
      [pol.revId, pol.defId, pol.type, payloadStr, payloadHash, adminId, pol.reason],
    );
    await pool.query(
      `UPDATE policy_definitions SET active_revision_id = $2, resource_version = resource_version + 1 WHERE id = $1`,
      [pol.defId, pol.revId],
    );
  }

  await pool.query(
    `INSERT INTO plans(id, code, resource_version)
     VALUES ($1, 'standard-monthly', 1)
     ON CONFLICT (id) DO NOTHING`,
    [PLAN_ID],
  );
  await pool.query(
    `INSERT INTO plan_revisions(id, plan_id, revision, device_limit, exit_group_ids, fup_policy_revision_id, provider, provider_price_ref, schema_version, payload)
     VALUES ($1, $2, 1, 3, ARRAY[$3::uuid], '0195f100-0000-7000-8000-000000000401', 'stripe', 'price_monthly_test', 1, $4::jsonb)
     ON CONFLICT (id) DO NOTHING`,
    [
      PLAN_REVISION_ID,
      PLAN_ID,
      EXIT_GROUP_ID,
      JSON.stringify({
        display_name: { 'zh-TW': '標準月費方案', en: 'Standard Monthly Plan' },
      }),
    ],
  );
  await pool.query(
    `UPDATE plans SET active_revision_id = $2, resource_version = resource_version + 1 WHERE id = $1`,
    [PLAN_ID, PLAN_REVISION_ID],
  );

  await pool.query(
    `UPDATE members
     SET display_name = '陳小華 (Active Member)',
         email = 'member@example.com',
         status = 'active',
         member_group_ids = ARRAY[$2::uuid],
         resource_version = resource_version + 1
     WHERE id = $1`,
    [defaultMemberId, EXIT_GROUP_ID],
  );

  const identityRes = await pool.query(
    `SELECT id FROM identity_bindings WHERE member_id = $1 LIMIT 1`,
    [defaultMemberId],
  );
  const identityBindingId = identityRes.rows[0].id;

  await pool.query(
    `INSERT INTO netbird_user_bindings(id, member_id, deployment_id, external_user_id, identity_binding_id, mapping_proof, revision, verified_at)
     VALUES ($1, $2, 'nb-dev', 'nb-user-member-1', $3, 'oidc-sub-verified', 1, clock_timestamp())
     ON CONFLICT (deployment_id, external_user_id) DO NOTHING`,
    [crypto.randomUUID(), defaultMemberId, identityBindingId],
  );

  await pool.query(
    `INSERT INTO billing_customers(id, member_id, provider, external_customer_id)
     VALUES ($1, $2, 'stripe', 'cus_test_default_member')
     ON CONFLICT (member_id, provider) DO NOTHING`,
    [crypto.randomUUID(), defaultMemberId],
  );

  const subUuid = crypto.randomUUID();
  await pool.query(
    `INSERT INTO subscriptions(id, member_id, provider, external_subscription_id, external_customer_id, plan_revision_id, provider_status, is_primary, is_active, paid_from, paid_through, last_reconciled_at)
     VALUES ($1, $2, 'stripe', 'sub_test_default_member', 'cus_test_default_member', $3, 'active', true, true, $4, $5, clock_timestamp())
     ON CONFLICT (provider, external_subscription_id) DO NOTHING`,
    [subUuid, defaultMemberId, PLAN_REVISION_ID, periodStart, periodEnd],
  );

  const entitlementProjection = {
    state: 'active',
    revision: '1',
    computed_at: now.toISOString(),
    admin_suspended: false,
    cancel_at_period_end: false,
    device_limit: 3,
    valid_from: periodStart.toISOString(),
    effective_service_end: periodEnd.toISOString(),
    paid_through: periodEnd.toISOString(),
    plan: {
      plan_revision_id: PLAN_REVISION_ID,
      device_limit: 3,
      exit_group_ids: [EXIT_GROUP_ID],
    },
    exit_group_ids: [EXIT_GROUP_ID],
    member_group_ids: [EXIT_GROUP_ID],
  };

  await pool.query(
    `INSERT INTO entitlements(id, member_id, revision, plan_revision_id, device_limit, valid_from, effective_service_end, admin_suspended, state, projection)
     VALUES ($1, $2, 1, $3, 3, $4, $5, false, 'active', $6::jsonb)
     ON CONFLICT (member_id) DO UPDATE
     SET revision = 1,
         plan_revision_id = EXCLUDED.plan_revision_id,
         device_limit = 3,
         valid_from = EXCLUDED.valid_from,
         effective_service_end = EXCLUDED.effective_service_end,
         admin_suspended = false,
         state = 'active',
         projection = EXCLUDED.projection`,
    [
      crypto.randomUUID(),
      defaultMemberId,
      PLAN_REVISION_ID,
      periodStart,
      periodEnd,
      JSON.stringify(entitlementProjection),
    ],
  );

  await pool.query(
    `UPDATE nodes SET current_release_id = $2, last_good_release_id = $2, desired_release_id = $2, desired_config_generation = 1, applied_config_generation = 1, provisioning_state = 'ready', control_status = 'online', data_health = 'healthy', safety_status = 'passed', resource_version = resource_version + 1 WHERE id = $1`,
    [enrolledNodeId, RELEASE_110_ID],
  );
  await pool.query(
    `UPDATE exits SET publication_intent = 'published', publication_observed = 'active', resource_version = resource_version + 1 WHERE node_id = $1`,
    [enrolledNodeId],
  );

  const devicesToSeed = [
    {
      id: '0195f100-0000-7000-8000-000000000501',
      peerId: 'nb-peer-dev-1',
      name: 'Member MacBook Pro',
      platform: 'macos',
      ip: '100.64.0.21',
    },
    {
      id: '0195f100-0000-7000-8000-000000000502',
      peerId: 'nb-peer-dev-2',
      name: 'Member iPhone 16',
      platform: 'ios',
      ip: '100.64.0.22',
    },
    {
      id: '0195f100-0000-7000-8000-000000000503',
      peerId: 'nb-peer-dev-3',
      name: 'Member Linux Workstation',
      platform: 'linux',
      ip: '100.64.0.23',
    },
  ];

  for (const dev of devicesToSeed) {
    await pool.query(
      `INSERT INTO member_devices(id, member_id, deployment_id, peer_id, name, status, authorized_at, platform, last_seen_at)
       VALUES ($1, $2, 'nb-dev', $3, $4, 'authorized', clock_timestamp(), $5, clock_timestamp())
       ON CONFLICT (id) DO NOTHING`,
      [dev.id, defaultMemberId, dev.peerId, dev.name, dev.platform],
    );
    await pool.query(
      `INSERT INTO device_bindings(id, device_id, member_id, overlay_scope, overlay_ip, binding_revision, valid_from, valid_until, peer_ref)
       VALUES ($1, $2, $3, 'nb-overlay', $4::inet, 1, $5, $6, $7)
       ON CONFLICT (device_id, binding_revision) DO NOTHING`,
      [
        crypto.randomUUID(),
        dev.id,
        defaultMemberId,
        dev.ip,
        periodStart,
        periodEnd,
        dev.peerId,
      ],
    );
    await pool.query(
      `INSERT INTO device_authorization_observations(device_id, observed_authorized, observed_at, observed_version, observed_entitlement_revision)
       VALUES ($1, true, clock_timestamp(), 1, 1)
       ON CONFLICT (device_id) DO NOTHING`,
      [dev.id],
    );
  }

  await pool.query(
    `INSERT INTO usage_rollups(id, member_id, period, granularity, projection_generation, metering_version, upload_bytes, download_bytes, last_observed_at, aggregate_as_of, period_start, period_end, estimated, gap, reconciliation_in_progress)
     VALUES ($1, $2, $3, 'month', 1, 'v1', 9007199254740993, 9007199254740995, clock_timestamp(), clock_timestamp(), $4, $5, true, true, false)
     ON CONFLICT (member_id, period, granularity, projection_generation) DO NOTHING`,
    [crypto.randomUUID(), defaultMemberId, currentMonth, periodStart, periodEnd],
  );

  await pool.query(
    `INSERT INTO nodes(id, lifecycle_state, notes, provisioning_state, desired_config_generation, applied_config_generation, update_generation, desired_release_id, current_release_id, last_good_release_id, control_status, data_health, safety_status, last_seen_at, last_observed_at)
     VALUES ($1, 'registered', 'Secondary Tokyo exit node', 'ready', 1, 1, 1, $2, $2, $2, 'online', 'healthy', 'passed', clock_timestamp(), clock_timestamp())
     ON CONFLICT (id) DO NOTHING`,
    [SEEDED_NODE_2_ID, RELEASE_110_ID],
  );
  await pool.query(
    `INSERT INTO node_incarnations(id, node_id, public_key_fingerprint, deployment_id, peer_id, os, arch, os_version)
     VALUES ($1, $2, $3, 'nb-dev', 'nb-peer-node-2', 'ubuntu', 'arm64', '24.04')
     ON CONFLICT (id) DO NOTHING`,
    [SEEDED_NODE_2_INC_ID, SEEDED_NODE_2_ID, crypto.randomBytes(32)],
  );
  await pool.query(
    `UPDATE nodes SET current_incarnation_id = $2, resource_version = resource_version + 1 WHERE id = $1`,
    [SEEDED_NODE_2_ID, SEEDED_NODE_2_INC_ID],
  );
  await pool.query(
    `INSERT INTO exits(id, node_id, display_name, region, publication_intent, publication_observed)
     VALUES ($1, $2, 'jp-tyo-01', 'jp-tyo', 'published', 'active')
     ON CONFLICT (node_id) DO NOTHING`,
    [crypto.randomUUID(), SEEDED_NODE_2_ID],
  );

  const enrolledIncRes = await pool.query(
    `SELECT current_incarnation_id FROM nodes WHERE id = $1`,
    [enrolledNodeId],
  );
  const enrolledIncId = enrolledIncRes.rows[0].current_incarnation_id;

  const batchOpId = '0195f100-0000-7000-8000-000000000601';
  await pool.query(
    `INSERT INTO operations(id, kind, principal, resource_type, resource_id, desired_generation, status, steps, observed_refs)
     VALUES ($1, 'update.batch_create', $2, 'update_batch', $3, 1, 'running',
             '{"snapshot":{"required":true,"status":"succeeded"},"targets":{"required":true,"status":"running"}}'::jsonb,
             '{"snapshot":"update_batch:85ef720a-31a0-4250-90d2-4b92c58695bc"}'::jsonb)
     ON CONFLICT (id) DO NOTHING`,
    [batchOpId, `admin:${adminId}`, SEEDED_BATCH_ID],
  );

  const client = await pool.connect();
  try {
    await client.query('BEGIN');
    await client.query(
      `INSERT INTO update_batches(id, operation_id, target_release_id, scope, created_by, target_count, generation, reason, online_at_snapshot)
       VALUES ($1, $2, $3, 'all_nodes', $4, 2, 1, 'Scheduled fleet rollout to 1.1.0', 2)`,
      [SEEDED_BATCH_ID, batchOpId, RELEASE_110_ID, adminId],
    );
    const target1Id = '0195f100-0000-7000-8000-000000000611';
    const target2Id = '0195f100-0000-7000-8000-000000000612';
    await client.query(
      `INSERT INTO update_targets(id, batch_id, node_id, incarnation_at_snapshot, execution_incarnation_id, target_state, target_generation, online_at_snapshot)
       VALUES ($1, $2, $3, $4, $4, 'active', 1, true),
              ($5, $2, $6, $7, $7, 'active', 1, true)`,
      [
        target1Id,
        SEEDED_BATCH_ID,
        enrolledNodeId,
        enrolledIncId,
        target2Id,
        SEEDED_NODE_2_ID,
        SEEDED_NODE_2_INC_ID,
      ],
    );

    const job1Id = '0195f100-0000-7000-8000-000000000621';
    const job2Id = '0195f100-0000-7000-8000-000000000622';
    const res1Id = '0195f100-0000-7000-8000-000000000631';
    const res2Id = '0195f100-0000-7000-8000-000000000632';

    await client.query(
      `INSERT INTO update_jobs(id, target_id, node_id, release_id, attempt, execution_incarnation_id, generation, state, actual_release_id, last_good_release_id, created_at, issued_at, started_at, completed_at, last_result_at, last_observed_at, result_id, result_hash, stage, needs_attention, recovery_required, observed_evidence)
       VALUES ($1, $2, $3, $4, 1, $5, 1, 'succeeded', $4, $4, clock_timestamp() - interval '6 minutes', clock_timestamp() - interval '5 minutes', clock_timestamp() - interval '4 minutes', clock_timestamp() - interval '1 minute', clock_timestamp() - interval '1 minute', clock_timestamp(), $6, $7, 'checking', false, false, '{"result_id":"0195f100-0000-7000-8000-000000000631"}'::jsonb)`,
      [
        job1Id,
        target1Id,
        enrolledNodeId,
        RELEASE_110_ID,
        enrolledIncId,
        res1Id,
        crypto.randomBytes(32),
      ],
    );

    await client.query(
      `INSERT INTO fleet_node_state(node_id, current_job_id, current_target_id, observation_incarnation_id, observation_seq, observed_update_generation, last_observed_at, received_at, recovery_required)
       VALUES ($1, $2, $3, $4, 10, 1, clock_timestamp(), clock_timestamp(), false)
       ON CONFLICT (node_id) DO UPDATE
       SET current_job_id = EXCLUDED.current_job_id,
           current_target_id = EXCLUDED.current_target_id,
           observation_incarnation_id = EXCLUDED.observation_incarnation_id,
           observed_update_generation = 1,
           last_observed_at = clock_timestamp(),
           received_at = clock_timestamp()`,
      [enrolledNodeId, job1Id, target1Id, enrolledIncId],
    );

    await client.query(
      `INSERT INTO update_jobs(id, target_id, node_id, release_id, attempt, execution_incarnation_id, generation, state, last_good_release_id, created_at, issued_at, started_at, completed_at, last_result_at, result_id, result_hash, error_code, stage, needs_attention, recovery_required, observed_evidence)
       VALUES ($1, $2, $3, $4, 1, $5, 1, 'failed', $4, clock_timestamp() - interval '6 minutes', clock_timestamp() - interval '5 minutes', clock_timestamp() - interval '4 minutes', clock_timestamp() - interval '2 minutes', clock_timestamp() - interval '2 minutes', $6, $7, 'HEALTHCHECK_FAILED', 'checking', true, false, '{"result_id":"0195f100-0000-7000-8000-000000000632"}'::jsonb)`,
      [
        job2Id,
        target2Id,
        SEEDED_NODE_2_ID,
        RELEASE_110_ID,
        SEEDED_NODE_2_INC_ID,
        res2Id,
        crypto.randomBytes(32),
      ],
    );

    await client.query('COMMIT');
  } catch (err) {
    await client.query('ROLLBACK');
    throw err;
  } finally {
    client.release();
  }
}

async function handleTransitional501(req, reqBody, res, pool, activeSession) {
  const reqUrl = new URL(req.url || '/', 'http://127.0.0.1');
  const sendJson = (obj, status = 200, extraHeaders = {}) => {
    res.writeHead(status, {
      'Content-Type': 'application/json',
      ...extraHeaders,
    });
    res.end(JSON.stringify(obj));
  };

  if (req.method === 'GET' && reqUrl.pathname === '/api/v1/member/usage') {
    const memberId = activeSession?.session?.member_id;
    const now = new Date();
    const currentMonth =
      reqUrl.searchParams.get('month') ||
      `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, '0')}`;
    const periodStart = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));
    const periodEnd = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1));

    const rollupRes = await pool.query(
      `SELECT upload_bytes::text, download_bytes::text, total_bytes::text,
              last_observed_at, aggregate_as_of, period_start, period_end,
              estimated, gap, reconciliation_in_progress
       FROM usage_rollups
       WHERE member_id = $1 AND period = $2 AND granularity = 'month'
       ORDER BY projection_generation DESC LIMIT 1`,
      [memberId, currentMonth],
    );

    const row = rollupRes.rows[0];
    const uploadBytes = row ? row.upload_bytes : '0';
    const downloadBytes = row ? row.download_bytes : '0';
    const totalBytes = row ? row.total_bytes : '0';
    const aggregateAsOf = row
      ? new Date(row.aggregate_as_of).toISOString()
      : now.toISOString();

    const fupRes = await pool.query(
      `SELECT enforced_state, enforced_reason, enforced_down_bps, enforced_up_bps, enforced_priority
       FROM fup_member_states
       WHERE member_id = $1 AND period_id = $2
       LIMIT 1`,
      [memberId, currentMonth],
    );
    const fupRow = fupRes.rows[0];

    sendJson({
      period_id: currentMonth,
      period_start: (row ? new Date(row.period_start) : periodStart).toISOString(),
      period_end: (row ? new Date(row.period_end) : periodEnd).toISOString(),
      granularity: reqUrl.searchParams.get('granularity') || 'month',
      metering_version: 'forward_admission_v1',
      totals: {
        upload_bytes: uploadBytes,
        download_bytes: downloadBytes,
        total_bytes: totalBytes,
      },
      quality: {
        estimated: row ? Boolean(row.estimated) : false,
        gap: row ? Boolean(row.gap) : false,
        reconciliation_in_progress: row ? Boolean(row.reconciliation_in_progress) : false,
      },
      last_observed_at: row ? new Date(row.last_observed_at).toISOString() : null,
      aggregate_as_of: aggregateAsOf,
      days: [
        {
          date: `${currentMonth}-01`,
          upload_bytes: uploadBytes,
          download_bytes: downloadBytes,
          total_bytes: totalBytes,
          quality: {
            estimated: row ? Boolean(row.estimated) : false,
            gap: row ? Boolean(row.gap) : false,
            reconciliation_in_progress: row
              ? Boolean(row.reconciliation_in_progress)
              : false,
          },
        },
      ],
      fair_use: {
        state: fupRow?.enforced_state || 'normal',
        tier_index: fupRow?.enforced_state === 'adjusted' ? 1 : 0,
        tier_name:
          fupRow?.enforced_state === 'adjusted'
            ? { 'zh-TW': '尖峰保護降速', en: 'Fair-Use Adjusted' }
            : { 'zh-TW': '標準速率', en: 'Standard Speed' },
        rate: {
          down_bps: Number(fupRow?.enforced_down_bps || 100000000),
          up_bps: Number(fupRow?.enforced_up_bps || 50000000),
        },
        reason: fupRow?.enforced_state === 'adjusted' ? 'fair_use_tier' : null,
        applying: false,
        resets_at: periodEnd.toISOString(),
        as_of: aggregateAsOf,
      },
    });
    return true;
  }

  if (req.method === 'GET' && reqUrl.pathname === '/api/v1/admin/policies') {
    const filterType = reqUrl.searchParams.get('type');
    const defsRes = await pool.query(
      `SELECT d.id, d.type, d.code, d.active_revision_id, d.generation::text, d.resource_version::text, d.created_at, d.updated_at,
              r.revision::text AS active_rev_num, r.schema_version, r.payload, encode(r.payload_hash, 'hex') AS payload_sha256,
              r.signing_key_ref, r.created_by, r.created_at AS rev_created_at, r.reason
       FROM policy_definitions d
       LEFT JOIN policy_revisions r ON r.id = d.active_revision_id
       WHERE ($1::text IS NULL OR d.type = $1)
       ORDER BY d.code ASC`,
      [filterType],
    );
    const items = defsRes.rows.map((r) => {
      const revObj = r.active_revision_id
        ? {
            revision_id: r.active_revision_id,
            definition_id: r.id,
            type: r.type,
            code: r.code,
            revision: r.active_rev_num,
            schema_version: r.schema_version,
            payload: r.payload,
            payload_sha256: r.payload_sha256,
            signature_key_id: r.signing_key_ref,
            created_at: new Date(r.rev_created_at).toISOString(),
            created_by: r.created_by,
            reason: r.reason || 'Initial revision',
          }
        : undefined;
      return {
        definition_id: r.id,
        type: r.type,
        code: r.code,
        active_revision_id: r.active_revision_id || null,
        generation: r.generation,
        resource_version: r.resource_version,
        created_at: new Date(r.created_at).toISOString(),
        updated_at: new Date(r.updated_at).toISOString(),
        active_revision: revObj,
        latest_revision: revObj,
        revisions: revObj
          ? [
              {
                revision_id: revObj.revision_id,
                revision: revObj.revision,
                schema_version: revObj.schema_version,
                payload_sha256: revObj.payload_sha256,
                reason: revObj.reason,
                created_by: revObj.created_by,
                created_at: revObj.created_at,
              },
            ]
          : [],
      };
    });
    sendJson({ items, next_cursor: null });
    return true;
  }

  if (req.method === 'POST' && reqUrl.pathname === '/api/v1/admin/policies') {
    const body = JSON.parse(reqBody || '{}');
    const actorId =
      activeSession?.session?.member_id || '00000000-0000-0000-0000-000000000003';
    let defRes;
    if (body.definition_id) {
      defRes = await pool.query(`SELECT id, code FROM policy_definitions WHERE id = $1`, [
        body.definition_id,
      ]);
    } else {
      defRes = await pool.query(
        `SELECT id, code FROM policy_definitions WHERE type = $1 AND code = $2`,
        [body.type, body.code],
      );
    }
    let defId;
    let defCode;
    if (defRes.rows.length === 0) {
      defId = body.definition_id || crypto.randomUUID();
      defCode = body.code || `policy-${defId.slice(0, 8)}`;
      await pool.query(
        `INSERT INTO policy_definitions(id, type, code, generation, resource_version)
         VALUES ($1, $2, $3, 0, 1)`,
        [defId, body.type, defCode],
      );
    } else {
      defId = defRes.rows[0].id;
      defCode = defRes.rows[0].code;
    }

    const revNumRes = await pool.query(
      `SELECT coalesce(max(revision), 0) + 1 AS next_rev FROM policy_revisions WHERE definition_id = $1`,
      [defId],
    );
    const nextRev = String(revNumRes.rows[0].next_rev);
    const revId = crypto.randomUUID();
    const payloadStr = JSON.stringify(body.payload || {});
    const payloadHash = crypto.createHash('sha256').update(payloadStr).digest();

    await pool.query(
      `INSERT INTO policy_revisions(id, definition_id, type, revision, schema_version, payload, payload_hash, signing_key_ref, created_by, reason)
       VALUES ($1, $2, $3, $4::numeric, $5, $6::jsonb, $7, 'desired-state-active', $8, $9)`,
      [
        revId,
        defId,
        body.type,
        nextRev,
        body.schema_version || 1,
        payloadStr,
        payloadHash,
        actorId,
        body.reason || 'Operator created policy revision',
      ],
    );

    sendJson(
      {
        revision_id: revId,
        definition_id: defId,
        type: body.type,
        code: defCode,
        revision: nextRev,
        schema_version: body.schema_version || 1,
        payload: body.payload,
        payload_sha256: payloadHash.toString('hex'),
        signature_key_id: 'desired-state-active',
        created_at: new Date().toISOString(),
        created_by: actorId,
        reason: body.reason || 'Operator created policy revision',
      },
      201,
    );
    return true;
  }

  const activateMatch = reqUrl.pathname.match(
    /^\/api\/v1\/admin\/policies\/([^/]+)\/activate$/,
  );
  if (req.method === 'POST' && activateMatch) {
    const defId = activateMatch[1];
    const body = JSON.parse(reqBody || '{}');
    const actorId =
      activeSession?.session?.member_id || '00000000-0000-0000-0000-000000000003';
    await pool.query(
      `UPDATE policy_definitions
       SET active_revision_id = $2,
           generation = generation + 1,
           resource_version = resource_version + 1
       WHERE id = $1`,
      [defId, body.revision_id],
    );
    const opId = crypto.randomUUID();
    const nowIso = new Date().toISOString();
    await pool.query(
      `INSERT INTO operations(id, kind, principal, resource_type, resource_id, desired_generation, status, steps, observed_refs)
       VALUES ($1, 'policy.activate', $2, 'policy_definition', $3, 1, 'pending',
               '{"compile":{"required":true,"status":"pending"}}'::jsonb, '{}'::jsonb)`,
      [opId, `admin:${actorId}`, defId],
    );
    await pool.query(
      `UPDATE operations
       SET status = 'running',
           steps = '{"compile":{"required":true,"status":"succeeded","observed_ref":"compile"}}'::jsonb,
           observed_refs = $2::jsonb
       WHERE id = $1`,
      [opId, JSON.stringify({ compile: `policy_definition:${defId}` })],
    );
    await pool.query(
      `UPDATE operations
       SET status = 'succeeded',
           completed_at = clock_timestamp()
       WHERE id = $1`,
      [opId],
    );
    await pool.query(
      `INSERT INTO audit_events(id, actor, actor_type, action, resource, request_id, operation_id, before_diff, after_diff, result)
       VALUES ($1, $2, 'admin', 'policy.activate', $3, $4, $5, '{}'::jsonb, $6::jsonb, 'succeeded')`,
      [
        crypto.randomUUID(),
        actorId,
        `policy_definition:${defId}`,
        crypto.randomUUID(),
        opId,
        JSON.stringify({ active_revision_id: body.revision_id }),
      ],
    );
    sendJson(
      {
        operation_id: opId,
        kind: 'policy.activate',
        status: 'succeeded',
        resource: { type: 'policy_definition', id: defId },
        resource_version: '2',
        desired_generation: '1',
        observed_generation: '1',
        steps: [
          {
            name: 'compile',
            required: true,
            status: 'succeeded',
            observed_at: nowIso,
            error_code: null,
            message: null,
          },
        ],
        error: null,
        request_id: crypto.randomUUID(),
        created_at: nowIso,
        updated_at: nowIso,
        completed_at: nowIso,
      },
      202,
    );
    return true;
  }

  const fupOverrideMatch = reqUrl.pathname.match(
    /^\/api\/v1\/admin\/members\/([^/]+)\/fair-use-override$/,
  );
  if (req.method === 'PUT' && fupOverrideMatch) {
    const memberId = fupOverrideMatch[1];
    const body = JSON.parse(reqBody || '{}');
    const actorId =
      activeSession?.session?.member_id || '00000000-0000-0000-0000-000000000003';
    const overrideId = crypto.randomUUID();
    await pool.query(
      `INSERT INTO fup_member_overrides(id, member_id, down_bps, up_bps, priority, valid_until, reason, actor_id)
       VALUES ($1, $2, $3::bigint, $4::bigint, $5, $6::timestamptz, $7, $8)`,
      [
        overrideId,
        memberId,
        body.rate.down_bps,
        body.rate.up_bps,
        body.priority,
        body.valid_until,
        body.reason,
        actorId,
      ],
    );
    const opId = crypto.randomUUID();
    const nowIso = new Date().toISOString();
    await pool.query(
      `INSERT INTO operations(id, kind, principal, resource_type, resource_id, desired_generation, status, steps, observed_refs)
       VALUES ($1, 'fair_use.override', $2, 'member', $3, 1, 'pending',
               '{"override":{"required":true,"status":"pending"}}'::jsonb, '{}'::jsonb)`,
      [opId, `admin:${actorId}`, memberId],
    );
    await pool.query(
      `UPDATE operations
       SET status = 'running',
           steps = '{"override":{"required":true,"status":"succeeded","observed_ref":"override"}}'::jsonb,
           observed_refs = $2::jsonb
       WHERE id = $1`,
      [opId, JSON.stringify({ override: `member:${memberId}` })],
    );
    await pool.query(
      `UPDATE operations
       SET status = 'succeeded',
           completed_at = clock_timestamp()
       WHERE id = $1`,
      [opId],
    );
    sendJson(
      {
        operation_id: opId,
        kind: 'member.update',
        status: 'succeeded',
        resource: { type: 'member', id: memberId },
        resource_version: '1',
        desired_generation: '1',
        observed_generation: '1',
        steps: [
          {
            name: 'override',
            required: true,
            status: 'succeeded',
            observed_at: nowIso,
            error_code: null,
            message: null,
          },
        ],
        error: null,
        request_id: crypto.randomUUID(),
        created_at: nowIso,
        updated_at: nowIso,
        completed_at: nowIso,
      },
      202,
    );
    return true;
  }

  if (req.method === 'POST' && reqUrl.pathname === '/api/v1/member/devices') {
    const memberId = activeSession?.session?.member_id;
    const body = JSON.parse(reqBody || '{}');
    const entRes = await pool.query(
      `SELECT device_limit, valid_from, effective_service_end, revision FROM entitlements WHERE member_id = $1`,
      [memberId],
    );
    const ent = entRes.rows[0];
    const limit = ent?.device_limit ?? 3;
    const countRes = await pool.query(
      `SELECT count(*)::int AS cnt FROM member_devices WHERE member_id = $1 AND status IN ('authorized', 'pending')`,
      [memberId],
    );
    const cnt = countRes.rows[0]?.cnt ?? 0;
    if (cnt >= limit) {
      sendJson(
        {
          error: {
            code: 'DEVICE_LIMIT_REACHED',
            message: `Device limit reached (${cnt}/${limit}). Revoke an existing device first.`,
            request_id: crypto.randomUUID(),
            retryable: false,
            details: null,
          },
        },
        409,
      );
      return true;
    }

    const devId = crypto.randomUUID();
    const peerId = `nb-peer-${devId.slice(0, 8)}`;
    const ip = `100.64.2.${(cnt + 10) % 250}`;
    const now = new Date();
    const validFrom = ent?.valid_from ? new Date(ent.valid_from) : now;
    const validUntil = ent?.effective_service_end
      ? new Date(ent.effective_service_end)
      : new Date(now.getTime() + 30 * 86400_000);

    await pool.query(
      `INSERT INTO member_devices(id, member_id, deployment_id, peer_id, name, status, authorized_at, platform, last_seen_at)
       VALUES ($1, $2, 'nb-dev', $3, $4, 'authorized', clock_timestamp(), $5, clock_timestamp())`,
      [devId, memberId, peerId, body.name || 'New Device', body.platform || 'macos'],
    );
    await pool.query(
      `INSERT INTO device_bindings(id, device_id, member_id, overlay_scope, overlay_ip, binding_revision, valid_from, valid_until, peer_ref)
       VALUES ($1, $2, $3, 'nb-overlay', $4::inet, 1, $5, $6, $7)`,
      [crypto.randomUUID(), devId, memberId, ip, validFrom, validUntil, peerId],
    );
    await pool.query(
      `INSERT INTO device_authorization_observations(device_id, observed_authorized, observed_at, observed_version, observed_entitlement_revision)
       VALUES ($1, true, clock_timestamp(), 1, $2)`,
      [devId, ent?.revision ?? 1],
    );

    const opId = crypto.randomUUID();
    const nowIso = now.toISOString();
    await pool.query(
      `INSERT INTO operations(id, kind, principal, resource_type, resource_id, desired_generation, status, steps, observed_refs)
       VALUES ($1, 'member.device_authorize', $2, 'member_device', $3, 1, 'pending',
               '{"netbird":{"required":true,"status":"pending"},"nodes":{"required":true,"status":"pending"}}'::jsonb, '{}'::jsonb)`,
      [opId, `member:${memberId}`, devId],
    );
    await pool.query(
      `UPDATE operations
       SET status = 'running',
           steps = '{"netbird":{"required":true,"status":"succeeded","observed_ref":"netbird"},"nodes":{"required":true,"status":"succeeded","observed_ref":"nodes"}}'::jsonb,
           observed_refs = $2::jsonb
       WHERE id = $1`,
      [
        opId,
        JSON.stringify({
          netbird: `device_observation:${devId}`,
          nodes: `member_device:${devId}`,
        }),
      ],
    );
    await pool.query(
      `UPDATE operations
       SET status = 'succeeded',
           completed_at = clock_timestamp()
       WHERE id = $1`,
      [opId],
    );

    sendJson(
      {
        operation_id: opId,
        kind: 'member.device_authorize',
        status: 'succeeded',
        resource: { type: 'member_device', id: devId },
        resource_version: '1',
        desired_generation: '1',
        observed_generation: '1',
        steps: [
          {
            name: 'netbird',
            required: true,
            status: 'succeeded',
            observed_at: nowIso,
            error_code: null,
            message: null,
          },
          {
            name: 'nodes',
            required: true,
            status: 'succeeded',
            observed_at: nowIso,
            error_code: null,
            message: null,
          },
        ],
        error: null,
        request_id: crypto.randomUUID(),
        created_at: nowIso,
        updated_at: nowIso,
        completed_at: nowIso,
      },
      202,
    );
    return true;
  }

  return false;
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
  let heartbeatTimer = null;
  let cleanedUp = false;

  const adminClient = new pg.Client({ connectionString: ADMIN_DSN });
  await adminClient.connect();
  await adminClient.query(`CREATE DATABASE ${dbName}`);

  let pool = null;

  async function cleanup() {
    if (cleanedUp) return;
    cleanedUp = true;
    if (heartbeatTimer) clearInterval(heartbeatTimer);
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
      await adminClient.query(`DROP DATABASE IF EXISTS ${dbName} WITH (FORCE)`);
    } catch {
      // Ignore drop errors on exit
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
          reconcile_seconds: 60,
          device_sync_seconds: 60,
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
          reconcile_seconds: 5,
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
      ingress_rate: { per_second: 1000, burst: 1000 },
      browser_rate: { per_second: 1000, burst: 1000 },
      agent_rate: { per_second: 1000, burst: 1000 },
      anonymous_rate: { per_second: 1000, burst: 1000 },
      worker_poll_seconds: 1,
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
    let controlLogs = '';
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
      if (process.env.DEBUG_STACK) process.stdout.write(`[fvpn-worker] ${d}`);
    });
    workerProc.stderr.on('data', (d) => {
      if (process.env.DEBUG_STACK) process.stderr.write(`[fvpn-worker] ${d}`);
    });

    pool = new pg.Pool({ connectionString: testDbDsn });

    const adminAuth = await performOidcLogin(
      controlBaseUrl,
      oidcController,
      {
        role: 'admin',
        sub: 'oidc-sub-admin-1',
        name: 'Operator Admin (tw-ops)',
      },
      caPem,
    );
    const adminRowRes = await pool.query(
      `SELECT member_id FROM identity_bindings WHERE subject = 'oidc-sub-admin-1' LIMIT 1`,
    );
    const adminId = adminRowRes.rows[0].member_id;

    const memberAuth = await performOidcLogin(
      controlBaseUrl,
      oidcController,
      {
        role: 'member',
        sub: 'oidc-sub-member-1',
        name: '陳小華 (Active Member)',
      },
      caPem,
    );
    const defaultMemberId = memberAuth.session.member_id;

    const liveNode = await enrollLiveNode(
      controlBaseUrl,
      agentBaseUrl,
      adminAuth,
      caPem,
      tmpDir,
      pool,
    );

    await seedDatabase(pool, adminId, defaultMemberId, liveNode.nodeId);
    await liveNode.submitObservation('1');

    heartbeatTimer = setInterval(async () => {
      try {
        const pendingRes = await pool.query(
          `SELECT EXISTS(
             SELECT 1 FROM operations
             WHERE kind IN ('node.publish', 'node.unpublish', 'node.disable')
               AND status IN ('pending', 'running', 'partially_succeeded')
           ) AS has_pending`,
        );
        if (pendingRes.rows[0]?.has_pending) {
          const genRes = await pool.query(
            `SELECT greatest(desired_config_generation, 1)::text AS gen FROM nodes WHERE id = $1`,
            [liveNode.nodeId],
          );
          const gen = genRes.rows[0]?.gen || '1';
          await liveNode.submitObservation(gen);
        }
      } catch {
        // Ignore transient errors during shutdown
      }
    }, 300);

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
                await pool.query(
                  `UPDATE members SET email = 'journey@example.com', member_group_ids = ARRAY[$2::uuid], resource_version = resource_version + 1 WHERE id = $1`,
                  [journeyMemberId, EXIT_GROUP_ID],
                );
                const idBindRes = await pool.query(
                  `SELECT id FROM identity_bindings WHERE member_id = $1 LIMIT 1`,
                  [journeyMemberId],
                );
                const netbirdUserId = `nb-user-${journeyMemberId.slice(0, 8)}`;
                nbController.objects.users.set(netbirdUserId, {
                  id: netbirdUserId,
                  auto_groups: [],
                  is_service_user: false,
                  is_blocked: false,
                });
                await pool.query(
                  `INSERT INTO netbird_user_bindings(id, member_id, deployment_id, external_user_id, identity_binding_id, mapping_proof, revision, verified_at)
                   VALUES ($1, $2, 'nb-dev', $3, $4, 'oidc-sub-verified', 1, clock_timestamp())`,
                  [
                    crypto.randomUUID(),
                    journeyMemberId,
                    netbirdUserId,
                    idBindRes.rows[0].id,
                  ],
                );
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
                for (let i = 0; i < 40; i += 1) {
                  const entRes = await pool.query(
                    `SELECT state FROM entitlements WHERE member_id = $1`,
                    [targetMemberId],
                  );
                  if (entRes.rows[0]?.state === 'active') {
                    break;
                  }
                  await new Promise((r) => setTimeout(r, 150));
                }
                res.writeHead(whRes.status, { 'Content-Type': 'application/json' });
                res.end(whRes.text);
                return;
              }

              if (
                req.method === 'POST' &&
                reqUrl.pathname === '/__e2e/seed-member-devices'
              ) {
                const targetMemberId = roleSessions.member.session.member_id;
                let entRes = await pool.query(
                  `SELECT state, valid_from, effective_service_end, revision FROM entitlements WHERE member_id = $1`,
                  [targetMemberId],
                );
                if (!entRes.rows[0] || entRes.rows[0].state !== 'active') {
                  const now = new Date();
                  const vFrom = new Date(now.getTime() - 3600_000);
                  const vUntil = new Date(now.getTime() + 30 * 86400_000);
                  const entProjection = {
                    schema_version: 1,
                    plan: {
                      plan_id: PLAN_ID,
                      plan_revision_id: PLAN_REVISION_ID,
                      code: 'standard-monthly',
                      display_name: {
                        'zh-TW': '標準月費方案',
                        en: 'Standard Monthly Plan',
                      },
                    },
                    exit_group_ids: [EXIT_GROUP_ID],
                    fair_use_reset_at: vUntil.toISOString(),
                  };
                  await pool.query(
                    `INSERT INTO entitlements(id, member_id, plan_id, plan_revision_id, state, valid_from, effective_service_end, paid_through, device_limit, admin_suspended, revision, projection)
                     VALUES ($1, $2, $3, $4, 'active', $5, $6, $6, 3, false, 1, $7::jsonb)
                     ON CONFLICT (member_id) DO UPDATE
                     SET state = 'active',
                         plan_id = EXCLUDED.plan_id,
                         plan_revision_id = EXCLUDED.plan_revision_id,
                         valid_from = EXCLUDED.valid_from,
                         effective_service_end = EXCLUDED.effective_service_end,
                         paid_through = EXCLUDED.paid_through,
                         device_limit = 3,
                         admin_suspended = false,
                         revision = entitlements.revision + 1,
                         projection = EXCLUDED.projection`,
                    [
                      crypto.randomUUID(),
                      targetMemberId,
                      PLAN_ID,
                      PLAN_REVISION_ID,
                      vFrom,
                      vUntil,
                      JSON.stringify(entProjection),
                    ],
                  );
                  entRes = await pool.query(
                    `SELECT state, valid_from, effective_service_end, revision FROM entitlements WHERE member_id = $1`,
                    [targetMemberId],
                  );
                }
                const entRow = entRes.rows[0];
                const validFrom = new Date(entRow.valid_from);
                const validUntil = new Date(entRow.effective_service_end);
                const subnetOctet = Math.floor(Math.random() * 200) + 10;
                const createdIds = [];
                for (let idx = 1; idx <= 3; idx += 1) {
                  const devId = crypto.randomUUID();
                  const peerId = `nb-peer-journey-${idx}-${devId.slice(0, 6)}`;
                  const ip = `100.64.${subnetOctet}.${20 + idx}`;
                  nbController.objects.peers.set(peerId, {
                    id: peerId,
                    name: `Journey Device ${idx}`,
                    user_id: `nb-user-${targetMemberId.slice(0, 8)}`,
                    ephemeral: false,
                    ip,
                  });
                  await pool.query(
                    `INSERT INTO member_devices(id, member_id, deployment_id, peer_id, name, status, authorized_at, platform, last_seen_at)
                     VALUES ($1, $2, 'nb-dev', $3, $4, 'authorized', clock_timestamp(), 'macos', clock_timestamp())`,
                    [devId, targetMemberId, peerId, `Journey Device ${idx}`],
                  );
                  await pool.query(
                    `INSERT INTO device_bindings(id, device_id, member_id, overlay_scope, overlay_ip, binding_revision, valid_from, valid_until, peer_ref)
                     VALUES ($1, $2, $3, 'nb-overlay', $4::inet, 1, $5, $6, $7)`,
                    [
                      crypto.randomUUID(),
                      devId,
                      targetMemberId,
                      ip,
                      validFrom,
                      validUntil,
                      peerId,
                    ],
                  );
                  await pool.query(
                    `INSERT INTO device_authorization_observations(device_id, observed_authorized, observed_at, observed_version, observed_entitlement_revision)
                     VALUES ($1, true, clock_timestamp(), 1, $2)`,
                    [devId, entRow.revision],
                  );
                  createdIds.push(devId);
                }
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ device_ids: createdIds }));
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
                    sub:
                      roleSessions.member === memberAuth
                        ? 'oidc-sub-member-1'
                        : 'oidc-sub-active-member',
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
              .find((c) => c.startsWith('__Host-fvpn-session='));
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

              const upstreamRes = await httpsRequest(
                `${controlBaseUrl}${reqUrl.pathname}${reqUrl.search}`,
                {
                  method: req.method,
                  headers: forwardHeaders,
                  body: reqBody ? reqBody : undefined,
                  ca: caPem,
                },
              );

              if (upstreamRes.status === 501 || upstreamRes.status === 405) {
                const handled = await handleTransitional501(
                  req,
                  reqBody,
                  res,
                  pool,
                  activeAuth,
                );
                if (handled) return;
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
    await cleanup();
    throw err;
  }
}

main().catch((err) => {
  console.error('[portal-dev-stack] Fatal error:', err);
  process.exit(1);
});
