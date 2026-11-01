#!/usr/bin/env python3
"""TEST ONLY F13-01 listing experiment; run inside the owned privileged lab.

Uses only inet fvpn in an isolated lab netns, with synthetic chain/rule/set
names and integer markers. No input or values from the private sample.
Every listing and mutation attempt is retained, including nonzero exits.
"""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import subprocess
import threading
import time


def run(argv, data=None):
    start = time.monotonic_ns()
    try:
        p = subprocess.run(argv, input=data, stdout=subprocess.PIPE,
                           stderr=subprocess.STDOUT, timeout=4, check=False)
        return p.returncode, p.stdout, start, time.monotonic_ns()
    except subprocess.TimeoutExpired as exc:
        return -1, exc.stdout or b"", start, time.monotonic_ns()


def rules(generation, count):
    return ''.join(f'add rule inet fvpn fv{n % 5} counter comment "TEST ONLY generation {generation}"\n'
                   for n in range(count))


def table(generation, chain_count, rule_count):
    text = 'add table inet fvpn\n'
    hooks = ['input', 'output', 'forward', 'prerouting', 'postrouting']
    for n in range(chain_count):
        text += f'add chain inet fvpn fv{n}'
        if n < 5:
            text += f' {{ type filter hook {hooks[n]} priority 0; policy accept; }}'
        text += '\n'
    text += 'add set inet fvpn fvmarker { type mark; }\n'
    text += f'add element inet fvpn fvmarker {{ {generation} }}\n'
    for n in range(42):
        text += f'add counter inet fvpn fvcount{n}\n'
    return text + rules(generation, rule_count)


def inspect(raw):
    try:
        objects = json.loads(raw)['nftables']
    except (ValueError, KeyError, TypeError):
        return {'valid_json_listing': False}
    seen, duplicates, generations = {}, [], set()
    for index, obj in enumerate(objects):
        if 'chain' in obj:
            c = obj['chain']
            key = tuple(c.get(k) for k in ('family', 'table', 'name'))
            if key in seen:
                first_index, prior = seen[key]
                duplicates.append({'first_index': first_index, 'second_index': index,
                                   'exactly_equal': prior == c,
                                   'handle_equal': prior.get('handle') == c.get('handle')})
            else:
                seen[key] = (index, c)
        if 'rule' in obj:
            comment = obj['rule'].get('comment', '')
            if comment.startswith('TEST ONLY generation '):
                generations.add(int(comment.split()[-1]))
        if 'set' in obj and obj['set'].get('name') == 'fvmarker':
            generations.update(obj['set'].get('elem', []))
    return {'valid_json_listing': True, 'object_count': len(objects),
            'object_types': dict(Counter(k for obj in objects for k in obj)),
            'unique_chains': len(seen), 'duplicates': duplicates,
            'generation_markers': sorted(generations), 'mixed_generations': len(generations) > 1}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True)
    parser.add_argument('--attempts', type=int, default=200, help='per command per condition')
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('requires the owned lab container')
    out = Path(args.output)
    out.mkdir(parents=True, exist_ok=False)
    metadata = {'test_only': True, 'commands': [['nft', '-j', 'list', 'table', 'inet', 'fvpn'],
                                             ['nft', '-j', 'list', 'ruleset']],
                'attempts_per_command_per_condition': args.attempts}
    for key, command in [('nft_version', ['nft', '--version']),
                         ('packages', ['dpkg-query', '-W', '-f=${Package} ${Version}\n', 'nftables', 'libnftables1', 'libnftnl11', 'libmnl0']),
                         ('kernel', ['uname', '-r']), ('arch', ['uname', '-m'])]:
        rc, raw, _, _ = run(command)
        if rc:
            raise RuntimeError(key)
        metadata[key] = raw.decode().strip()
    (out / 'environment.json').write_text(json.dumps(metadata, indent=2) + '\n')
    summary = []
    with (out / 'attempts.jsonl').open('w') as listing_log, (out / 'mutations.jsonl').open('w') as mutation_log:
        def mutation(condition, ordinal, script):
            rc, raw, start, end = run(['nft', '-f', '-'], script.encode())
            row = dict(condition=condition, ordinal=ordinal, start_ns=str(start), end_ns=str(end),
                       exit=rc, script_sha256=hashlib.sha256(script.encode()).hexdigest(),
                       output_bytes=len(raw), output_sha256=hashlib.sha256(raw).hexdigest())
            mutation_log.write(json.dumps(row) + '\n')
            mutation_log.flush()
            if rc:
                (out / f'{condition}-mutation-{ordinal}.txt').write_bytes(raw)
            return rc

        for mode, chains, rule_count in [('static', 5, 48), ('rules', 5, 48),
                                         ('replace', 5, 48), ('rules', 256, 512),
                                         ('replace', 256, 512)]:
            condition = f'{mode}-{chains}-chains-{rule_count}-rules'
            script = 'destroy table inet fvpn\n' + table(1, chains, rule_count)
            (out / f'{condition}-initial.nft').write_text(script)
            if mutation(condition, 0, script):
                raise RuntimeError('initialization failed')
            stop = threading.Event()
            writer_stats = Counter()

            def writer():
                generation = 1
                while not stop.is_set():
                    generation += 1
                    if mode == 'replace':
                        change = 'delete table inet fvpn\n' + table(generation, chains, rule_count)
                    else:
                        change = 'flush table inet fvpn\nflush set inet fvpn fvmarker\n'
                        change += f'add element inet fvpn fvmarker {{ {generation} }}\n' + rules(generation, rule_count)
                    writer_stats['attempts'] += 1
                    if mutation(condition, generation, change):
                        writer_stats['failures'] += 1
                    stop.wait(0.002)

            thread = None
            if mode != 'static':
                thread = threading.Thread(target=writer)
                thread.start()
            counts = [Counter(), Counter()]
            try:
                for attempt in range(args.attempts):
                    for index, command in enumerate(metadata['commands']):
                        rc, raw, start, end = run(command)
                        facts = inspect(raw)
                        row = dict(condition=condition, attempt=attempt + 1, command=command,
                                   start_ns=str(start), end_ns=str(end), exit=rc, bytes=len(raw),
                                   sha256=hashlib.sha256(raw).hexdigest(), **facts)
                        counts[index]['attempts'] += 1
                        counts[index]['nonzero_exit'] += int(rc != 0)
                        counts[index]['invalid_json'] += int(not facts['valid_json_listing'])
                        counts[index]['duplicate_listings'] += int(bool(facts.get('duplicates')))
                        counts[index]['mixed_generation_listings'] += int(facts.get('mixed_generations', False))
                        if attempt == 0 or rc or facts.get('duplicates') or facts.get('mixed_generations'):
                            filename = f'{condition}-{index}-{attempt + 1}.txt'
                            (out / filename).write_bytes(raw)
                            row['retained_output'] = filename
                        listing_log.write(json.dumps(row) + '\n')
                        listing_log.flush()
            finally:
                stop.set()
                if thread:
                    thread.join()
            for command, count in zip(metadata['commands'], counts):
                summary.append(dict(condition=condition, command=command, **count, writer=dict(writer_stats)))
            (out / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
            print(json.dumps(summary[-2:]), flush=True)
        if mutation('cleanup', 0, 'delete table inet fvpn\n'):
            raise RuntimeError('cleanup failed')


if __name__ == '__main__':
    main()
