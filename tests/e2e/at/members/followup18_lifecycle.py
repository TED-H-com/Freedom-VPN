"""Join the retained export rejection to original producer bytes, read-only."""
import hashlib
import json
import subprocess
from pathlib import Path
from evidence_io import load, publish

ROOT=Path('docs/e2e/at/members/history/follow-up-18-run1')
MEASURED='b4317bd4e4ad178609b71d3fd4fbb986cb9947db'
PHASES={'prepare','quarantine','shape','install','release','verify','revoke','rollback'}


def main():
    case=load(ROOT/'AT02.json')
    source=Path('.agent/tmp/f18-run1/f18-parser-producer.jsonl')
    raw=source.read_bytes()
    lines=raw.splitlines(keepends=True)
    unsupported=[];offset=0
    for number,line in enumerate(lines,1):
        record=json.loads(line)
        timing=record.get('timing') or {}
        for index,command in enumerate(timing.get('commands') or []):
            if command['phase'] not in PHASES:
                unsupported.append(dict(line=number,record_byte_start=str(offset),record_sha256=hashlib.sha256(line).hexdigest(),
                    run_id=record['run_id'],boot_id=record['boot_id'],sequence=record['sequence'],command_index=index,command=command,
                    original_record=record,validator_rejection=f'command {index} invalid lifecycle'))
        offset+=len(line)
    receipts=[]
    for name in ['AT02','AT03','AT04','AT08','AT20','AT21','AT22','AT09']:
        current=load(ROOT/(name+'.json'))
        for step in current['steps']:
            if step['name']!='populated-grant serving readiness':continue
            value=json.loads(step['actual']);receipt=value.get('producer_stream') or {}
            if receipt.get('node')!='node' or not receipt.get('bytes'):continue
            size=int(receipt['bytes']);matched=len(raw)>=size and hashlib.sha256(raw[:size]).hexdigest()==receipt['sha256']
            rejected=next((r for r in unsupported if int(r['record_byte_start'])<size),None)
            if step['status']=='fail' or rejected:
                receipts.append(dict(case=name,step_at=step['at'],step_status=step['status'],ready=value['ready'],last_stage=value.get('last_stage'),reader_error=value.get('reader_error'),
                    original_export_receipt=receipt,exact_original_export_prefix_matches=matched,
                    first_unsupported_command={k:rejected[k] for k in ['line','run_id','boot_id','sequence','command_index','validator_rejection']} if matched and rejected else None))
    paths=['tests/e2e/at/members/command_lifecycle_test.go','tests/e2e/at/members/timing_stream_test.go','node/enforce/apply.go']
    result=dict(run_id=case['run_id'],source_commit=case['source_commit'],producer_bytes=str(len(raw)),producer_sha256=hashlib.sha256(raw).hexdigest(),
        allowed_phases_in_measured_harness=sorted(PHASES),unsupported_phase_commands=unsupported,original_export_receipt_joins=receipts,
        original_failed_attribution_checks=[s for s in case['steps'] if s['name'] in ('counter reader enforcement attribution','D-135 D-136 node-wide baseline transitions (provisional)')],
        source_pins={p:hashlib.sha256(subprocess.check_output(['git','show',MEASURED+':'+p])).hexdigest() for p in paths},
        scope='The measured parser rejects an unknown phase before checking its lifecycle stamps and discards the complete trace on any invalid row. Exact prefix pins join earlier read receipts to the final private producer stream. This demonstrates the phase compatibility defect only where such a join exists; it does not explain every central-readiness failure or packet loss. No validator, original result, availability classification or retry changed.')
    Path('.agent/tmp/f18-run1/lifecycle-replay.json').write_text(json.dumps(result,indent=2)+'\n')
    publish(Path('.agent/tmp/f18-run1/lifecycle-replay.json'))
    print('unsupported phase commands',len(unsupported),'exact receipt joins',sum(r['exact_original_export_prefix_matches'] for r in receipts))


if __name__=='__main__':main()
