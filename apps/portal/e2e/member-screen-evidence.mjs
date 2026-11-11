import path from 'node:path';

export function memberScreenEvidenceDirectory(value) {
  if (!value) return undefined;
  if (!path.isAbsolute(value)) {
    throw new Error(
      'PORTAL_E2E_EVIDENCE_DIR must be an absolute directory path (or unset/empty to disable capture).',
    );
  }
  return value;
}
