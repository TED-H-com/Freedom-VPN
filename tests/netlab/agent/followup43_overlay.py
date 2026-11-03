"""TEST ONLY: timestamp-only Go build overlays; frozen trees stay byte-identical."""
import hashlib
import json
from pathlib import Path
import subprocess


def replace_once(text, before, after):
    if text.count(before) != 1:
        raise ValueError('instrumentation anchor is not unique: ' + before)
    return text.replace(before, after, 1)


def instrument(text):
    text = replace_once(text, 'func (e *Engine) Step(ctx context.Context) error {',
                        'func (e *Engine) f43OriginalStep(ctx context.Context) error {')
    text = replace_once(text, '\twasServing := e.expectServing.Load()',
                        '\tf43Accounting := e.f43Span("pre_apply_accounting")\n'
                        '\tif f43Accounting != nil {\n'
                        '\t\tdefer f43Accounting.write(e)\n\t}\n'
                        '\twasServing := e.expectServing.Load()')
    text = replace_once(text, '\tres, err := e.plane.Apply(applyCtx, model, prev)',
                        '\tif f43Accounting != nil {\n'
                        '\t\tf43Accounting.ReachedApply = true\n'
                        '\t\tf43Accounting.stop()\n\t}\n'
                        '\tres, err := e.plane.Apply(applyCtx, model, prev)')
    return text


def prepare(root):
    out = root / '.agent/f43/timing/overlays'
    fixture = root / 'tests/netlab/agent/followup43_spans.go.txt'
    pins = {}
    for arm in ['A', 'B']:
        tree = root / '.agent/tmp/f43-timing' / arm
        dest = out / arm
        dest.mkdir(parents=True, exist_ok=False)
        original = tree / 'node/agent/engine.go'
        engine = dest / 'engine.go'
        engine.write_text(instrument(original.read_text()))
        helper = dest / 'followup43_spans.go'
        helper.write_bytes(fixture.read_bytes())
        subprocess.run(['gofmt', '-w', str(engine), str(helper)], check=True)
        overlay = {'Replace': {str(original): str(engine),
                   str(tree / 'node/agent/followup43_spans.go'): str(helper)}}
        (dest / 'overlay.json').write_text(json.dumps(overlay, indent=2) + '\n')
        pins[arm] = {p.name: hashlib.sha256(p.read_bytes()).hexdigest()
                     for p in [engine, helper]}
        pins[arm]['original_engine_sha256'] = hashlib.sha256(original.read_bytes()).hexdigest()
    (out / 'pins.json').write_text(json.dumps(pins, indent=2) + '\n')
    print(json.dumps(pins))


if __name__ == '__main__':
    prepare(Path.cwd())
