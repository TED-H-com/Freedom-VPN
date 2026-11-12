#!/usr/bin/env python3
"""TEST ONLY explicit extra API generation/path restoration during a stalled run."""
from pathlib import Path
import json,os,subprocess,tempfile
root=Path(__file__).resolve().parents[4]
os.chdir(root)
with tempfile.TemporaryDirectory(prefix='assist.',dir=root/'.agent/tmp') as temp:
 p=Path(temp)
 overlay=p/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(root/'tests/e2e/isolation_assist_test.go'):str(root/'tests/e2e/at/isolation/assist.go.txt')}}))
 env=dict(os.environ,GOFLAGS='-overlay='+str(overlay),TMPDIR=temp,GOTMPDIR=temp,GOTOOLCHAIN='auto',CGO_ENABLED='0',FVPN_E2E='1',FVPN_E2E_PREFIX='w4-isolation',FVPN_E2E_PORT_BASE='36000',FVPN_E2E_STATE=str(root/'.agent/e2e-state-w4-isolation'),ASSIST_EVIDENCE=str(root/'docs/e2e/at/isolation/recovery-action.json'))
 raise SystemExit(subprocess.run(['go','test','-tags','e2e','-count=1','-timeout=90s','-run=^TestIsolationAssist$','-v','./tests/e2e'],env=env).returncode)
