#!/usr/bin/env python3
"""TEST ONLY gate-2 wrapper: recheck suite count after topology startup."""
import os
from pathlib import Path
import sys
sys.dont_write_bytecode=True
if sys.argv[1:2]==['test'] and os.environ.get('FVPN_NETLAB')=='1':
    sys.path.insert(0,str(Path.cwd()/'docs/poc/G2-followup27-evidence'))
    import host
    host.admission('admission/gate2-after-startup')
os.execv(os.environ['FVPN_F27_REAL_GO'],[os.environ['FVPN_F27_REAL_GO'],*sys.argv[1:]])
