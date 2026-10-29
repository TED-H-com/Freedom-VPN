/**
 * DNS domain normalization, rule validation, and priority match preview (§12.4).
 *
 * Rules from §12.4:
 * - `domain` accepts a domain name only: no scheme (`https://`), port (`:443`),
 *   path (`/foo`), query (`?a=1`), fragment (`#bar`), userinfo (`@`), or wildcards (`*`).
 * - Normalized to lowercase ASCII with trailing root dot `.` removed.
 * - `include_subdomains=true` matches the apex and any subdomain ending on a full label
 *   boundary (`example.com` matches `sub.example.com`, NEVER `notexample.com`).
 * - Priority hierarchy:
 *   Highest: Fixed Internet-only network isolation (cannot be overridden by DNS allowlist)
 *   1: Admin explicit deny (`action === 'block'`)
 *   2: Admin explicit allow (`action === 'allow'`)
 *   3: Enabled threat / category feeds
 *   4: Standard upstream resolution
 */

import type { DnsRule } from '../api/types';

export interface DomainValidationResult {
  valid: boolean;
  canonicalDomain: string;
  errorCode?:
    | 'EMPTY_DOMAIN'
    | 'URL_OR_PATH_NOT_ALLOWED'
    | 'WILDCARD_NOT_ALLOWED'
    | 'INVALID_LABEL';
}

const LABEL_REGEX = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

export function normalizeAndValidateDomain(rawInput: string): DomainValidationResult {
  const trimmed = rawInput.trim();
  if (!trimmed) {
    return { valid: false, canonicalDomain: '', errorCode: 'EMPTY_DOMAIN' };
  }

  if (
    trimmed.includes('://') ||
    trimmed.includes('/') ||
    trimmed.includes('?') ||
    trimmed.includes('#') ||
    trimmed.includes(':') ||
    trimmed.includes('@')
  ) {
    return { valid: false, canonicalDomain: '', errorCode: 'URL_OR_PATH_NOT_ALLOWED' };
  }

  if (trimmed.includes('*')) {
    return { valid: false, canonicalDomain: '', errorCode: 'WILDCARD_NOT_ALLOWED' };
  }

  const withoutTrailingDot = trimmed.endsWith('.') ? trimmed.slice(0, -1) : trimmed;
  let asciiLower = withoutTrailingDot.toLowerCase();

  try {
    // Use standard WHATWG URL host parser for IDNA / punycode normalization when non-ASCII is present
    if (/[^\u0020-\u007f]/.test(asciiLower)) {
      const parsedUrl = new URL(`http://${asciiLower}`);
      asciiLower = parsedUrl.hostname.toLowerCase();
    }
  } catch {
    return { valid: false, canonicalDomain: '', errorCode: 'INVALID_LABEL' };
  }

  if (asciiLower.length === 0 || asciiLower.length > 253) {
    return { valid: false, canonicalDomain: '', errorCode: 'INVALID_LABEL' };
  }

  const labels = asciiLower.split('.');
  if (labels.length < 2) {
    return { valid: false, canonicalDomain: '', errorCode: 'INVALID_LABEL' };
  }

  for (const label of labels) {
    if (!LABEL_REGEX.test(label)) {
      return { valid: false, canonicalDomain: '', errorCode: 'INVALID_LABEL' };
    }
  }

  return { valid: true, canonicalDomain: asciiLower };
}

/**
 * Label-boundary domain match per §12.4.
 * `ruleDomain="example.com"` with `includeSubdomains=true` matches `"example.com"`
 * and `"a.b.example.com"`, but NEVER `"notexample.com"`.
 */
export function matchesDomainRule(
  queryDomain: string,
  ruleDomain: string,
  includeSubdomains: boolean,
): boolean {
  const normQuery = queryDomain.toLowerCase();
  const normRule = ruleDomain.toLowerCase();

  if (normQuery === normRule) {
    return true;
  }
  if (!includeSubdomains) {
    return false;
  }
  return normQuery.endsWith(`.${normRule}`);
}

export interface DnsMatchPreviewResult {
  validQuery: boolean;
  canonicalQuery: string;
  errorCode?: DomainValidationResult['errorCode'];
  outcome: 'blocked_admin' | 'allowed_admin' | 'blocked_threat_feed' | 'resolved_normal';
  priorityLevel: 1 | 2 | 3 | 4;
  matchedRule?: DnsRule;
  isolationNoteApplies: boolean;
}

const SIMULATED_THREAT_FEED_DOMAINS = [
  'phishing-login.example.invalid',
  'malware-drop.example.invalid',
];

export function previewDnsMatch(
  rawQuery: string,
  rules: readonly DnsRule[],
  enableThreatFeed = true,
): DnsMatchPreviewResult {
  const norm = normalizeAndValidateDomain(rawQuery);
  if (!norm.valid) {
    return {
      validQuery: false,
      canonicalQuery: '',
      errorCode: norm.errorCode,
      outcome: 'resolved_normal',
      priorityLevel: 4,
      isolationNoteApplies: false,
    };
  }

  const q = norm.canonicalDomain;
  const isolationNoteApplies =
    q.endsWith('.internal.example.invalid') || q.startsWith('metadata.');

  // Priority 1: Admin explicit deny (block)
  const adminBlock = rules.find(
    (r) => r.action === 'block' && matchesDomainRule(q, r.domain, r.include_subdomains),
  );
  if (adminBlock) {
    return {
      validQuery: true,
      canonicalQuery: q,
      outcome: 'blocked_admin',
      priorityLevel: 1,
      matchedRule: adminBlock,
      isolationNoteApplies,
    };
  }

  // Priority 2: Admin explicit allow
  const adminAllow = rules.find(
    (r) => r.action === 'allow' && matchesDomainRule(q, r.domain, r.include_subdomains),
  );
  if (adminAllow) {
    return {
      validQuery: true,
      canonicalQuery: q,
      outcome: 'allowed_admin',
      priorityLevel: 2,
      matchedRule: adminAllow,
      isolationNoteApplies,
    };
  }

  // Priority 3: Enabled threat feed
  if (
    enableThreatFeed &&
    SIMULATED_THREAT_FEED_DOMAINS.some((td) => matchesDomainRule(q, td, true))
  ) {
    return {
      validQuery: true,
      canonicalQuery: q,
      outcome: 'blocked_threat_feed',
      priorityLevel: 3,
      isolationNoteApplies,
    };
  }

  // Priority 4: Normal resolution
  return {
    validQuery: true,
    canonicalQuery: q,
    outcome: 'resolved_normal',
    priorityLevel: 4,
    isolationNoteApplies,
  };
}

export const normalizeAndValidateDomainInput = normalizeAndValidateDomain;
export const previewDnsDomainMatch = previewDnsMatch;
