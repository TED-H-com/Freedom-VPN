#!/usr/bin/env python3
"""D-160 admission journal and timed-section protocol, owned by the harness."""
import datetime
import json
import os
from pathlib import Path
import subprocess
import sys
import time


def stamp():
    return dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z'),
                mono_ns=str(time.monotonic_ns()))


def append(event):
    path = os.environ.get('FVPN_SECOPS_TIMING_JOURNAL')
    if not path:
        raise RuntimeError('timing journal is required')
    with open(path, 'a', encoding='utf-8') as out:
        out.write(json.dumps(event, separators=(',', ':'))+'\n')


def active():
    path = os.environ.get('FVPN_SECOPS_TIMING_ACTIVE')
    return Path(path).read_text() if path and Path(path).exists() else ''


def check(command):
    if active():
        raise RuntimeError('admission check forbidden inside timed section')
    started = stamp()
    result = subprocess.run(['bash', os.environ['FVPN_HOLD_SH']], capture_output=True, text=True)
    free = result.returncode == 0 and result.stdout.strip() == 'free'
    # No private reasons, paths or foreign argv in public evidence.
    event = dict(kind='admission', command=command, answer='free' if free else 'held',
                 started=started, **stamp())
    append(event)
    return free


def admit(command, sleep=time.sleep):
    for attempt in range(121):
        if check(command):
            return
        if attempt == 120:
            raise RuntimeError('120-minute admission limit reached')
        sleep(60)


def begin(name):
    if active():
        raise RuntimeError('nested timed section')
    admit('timed-section:'+name)
    event = dict(kind='section_start', section=name, **stamp())
    # No child command can run between marker installation and the receipt.
    Path(os.environ['FVPN_SECOPS_TIMING_ACTIVE']).write_text(name)
    append(event)
    return event


def end(name):
    if active() != name:
        raise RuntimeError('timed section mismatch')
    event = dict(kind='section_end', section=name, **stamp())
    append(event)
    Path(os.environ['FVPN_SECOPS_TIMING_ACTIVE']).unlink()
    return event


def before_command(command):
    section = active()
    if section:
        # This is an execution receipt, NOT an admission check. D-133 monitor
        # observes overlaps without delaying the subject or querying admission.
        append(dict(kind='timed_command', section=section, command=command, **stamp()))
    else:
        admit(command)


def validate(events):
    errors, sections, opened = [], [], None
    previous = None
    for event in events:
        kind = event['kind']
        if kind == 'section_start':
            if not previous or previous['kind'] != 'admission' or previous.get('answer') != 'free' or previous.get('command') != 'timed-section:'+event['section']:
                errors.append('missing free boundary admission: '+event['section'])
            if opened:
                errors.append('nested section: '+event['section'])
            opened = event
        elif kind == 'section_end':
            if not opened or opened['section'] != event['section']:
                errors.append('unmatched section end: '+event['section'])
            else:
                sections.append(dict(section=opened['section'], start=opened, end=event))
            opened = None
        previous = event
    if opened:
        errors.append('unclosed section: '+opened['section'])
    for event in events:
        n = int(event['mono_ns'])
        if event['kind'] == 'admission':
            first = int(event.get('started', event)['mono_ns'])
            for section in sections:
                if first <= int(section['end']['mono_ns']) and n >= int(section['start']['mono_ns']):
                    errors.append(event['answer']+' admission inside '+section['section'])
        if event['kind'] == 'timed_command' and not any(
                s['section'] == event['section'] and int(s['start']['mono_ns']) <= n <= int(s['end']['mono_ns'])
                for s in sections):
            errors.append('command outside declared section: '+event['section'])
    return dict(passed=not errors, errors=errors, sections=sections,
                admission_checks=[e for e in events if e['kind']=='admission'],
                timed_command_count=sum(e['kind']=='timed_command' for e in events))


def main():
    action, name = sys.argv[1:3]
    if action == 'command':
        before_command(name)
    else:
        result = begin(name) if action == 'begin' else end(name)
        print(json.dumps(result))


if __name__ == '__main__':
    main()
