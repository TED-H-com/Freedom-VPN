import assert from 'node:assert/strict';
import test from 'node:test';
import { publicationGateDiagnostics } from './publication-diagnostics.mjs';

test('D-074/D-107: timeout diagnostic preserves code, cause and pending state for both gates', () => {
  for (const name of ['agent_policy', 'data_path_probe']) {
    const op = {
      kind: 'node.publish',
      status: 'running',
      error: null,
      completed_at: null,
      steps: [
        {
          name,
          status: 'pending',
          error_code: 'NODE_NOT_SERVICEABLE',
          message: 'member_forwarding_unconfirmed (lease_unconfirmed)',
        },
        { name: 'netbird_route_readback', status: 'succeeded' },
      ],
    };
    assert.deepEqual(publicationGateDiagnostics(op), [
      {
        ...op.steps[0],
        diagnostic:
          'NODE_NOT_SERVICEABLE: member_forwarding_unconfirmed (lease_unconfirmed)',
      },
    ]);
    op.steps[0].message = null;
    assert.equal(publicationGateDiagnostics(op)[0].diagnostic, 'NODE_NOT_SERVICEABLE');
    assert.equal(publicationGateDiagnostics(op)[0].status, 'pending');
    assert.equal(op.status, 'running');
    assert.equal(op.error, null);
    assert.equal(op.completed_at, null);
    op.kind = 'node.unpublish';
    assert.deepEqual(publicationGateDiagnostics(op), []);
  }
  assert.deepEqual(publicationGateDiagnostics(null), []);
});
