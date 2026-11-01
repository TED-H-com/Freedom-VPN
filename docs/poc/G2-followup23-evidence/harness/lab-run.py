import json,os,subprocess,time
from pathlib import Path
prefix='fvpn-w1-netlab-f23'
evidence=Path('docs/poc/G2-followup23-evidence')
receipts=[]
def suites(label):
 p=subprocess.run(['pgrep','-af','gates.sh|mk/at-'],capture_output=True,text=True)
 lines=p.stdout.splitlines()
 receipts.append(dict(label=label,wall_utc=time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()),matches=len(lines)))
 (Path('.agent')/('f23-'+label+'-processes.txt')).write_text(p.stdout)
 (evidence/'lab-admission.json').write_text(json.dumps(receipts,indent=2)+'\n')
 return len(lines)
while suites('before-start')>=2:
 print('waiting: at least two competing suite processes',flush=True)
 time.sleep(60)
make=['make','-f','mk/netlab.mk',f'FVPN_LAB_PREFIX={prefix}']
env=dict(os.environ,FVPN_LAB_PREFIX=prefix)
try:
 with Path('.agent/f23-lab-up.txt').open('w') as log:
  p=subprocess.run(make+['netlab-up'],stdout=log,stderr=subprocess.STDOUT,env=env)
 if p.returncode: raise RuntimeError('lab startup failed')
 print('lab started',flush=True)
 while suites('after-start')>=2:
  print('waiting after startup: at least two competing suite processes',flush=True)
  time.sleep(60)
 # Empty isolated lab namespace; no host network or other lab state touched.
 subprocess.run(['docker','exec',prefix,'ip','netns','add','fv23'],check=True)
 with Path('.agent/f23-lab-experiment.txt').open('w') as log:
  p=subprocess.run(['docker','exec',prefix,'ip','netns','exec','fv23','python3','/work/tests/netlab/scripts/f23_listing.py','--output','/scratch/f23-listing','--attempts','50'],stdout=log,stderr=subprocess.STDOUT)
 subprocess.run(['docker','cp',prefix+':/scratch/f23-listing',str(evidence/'lab-final')],check=True)
 if p.returncode: raise RuntimeError('experiment failed; all attempts retained')
 print('experiment completed',flush=True)
finally:
 with Path('.agent/f23-lab-down.txt').open('w') as log:
  subprocess.run(make+['netlab-down'],stdout=log,stderr=subprocess.STDOUT,env=env)
  subprocess.run(['docker','image','rm',prefix+':lab'],stdout=log,stderr=subprocess.STDOUT)
 print('owned lab teardown complete',flush=True)
