#!/usr/bin/env python3
"""Publish bounded categories/counts from protected native-run gateway logs."""
import re
from run import PRIVATE, PUBLIC, write_json


def main():
    raw=(PRIVATE/'captured-logs/gateway_stdout_stderr.log').read_text(errors='replace')
    lines=raw.splitlines()
    budget=[x for x in lines if '256 worker_connections are not enough' in x]
    auth=[x for x in lines if re.search(r'[?&]user_code=[^\s&"\\]+',x)]
    def stamp(line):
        m=re.match(r'(\d{4}/\d\d/\d\d) (\d\d:\d\d:\d\d)',line)
        return m[1].replace('/','-')+'T'+m[2]+'Z' if m else None
    record={'source':'protected gateway stdout/stderr from the isolated admission recheck',
            'evidence':'native-scale.json','owner':'w3-devstack-e2e',
            'configuration':'deploy/devstack/nginx.conf:1–3: worker_processes 1; worker_connections 256',
            'connection_budget':{'matching_lines':len(budget),'first_utc':stamp(budget[0]) if budget else None,'last_utc':stamp(budget[-1]) if budget else None,
                                 'fixed_message':'256 worker_connections are not enough','configuration_changed':False},
            'auth_query_logging':{'status':'failed' if auth else 'not observed','matching_lines':len(auth),'query_key':'user_code','query_values_published':False,
                                  'scope':'Native device-flow query appears in gateway error diagnostics during upstream connection exhaustion; known-secret canary checks alone do not cover this value.'},
            'infrastructure_addresses_and_account_ids_published':False}
    write_json(PUBLIC/'native-diagnostics.json',record)
    print('gateway budget messages:',len(budget),'auth-query lines:',len(auth))


if __name__=='__main__':
    main()
