#!/usr/bin/env python3
"""Auxiliary same-run clone chain. Never changes AT scoring or expected gaps."""
import json
from pathlib import Path
import re
import shlex
from auxiliary_inputs import inputs
from candidate_evidence import wall_ns
from evidence_io import read_json, redact_error, redact_text

SOURCE = 'fvpn-w4-isolation-node'
CLONE = SOURCE+'-clone'


def heartbeat(text):
    lines = text.splitlines()
    starts = [i for i, line in enumerate(lines) if line == 'v1']
    if len(starts) != 1:
        raise ValueError('one heartbeat required')
    fields = {}
    for line in lines[starts[0]+1:]:
        key, _, value = line.partition(' ')
        if key not in ('boot_id', 'mono_ns', 'wall_unix_ns', 'policy_expires_mono_ns',
                       'baseline_ok', 'serving', 'generation', 'reason', 'evidence'):
            break
        if key in fields or not value:
            raise ValueError('duplicate or empty heartbeat field')
        fields[key] = value
    for key in ('mono_ns', 'wall_unix_ns', 'policy_expires_mono_ns', 'generation'):
        if not re.fullmatch(r'[0-9]+', fields.get(key, '')):
            raise ValueError('invalid heartbeat '+key)
    if fields.get('serving') not in ('0', '1') or fields.get('baseline_ok') not in ('0', '1') or not fields.get('reason') or not fields.get('boot_id'):
        raise ValueError('incomplete heartbeat')
    return fields


def log_rows(read):
    rows, errors = [], []
    if read.get('status') != 'available':
        errors.append('log '+read.get('status', 'unavailable'))
    for number, line in enumerate(read.get('text', '').splitlines(), 1):
        if not line.strip():
            continue
        try:
            row = dict(word.split('=', 1) for word in shlex.split(line) if '=' in word)
            if not row.get('time') or not row.get('msg'):
                raise ValueError('not a structured agent log row')
            wall_ns(row['time'])
            row['line'] = number
            rows.append(row)
        except (ValueError, TypeError):
            errors.append('unparsed log line '+str(number))
    return rows, errors


def frozen_heartbeat(row):
    beat = heartbeat(row['readback'])
    view = row.get('forward_view', {})
    return (beat['reason'] == 'identity_frozen' and beat['serving'] == '0' and
            beat['policy_expires_mono_ns'] == '0' and beat['baseline_ok'] == '1' and
            view.get('Baseline') is True and view.get('Policy') == 'drop'), beat


def evaluate(manifest, data, stops):
    results, errors = {}, []
    def item(name, ok, evidence, missing):
        results[name] = dict(status='established' if ok else 'not established',
                             evidence=evidence, missing=[] if ok else missing)
    freeze = data.get('identity-freeze.json', {})
    rows = freeze.get('rows', [])
    snapshots = data.get('clone-runtime.json', {}).get('snapshots', [])
    for row in snapshots:
        if any(row.get(k) != manifest.get(k) for k in ('run_id', 'build', 'source_dirty')):
            errors.append('snapshot provenance mismatch')
    clone = [s for s in snapshots if s.get('role') == 'clone' and s.get('container') == CLONE and s.get('status') == 'available']
    created = freeze.get('fixture_created') is True and bool(clone)
    item('clone_created', created, [dict(phase=s['phase'], **s.get('container_identity', {}), read_start=s['read_start'], read_end=s['read_end']) for s in clone], ['successful real fixture and identified clone container'])
    identities = []
    for i, row in enumerate(rows):
        ob = row.get('clone_observation', {})
        source, target = ob.get('Source', {}), ob.get('Clone', {})
        keys = ('machine_id_sha256', 'agent_spki_sha256', 'netbird_public_key_sha256')
        if not row.get('clone_read_error') and ob.get('same_identity') is True and all(re.fullmatch('[0-9a-f]{64}', source.get(k, '')) and source.get(k) == target.get(k) for k in keys):
            identities.append(dict(row=i, read_start=row['read_start'], read_end=row['read_end'], Source=source, Clone=target))
    item('identity_fingerprints_match', bool(identities), identities, ['successful same-identity fingerprint observation'])
    direct = {r.get('container'): r for r in stops.get('stops', []) if r.get('label') == 'direct_agent'}
    stop_ok = (not stops.get('errors') and stops.get('run_id') == manifest['run_id'] and all(direct.get(c, {}).get('final_stop_matched') is True for c in (SOURCE, CLONE)))
    item('final_agent_stop', stop_ok, [direct.get(c, {'container': c, 'status': 'unavailable'}) for c in (SOURCE, CLONE)], ['same-run source/clone launch ID, PID, start-ticks and parent wait receipt joins'])
    runtime, transport, log_diagnostics, snat = [], [], [], []
    for index, row in enumerate(clone):
        snap = row['snapshot']; proc = snap.get('process_before', {})
        receipt = direct.get(CLONE, {}).get('parent_wait_receipt', {})
        config = snap.get('configs', {}).get(proc.get('config_path'), {})
        marker = snap.get('agent_selected', {}).get('text', '').strip()
        expected_exe = '/usr/local/bin/fvpn-agent' if marker == '/usr/local' else marker+'/fvpn-agent'
        expected_config = '/etc/vpn-node/agent-run.json' if marker == '/usr/local' else '/var/lib/vpn-node/e2e/agent-'+marker.rsplit('/', 1)[-1]+'.json'
        identity = (snap.get('process_coherent') is True and proc.get('role') == 'fvpn-agent' and
                    proc.get('executable') == expected_exe and proc.get('command_binary') == expected_exe and
                    proc.get('config_path') == expected_config and config.get('stable_during_snapshot') is True and
                    re.fullmatch('[0-9a-f]{64}', snap.get('executable_sha256', '')))
        if identity:
            runtime.append(dict(snapshot=index, phase=row['phase'], process=proc, executable_sha256=snap['executable_sha256'], config=config,
                                read_start=snap['read_start'], read_end=snap['read_end']))
        logs = snap.get('logs', {})
        parsed, parse_errors = log_rows(logs.get('agent', {}))
        log_diagnostics.append(dict(phase=row['phase'], errors=parse_errors,
                                    logs={k:{a:v for a,v in value.items() if a!='text'} for k,value in logs.items()}))
        try:
            # The matching terminal parent receipt supplies this launch's bounds;
            # inherited lines, old PID reuse and enqueue-only lines cannot qualify.
            matched = (direct.get(CLONE, {}).get('final_stop_matched') is True and
                       receipt['pid'] == proc['pid'] and receipt['start_ticks'] == proc['start_ticks'] and
                       receipt['id'] == snap['agent_process_id']['text'].strip() and
                       wall_ns(receipt['started_at']) <= wall_ns(snap['read_start']) <= wall_ns(snap['read_end']) <= wall_ns(receipt['wait_observed_at']))
            active = [r for r in parsed if wall_ns(receipt['started_at']) <= wall_ns(r['time']) <= wall_ns(snap['read_end']) and
                      wall_ns(row['container_identity']['started_at']) <= wall_ns(r['time'])]
            enqueued = {r.get('report_id') for r in active if r['msg'] == 'observation enqueued'}
            accepted = [r for r in active if r['msg'] == 'observation submission' and r.get('process_id') == str(proc['pid']) and
                        r.get('outcome') == 'accepted' and r.get('http_status') == '202' and
                        r.get('report_ids') == r.get('accepted_report_ids') and r.get('report_ids', '')[1:-1] in enqueued]
        except (KeyError, TypeError, ValueError):
            matched, accepted = False, []
        net = snap.get('network', {})
        socket_read = net.get('transport_sockets', {})
        sockets = socket_read.get('text', '').splitlines()
        proxies = [p for p in snap.get('services', []) if p.get('role') == 'resolver-proxy' and p.get('status') == 'available']
        # Passive socket ownership directly observes the two configured hops.
        # No request is attributed from a central identity row alone.
        agent_socket = any('ESTAB' in s and re.search(r'127\.0\.0\.1:9444\s', s) and 'pid='+str(proc.get('pid'))+',' in s for s in sockets)
        control_host = snap.get('control_host', {})
        proxy_socket = control_host.get('status')=='available' and any(len(s.split())>=5 and s.split()[0]=='ESTAB' and s.split()[4] in [ip+':9443' for ip in control_host.get('addresses', [])] and 'pid='+str(p['pid'])+',' in s for s in sockets for p in proxies)
        listener = net.get('proxy_listener', {})
        proxy_listener = listener.get('status') == 'available' and any('127.0.0.1:9444' in listener.get('text', '') and 'pid='+str(p['pid'])+',' in listener.get('text', '') for p in proxies)
        if identity and matched and not parse_errors and accepted and logs['agent'].get('inode') == proc.get('stdout_inode') and config.get('fields', {}).get('control_url') == 'https://127.0.0.1:9444' and socket_read.get('status') == 'available' and agent_socket and proxy_socket and proxy_listener:
            transport.append(dict(snapshot=index, phase=row['phase'], accepted_submissions=accepted,
                                  socket_read=socket_read, proxy_listener=listener, proxy_processes=proxies, control_host=control_host,
                                  process=proc, launch_id=receipt['id'], config=config))
        address_read = net.get('addresses', {})
        try:
            addresses = json.loads(address_read['text'])
        except (KeyError, TypeError, ValueError):
            addresses = None
        snat.append(dict(phase=row['phase'], configs=snap.get('configs'), addresses=addresses,
                         addresses_status=address_read.get('status', 'unavailable'), watchdog_processes=[p for p in snap.get('services', []) if p.get('role') == 'fvpn-watchdog'],
                         scope='assigned addresses and current candidate SNAT values only; no member-forwarding validation or causal attribution'))
    item('executing_release_config', bool(runtime), runtime, ['coherent live process, approved argv/config, executable digest and stable file reads'])
    item('corrected_transport', bool(transport), transport, ['same-launch freshly enqueued accepted clone report, matching log inode, live agent-to-9444 and proxy-to-9443 socket owners and listener; static URL/central rows alone are insufficient'])
    central = [dict(row=i, observation=r['clone_observation'], read_start=r['read_start'], read_end=r['read_end']) for i,r in enumerate(rows) if not r.get('clone_read_error') and r.get('clone_observation', {}).get('conflicts', 0)>0 and r['clone_observation'].get('mapping_frozen') is True]
    item('central_conflict_frozen', bool(central), central, ['successful central conflict count and mapping_frozen observation'])
    source_wire = []
    for i, transaction in enumerate(data.get('receiver-capture.json', {}).get('transactions', [])):
        body, response = transaction.get('body') or {}, transaction.get('response') or {}
        try:
            in_window = wall_ns(freeze['started_at']) <= wall_ns(transaction['received_at']) <= wall_ns(transaction['response_at']) <= wall_ns(manifest['completed_at'])
        except (KeyError, TypeError, ValueError):
            in_window = False
        if (in_window and transaction.get('path') == '/api/v1/agent/observations' and transaction.get('status') == 202 and
            response.get('accepted') is True and response.get('identity_frozen') is True and
            body.get('report_id') and response.get('report_id') == body['report_id']):
            source_wire.append(dict(transaction=i, received_at=transaction['received_at'], response_at=transaction['response_at'],
                                    response=response, report={k:body.get(k) for k in ('report_id','observed_at','member_forwarding_enabled','applied_config_generation','errors')},
                                    consumer=SOURCE, provenance='source-only 9445 receiver; never attributed to clone'))
    source_report = [r for r in source_wire if r['report']['member_forwarding_enabled'] is False and
                     r['report']['applied_config_generation'] is None and any(e.get('stage') == 'identity' and e.get('message') == 'identity_frozen' and e.get('code') == 'HEALTH_CHECK_FAILED' for e in (r['report']['errors'] or []))]
    item('source_freeze_response_and_report', bool(source_report), source_wire, ['source receiver accepted identity_frozen response and matching accepted non-forwarding identity-stage report'])
    source_bracket = {}
    for phase in ('after-clone-creation', 'after-clone-observation'):
        candidates = [r for r in snapshots if r.get('role')=='source' and r.get('container')==SOURCE and r.get('phase')==phase and r.get('status')=='available']
        if len(candidates)==1:
            candidate=candidates[0];snap=candidate['snapshot'];proc=snap.get('process_before', {})
            receipt=direct.get(SOURCE, {}).get('parent_wait_receipt', {})
            if (snap.get('process_coherent') is True and direct.get(SOURCE, {}).get('final_stop_matched') is True and
                proc.get('pid')==receipt.get('pid') and proc.get('start_ticks')==receipt.get('start_ticks') and
                snap.get('agent_process_id', {}).get('text', '').strip()==receipt.get('id')):
                source_bracket[phase]=candidate
    latch = []
    contradiction = False
    for i, row in enumerate(rows):
        if row.get('label') != 'freeze latch held' or i == 0:
            continue
        try:
            first, held = rows[i-1], row
            ok1, beat1 = frozen_heartbeat(first); ok2, beat2 = frozen_heartbeat(held)
            contradiction = not ok2
            bracketed = (len(source_bracket)==2 and
                         wall_ns(source_bracket['after-clone-creation']['snapshot']['read_end']) <= wall_ns(first['read_start']) and
                         wall_ns(held['read_end']) <= wall_ns(source_bracket['after-clone-observation']['snapshot']['read_start']))
            current = all(wall_ns(freeze['completed_at']) <= int(b['wall_unix_ns']) <= wall_ns(r['read_end']) for r,b in ((first,beat1),(held,beat2)))
            if (ok1 and ok2 and current and bracketed and beat1['boot_id']==beat2['boot_id'] and int(beat2['mono_ns'])>int(beat1['mono_ns']) and
                wall_ns(held['read_start'])-wall_ns(first['read_end']) >= 3_000_000_000 and
                first.get('clone_observation', {}).get('mapping_frozen') is True and not first.get('clone_read_error')):
                latch.append(dict(first_row=i-1, held_row=i, first_heartbeat=beat1, held_heartbeat=beat2,
                                  first_read_start=first['read_start'], held_read_end=held['read_end'], consumer=SOURCE, source_process_bracket={p:{'read_start':v['snapshot']['read_start'],'read_end':v['snapshot']['read_end'],'process':v['snapshot']['process_before'],'launch_id':v['snapshot']['agent_process_id']['text'].strip()} for p,v in source_bracket.items()},
                                  central_report_at_first=first.get('observation'), central_report_at_held=held.get('observation'),
                                  scope='source local heartbeat/ruleset only; central reports not attributed to clone; existing three-second held sample, no restart or indefinite persistence proof'))
        except (KeyError, TypeError, ValueError) as error:
            errors.append('consumer heartbeat: '+redact_error(error))
    item('consumer_latch_held', bool(latch), latch, ['fresh source identity_frozen heartbeat, serving 0, zero policy expiry, verified baseline, then unchanged three-second held sample'])
    retention = []
    for phase in ('before-terminal-stop', 'after-terminal-stop'):
        matches = [s for s in clone if s['phase'] == phase]
        if len(matches) == 1:
            logs = matches[0]['snapshot'].get('logs', {})
            try:
                closure=direct[CLONE]
                complete=all(logs.get(n, {}).get('status') == 'available' for n in ('agent', 'watchdog'))
                if phase=='before-terminal-stop':
                    complete=complete and all(wall_ns(logs[n]['read_end']) <= wall_ns(closure['closure_start']) for n in ('agent','watchdog'))
                else:
                    complete=complete and all(wall_ns(logs[n]['read_start']) >= wall_ns(closure['closure_end']) for n in ('agent','watchdog'))
                retention.append(complete)
            except (KeyError, TypeError, ValueError):
                retention.append(False)
        else:
            retention.append(False)
    item('clone_stop_log_retention', all(retention), log_diagnostics, ['available independent pre/post agent and watchdog reads; absent/truncated logs are not successful empty logs'])
    complete = not errors and all(r['status'] == 'established' for r in results.values())
    status = 'pass' if complete else 'not established'
    if central and contradiction:
        status = 'fail'
    return dict(run_id=manifest['run_id'], build=manifest['build'], source_dirty=manifest['source_dirty'],
                status=status, criteria=results, errors=errors, snat=snat, source_runtime=[{k:v for k,v in r.items() if k != 'snapshot'} | {'process':r.get('snapshot',{}).get('process_before'), 'process_coherent':r.get('snapshot',{}).get('process_coherent'), 'configs':r.get('snapshot',{}).get('configs')} for r in snapshots if r.get('role')=='source'], input_manifest='auxiliary-inputs.json',
                scope='Auxiliary only. Socket/read/receipt bounds, not kernel exit times. Accepted clone reports plus owned sockets establish bounded live transport; no claim of packet-by-packet HTTP tracing, indefinite latch persistence, clone datapath correctness, watchdog stop, or AT re-score.')


def review(root, run_id):
    try:
        manifest, data = inputs(root, run_id)
        stops = read_json(root/'stop-receipt-review.json')
        result = evaluate(manifest, data, stops)
    except (OSError, KeyError, TypeError, ValueError) as error:
        result = dict(run_id=run_id, status='not established', criteria={}, errors=[redact_error(error)])
    (root/'clone-transport-review.json').write_text(redact_text(json.dumps(result, indent=2)+'\n'))
    return result


if __name__ == '__main__':
    import sys
    result = review(Path(sys.argv[1]), sys.argv[2])
    print(json.dumps(dict(run_id=result['run_id'], status=result['status'], errors=result['errors'], criteria={k:v['status'] for k,v in result['criteria'].items()}), indent=2))
    raise SystemExit(0 if result['status']=='pass' else 1)
