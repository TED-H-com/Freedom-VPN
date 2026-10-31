#!/usr/bin/env python3
"""TEST ONLY explicit extra API generation/path restoration during a stalled run."""
from pathlib import Path
import json,os,subprocess,tempfile
root=Path(__file__).resolve().parents[4]
os.chdir(root)
with tempfile.TemporaryDirectory(prefix='assist.',dir=root/'.agent/tmp') as temp:
 p=Path(temp)
 (p/'assist_test.go').write_text('''//go:build e2e
package e2e
import("testing";"context";"time";"encoding/json";"os";"os/exec";"github.com/google/uuid")
func TestIsolationAssist(t *testing.T){
 ctx,cancel:=context.WithTimeout(t.Context(),60*time.Second);defer cancel()
 lab,e:=openLab();if e!=nil{t.Fatal(e)}
 if lab.prefix!="fvpn-w4-isolation"{t.Fatal("binding")}
 raw,e:=lab.docker(lab.node,"cat","/var/lib/vpn-node/identity/node.json");if e!=nil{t.Fatal(e)}
 var id struct{Node string `json:"node_id"`};if json.Unmarshal(raw,&id)!=nil{t.Fatal("identity")}
 admin,e:=lab.login(ctx,"e2e-admin","admin");if e!=nil{t.Fatal(e)}
 path:=lab.public+"/api/v1/admin/nodes/"+id.Node
 get,e:=admin.call(ctx,"GET",path,nil,nil);if e!=nil{t.Fatal(e)}
 start:=time.Now().UTC()
 patch,e:=admin.call(ctx,"PATCH",path,map[string]any{"capacity":map[string]any{"down_bps":208000000,"up_bps":200000000,"member_down_bps_cap":100000000,"member_up_bps_cap":100000000}},map[string]string{"If-Match":get.Header.Get("ETag"),"Idempotency-Key":uuid.NewString()})
 out,restoreErr:=exec.Command("docker","exec",lab.node,"sh","-c","ip link set fvmb0 up; ip -4 address restore < /tmp/isolation-member-addresses; ip -4 route restore < /tmp/isolation-member-routes").CombinedOutput()
 evidence:=map[string]any{"at":start,"completed_at":time.Now().UTC(),"action":"Extra clean capacity generation (208000000) after recovery stalled; restore saved lab member addresses/routes again. No row or observation edits.","http_status":patch.Status,"response":json.RawMessage(patch.Body),"restore_output":string(out),"restore_ok":restoreErr==nil,"api_ok":e==nil,"assisted":true}
 body,_:=json.MarshalIndent(evidence,"","  ");if e=os.WriteFile(os.Getenv("ASSIST_EVIDENCE"),append(body,'\\n'),0600);e!=nil{t.Fatal(e)}
 t.Logf("extra generation status=%d restore_ok=%t",patch.Status,restoreErr==nil)
}
''')
 overlay=p/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(root/'tests/e2e/isolation_assist_test.go'):str(p/'assist_test.go')}}))
 env=dict(os.environ,GOFLAGS='-overlay='+str(overlay),TMPDIR=temp,GOTMPDIR=temp,GOTOOLCHAIN='local',CGO_ENABLED='0',FVPN_E2E='1',FVPN_E2E_PREFIX='w4-isolation',FVPN_E2E_PORT_BASE='36000',FVPN_E2E_STATE=str(root/'.agent/e2e-state-w4-isolation'),ASSIST_EVIDENCE=str(root/'docs/e2e/at/isolation/recovery-action.json'))
 raise SystemExit(subprocess.run(['go','test','-tags','e2e','-count=1','-timeout=90s','-run=^TestIsolationAssist$','-v','./tests/e2e'],env=env).returncode)
