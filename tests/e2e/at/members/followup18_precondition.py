"""Retain the original finite-grant transition and later AT09 coverage gap."""
from pathlib import Path
import hashlib
import json

from evidence_io import load

ROOT = Path('docs/e2e/at/members')


def main():
    a, b = (load(ROOT/(name+'.json')) for name in ('AT02', 'AT09'))
    assert a['run_id'] == b['run_id']
    index, transition = [(i, s) for i, s in enumerate(a['steps'])
                         if s['name'] == 'finite grant production transition'][-1]
    value = json.loads(transition['actual'])['transition']
    reauth = next(s for s in b['steps'] if s['name'] == 'AT09 member OIDC reauthentication')
    member = json.loads(reauth['actual'])['member_id']
    assert member == value['member_id']
    sources = ['tests/e2e/at/members/finite_grant_test.go',
               'tests/e2e/at/members/forwarding_test.go',
               'tests/e2e/at/members/native_matrix_test.go']
    result = dict(run_id=a['run_id'], source_commit=a['source_commit'], member_id=member,
        original_finite_transition=transition, valid_until=value['valid_until'],
        subsequent_readiness_checks=[s for s in a['steps'][index+1:]
                                    if s['name'] == 'populated-grant serving readiness'],
        restoration_steps=[s for s in a['steps'][index+1:]
                           if s['name'].startswith('finite expiry restor')],
        AT09_reauthentication=reauth,
        AT09_visible_exits=next(s for s in b['steps'] if s['name'] == 'member sees both published exits'),
        AT09_original_forwarding=b.get('forwarding', []),
        AT09_original_blockers=b.get('blockers', []),
        AT09_original_member_responses=b.get('member_http', []),
        AT02_original_blockers=a.get('blockers', []),
        source_sha256={p: hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in sources},
        source_path_explanation='finiteExpiry installs a 75-second grant and returns on a failed positive trace before its normal restoration. Successful paths continue to the restoration steps retained below. nativeMatrix returns before applying inputs only when awaitDevice yields no admitted device.',
        interpretation='Same member UUID joins the accepted finite transition to the later AT09 session. Restoration steps, member response and original blockers below determine whether the prerequisite remained available. Do not infer a missing restoration or unexercised matrix when the original rows show restoration and executed inputs.',
        limits='The early-return explanation combines original records with the measured source path; no newly queried database state or synthetic expiry observation is substituted for a missing original check.')
    (ROOT/'F18-AT09-precondition.json').write_text(json.dumps(result, indent=2)+'\n')
    print('AT09 member join:', member, 'grant deadline:', value['valid_until'],
          'restoration records:', len(result['restoration_steps']))


if __name__ == '__main__':
    main()
