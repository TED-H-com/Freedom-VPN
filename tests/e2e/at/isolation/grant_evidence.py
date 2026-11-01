#!/usr/bin/env python3
"""Independent D-087 receipt/wire audit; a passing counter alone cannot qualify."""
from pathlib import Path
from datetime import datetime
import calendar
import hashlib
import json
from evidence_io import read_json
import sys


def ns(value):
    assert value.endswith('Z')
    base, _, fraction = value[:-1].partition('.')
    return calendar.timegm(datetime.strptime(base,'%Y-%m-%dT%H:%M:%S').timetuple())*10**9 + int(fraction.ljust(9,'0'))


def validate(t, events):
    assert t['recipe_executed'] and t['proof']['invariants_byte_equal']
    f,tr=t['false_flip'],t['true_flip'];p,s=t['provisional_receipt'],t['serving_receipt'];e,l=t['empty_lease'],t['populated_lease']
    assert f['ready'] is False and tr['ready'] is True
    timeline=[f['completed'],e['started'],e['completed'],p['started'],p['completed'],tr['started'],tr['completed'],l['started'],l['completed'],s['started'],s['completed']]
    assert all(ns(a)<=ns(b) for a,b in zip(timeline,timeline[1:]))
    assert ns(f['completed'])<ns(e['started']) and ns(p['completed'])<ns(tr['started']) and ns(tr['completed'])<ns(l['started'])
    assert e['http_status']==l['http_status']==200 and p['http_status']==s['http_status']==202
    for row in [p,s]:
     b=row['request'];assert b['provisioning_state']=='ready' and b['safety_status']=='passed'
    b=p['request'];assert b['member_forwarding_enabled'] is False and b['forwarding_block_reasons']==['awaiting_member_grants']
    assert len(b['errors'])==1 and b['errors'][0]['code']=='AWAITING_MEMBER_GRANTS' and b['errors'][0]['stage']=='lease'
    checks={c['name']:c for c in b['checks']}
    assert all(checks[n]['error_code']=='AWAITING_MEMBER_GRANTS' for n in ['lease','qos','source_binding'])
    assert s['request']['member_forwarding_enabled'] and not s['request']['forwarding_block_reasons']
    assert p['request']['report_id']!=s['request']['report_id']
    assert e['response']['signed']['lease_id']!=l['response']['signed']['lease_id']
    assert e['response']['signed']['members']==[] and l['response']['signed']['members']
    g=t['before']['node'][3]
    for report,lease in [(p,e),(s,l)]:
     assert report['request']['lease']['lease_id']==lease['response']['signed']['lease_id']
     assert report['request']['applied_config_generation']==lease['response']['signed']['config_generation']==g
     assert lease['response']['signatures']
    assert t['before']==t['provisional_invariants']==t['after']
    assert t['before']['offline_seconds']==60
    for name in ['initial_counted_forwarding','counted_forwarding']:
     v=t[name];assert v['counted'] and int(v['admission_bytes_after'])>int(v['admission_bytes_before'])
    count=0
    for row in events:
     for key in ['request','response']:
      if row.get(key+'_wire'):
       count+=1;raw=row[key+'_wire'];assert hashlib.sha256(raw.encode()).hexdigest()==row[key+'_sha256']
       assert json.loads(raw)==row[key]
    # The HTTP desired document and ETag are independently held constant throughout.
    desired=[v for v in events if v['kind']=='desired' and v.get('http_status') in (200,304) and ns(f['completed'])<=ns(v['started'])<=ns(s['completed'])]
    assert desired
    baseline=[v for v in events if v['kind']=='desired' and v.get('http_status')==200 and ns(v['completed'])<ns(f['started'])][-1]
    assert all(v['etag']==baseline['etag'] for v in desired)
    assert all(v['http_status']==304 or v['response_wire']==baseline['response_wire'] for v in desired)
    assert baseline['etag']==t['before']['envelope']['etag']
    assert baseline['response']==t['before']['envelope']['envelope']
    for event in [f,tr,p,s,e,l]:
     assert event in events
    assert int(t['proof']['true_to_serving_receipt_ms'])==(ns(s['completed'])-ns(tr['completed']))//1000000
    return dict(run_id=t['run_id'], status='pass', original_wire_hashes=count,
                desired_responses_in_window=len(desired), proof=t['proof'])


if __name__ == '__main__':
    root=Path(sys.argv[1] if len(sys.argv)>1 else 'docs/e2e/at/isolation')
    result=validate(read_json(root/'member-grants-transition.json'),
                    read_json(root/'member-grants-wire.json'))
    print(json.dumps(result,indent=2))
