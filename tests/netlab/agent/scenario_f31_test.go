//go:build netlab

package agentlab

import (
	"os"
	"path/filepath"
	"testing"
)

// TEST ONLY opt-in observations. No lease refresh, wait-for-serving predicate
// or assertion is added. Read the original persisted anchors and signed member
// deadlines, not an inferred validity from later lease_expired log lines.
func followup31RestartCapture(t *testing.T) func(string) {
	t.Helper()
	dir := os.Getenv("FVPN_F31_RESTART_EVIDENCE_DIR")
	if dir == "" {
		return func(string) {}
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(dir, "iteration-")
	if err != nil {
		t.Fatal(err)
	}
	capture := func(stage string) {
		t.Helper()
		raw, errOut, err := dockerExec("python3", "-c", followup31RestartSnapshot)
		if err != nil {
			t.Errorf("restart snapshot unavailable: %v %s", err, errOut)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, stage+".json"), []byte(raw), 0600); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() {
		capture("finished")
		for _, name := range []string{"operations.jsonl", "agent.log", "restart-stop.json"} {
			raw, _, err := dockerExec("cat", labDir+"/"+name)
			if err == nil {
				err = os.WriteFile(filepath.Join(dir, name), []byte(raw), 0600)
			}
			if err != nil {
				t.Error("restart capture", name, err)
			}
		}
		t.Log("TEST ONLY restart evidence:", filepath.Base(dir))
	})
	return capture
}

const followup31RestartSnapshot = `import calendar,datetime,json,pathlib,sqlite3,time
root=pathlib.Path('/scratch/w2'); before=time.monotonic_ns()
db=sqlite3.connect('file:'+str(root/'state/agent.db')+'?mode=ro',uri=True)
row=db.execute('SELECT envelope,mono_sent_ns,boot_id FROM lease WHERE id=1').fetchone()
out={'start_ns':str(before),'boot_id':pathlib.Path('/proc/sys/kernel/random/boot_id').read_text().strip()}
def ns(s):
 whole,_,fraction=s.removesuffix('Z').partition('.')
 return calendar.timegm(datetime.datetime.fromisoformat(whole).timetuple())*1000000000+int(fraction.ljust(9,'0'))
if row:
 lease=json.loads(row[0])['signed']; anchor=row[1]; issued=ns(lease['issued_at'])
 out['lease']={k:lease[k] for k in ['lease_id','config_generation','issued_at','expires_at','max_offline_seconds']}
 out['lease']['stored_boot_id']=row[2]; out['lease']['request_mono_ns']=str(anchor)
 out['lease']['node_deadline_ns']=str(anchor+ns(lease['expires_at'])-issued)
 out['lease']['members']=[{'member_id':m['member_id'],'lease_end':m['lease_end'],'effective_service_end':m['effective_service_end'],'deadline_ns':str(anchor+min(ns(m['lease_end']),ns(m['effective_service_end']),ns(lease['expires_at']))-issued)} for m in lease['members']]
out['desired']=db.execute('SELECT accepted_generation,applied_generation FROM desired WHERE id=1').fetchone()
db.close()
try: out['heartbeat']=root.joinpath('heartbeat').read_text()
except FileNotFoundError: out['heartbeat']=None
out['end_ns']=str(time.monotonic_ns())
if row:
 out['lease']['node_remaining_ns']=str(int(out['lease']['node_deadline_ns'])-int(out['end_ns']))
 for m in out['lease']['members']: m['remaining_ns']=str(int(m['deadline_ns'])-int(out['end_ns']))
print(json.dumps(out))
`
