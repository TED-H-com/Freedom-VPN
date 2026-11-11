import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';
import { memberScreenEvidenceDirectory } from './member-screen-evidence.mjs';

test('TestAT04_F404_EvidenceCapture_UnsetOrEmptyIsDisabled', () => {
  assert.equal(memberScreenEvidenceDirectory(undefined), undefined);
  assert.equal(memberScreenEvidenceDirectory(''), undefined);
});

test('TestAT04_F404_EvidenceCapture_RelativePathsFailBeforeIO', () => {
  for (const value of ['screens', '../../.agent/screens', './screens', ' ']) {
    assert.throws(
      () => memberScreenEvidenceDirectory(value),
      /PORTAL_E2E_EVIDENCE_DIR must be an absolute directory path/,
    );
  }
});

test('TestAT04_F404_EvidenceCapture_AbsolutePathIsPreserved', () => {
  // Construct a syntax-only path from this module; this test performs no file I/O.
  const directory = new URL('.', import.meta.url).pathname;
  assert.ok(path.isAbsolute(directory));
  assert.equal(memberScreenEvidenceDirectory(directory), directory);
});
