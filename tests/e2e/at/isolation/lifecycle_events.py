"""Owned-container lifecycle observer; PID1 events never imply agent exit."""
import json
from pathlib import Path
import subprocess


def start(directory):
    path=Path(directory)/'isolation-docker-events.jsonl'
    stream=path.open('w')
    args=['docker','events','--filter','type=container']
    for container in ['fvpn-w4-isolation-node','fvpn-w4-isolation-node-clone']:
        args+=['--filter','container='+container]
    for event in ['kill','die','start','restart','stop','pause','unpause']:
        args+=['--filter','event='+event]
    args+=['--format','{{json .}}']
    return subprocess.Popen(args,stdout=stream),stream,path


def finish(observer,root):
    process,stream,path=observer
    process.terminate();process.wait(timeout=5);stream.close()
    rows=[]
    for line in path.read_text().splitlines():
        value=json.loads(line);attrs=value['Actor']['Attributes']
        rows.append(dict(action=value['Action'],time_ns=str(value['timeNano']),
            attributes={k:attrs[k] for k in ['name','signal','exitCode'] if k in attrs}))
    run_id=json.loads((root/'AT10.json').read_text())['run_id']
    result=dict(run_id=run_id,scope='Owned Docker lifecycle observer started before the assembled driver, stopped after stack teardown. Container PID1 events are separate from pidfd agent-stop samples.',events=rows)
    (root/'container-lifecycle-events.json').write_text(json.dumps(result,indent=2)+'\n')
