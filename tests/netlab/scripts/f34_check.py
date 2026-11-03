"""Recompute D-146/D-147 assertions from retained header/counter evidence."""
import json
from pathlib import Path
import sys

DEVICE = '019a00000000700080000000000000a1'


def check(root, mode, before=False):
    def row(name):
        return json.loads((root/(name+'.json')).read_text())
    def result(r):
        assert r['result']['code'] == 0, r['result']
        return json.loads(r['result']['stdout'])
    def count(r, name):
        return r['counter_delta'].get(name, dict(bytes=0, packets=0))
    def meter(r):
        for direction, prefix, dev in [('upload','u_','fv-ih'), ('download','d_','fv-fh')]:
            c = count(r, prefix+DEVICE)
            assert c == r['forwarded'][direction], (r['name'], direction, c, r['forwarded'][direction])
            leaf = r['tc_delta'][dev].get('1:2f', dict(bytes=0, packets=0, dropped=0))
            assert leaf == dict(bytes=c['bytes']+14*c['packets'], packets=c['packets'], dropped=0), (r['name'], dev, c, leaf)
    def advertised(r, role, synack):
        return sorted({p['mss'] for p in r['mss'][role] if bool(p['flags'] & 16) == synack})
    reports = []
    for direction in (['upload'] if mode == 'router' else ['upload','download']):
        r = row(mode+'-'+direction)
        res = result(r)
        if before and (mode == 'router' or direction == 'download'):
            assert res['bytes'] < 64064 and res['error'] == 'TimeoutError', res
        else:
            assert res['bytes'] == 64064 and not res['error'], res
        meter(r)
        if mode == 'receive':
            expected = {'member_syn': [1380], 'origin_syn': [1380 if before else 1160],
                        'origin_synack': [1460], 'member_synack': [1460 if before else 1160]}
            got = {role+suffix: advertised(r,role,synack) for role in ['member','origin']
                   for suffix,synack in [('_syn',False),('_synack',True)]}
            assert got == expected, (got, expected)
            # No ICMP at any observed hop; no generated PTB silently assists TCP.
            assert not any(r['icmp'].values()), r['icmp']
            sizes = r['outer_receive']['sizes']
            if not before:
                assert sizes and all(int(size)<=1280 and not v['absent'] for size,v in sizes.items()), sizes
            elif direction == 'download':
                assert any(int(size)>1280 and v['absent']>0 for size,v in sizes.items()), sizes
            reports.append(dict(name=r['name'], bytes=res['bytes'], mss=got, outer_receive=r['outer_receive'], metering='exact', classification='1:2f'))
        else:
            errors = [p for p in r['icmp']['node'] if p['direction']=='in' and p['src']=='192.168.34.1' and p['icmp_type']==3 and p['icmp_code']==4]
            delivered = [p for p in r['icmp']['member'] if p['direction']=='in' and p['src']=='192.168.34.1' and p['icmp_type']==3 and p['icmp_code']==4]
            assert errors, 'router generated no fragmentation-needed'
            if before:
                assert not delivered and count(r,'ctdenyd')['packets'] == len(errors)
            else:
                assert res['pmtu']==1000 and len(delivered)==len(errors) and count(r,'ctdenyd')['packets']==0
                # Independent error-only sum is included exactly once in the
                # already checked download IP-byte total (not a diagnostic).
                assert sum(p['ip_length'] for p in delivered) <= count(r,'d_'+DEVICE)['bytes']
            reports.append(dict(name=r['name'], bytes=res['bytes'], learned_pmtu=res['pmtu'], errors=len(errors), delivered_errors=len(delivered),
                                error_bytes=sum(p['ip_length'] for p in delivered), metering='exact', classification='1:2f'))
    if mode == 'receive':
        r=row('receive-small-mss');meter(r)
        assert result(r)['bytes']==64064
        assert advertised(r,'member',False)==[1000] and advertised(r,'origin',False)==[1000]
        r=row('receive-udp');meter(r)
        assert result(r)['bytes']==0 and result(r)['error']=='TimeoutError'
        assert any(int(size)>1280 and v['absent'] for size,v in r['outer_receive']['sizes'].items())
        assert not any(r['icmp'].values())
        reports.append(dict(name='UDP limitation', bytes=0, quic='not tested; no client'))
    else:
        for name,kind,counter in [('router-time-exceeded',11,'ctdenyd'),('router-unsolicited',3,'ctinv')]:
            r=row(name)
            assert count(r,counter)['packets']==1, (name,r['counter_delta'])
            assert count(r,'d_'+DEVICE)['packets']==0
            assert not [p for p in r['icmp']['member'] if p['src']=='192.168.34.1'], name
            assert any(p['icmp_type']==kind for p in r['icmp']['node']), name
            reports.append(dict(name=name, drop_counter=counter, member_deliveries=0))
        if not before:
            r=row('router-denied-upload')
            assert count(r,'ctdeny')['packets']==1 and count(r,'u_'+DEVICE)['packets']==0, r['counter_delta']
            reports.append(dict(name=r['name'], drop_counter='ctdeny', upload_usage=0))
    return reports


if __name__ == '__main__':
    print(json.dumps(check(Path(sys.argv[1]), sys.argv[2], len(sys.argv)>3 and sys.argv[3]=='before'), indent=2))
