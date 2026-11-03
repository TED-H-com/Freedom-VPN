"""Read-only SYN correlation across the newly approved MSS rewrite."""
import copy


def syn_match(sent, received):
    a, b = copy.deepcopy(sent), copy.deepcopy(received)
    if a.get('flags') != 2 or b.get('flags') != 2:
        return False
    x = [o for o in a.get('options', []) if 'mss' in o]
    y = [o for o in b.get('options', []) if 'mss' in o]
    if len(x) == len(y) == 1 and x[0]['mss'] > 1160 and y[0]['mss'] == 1160:
        x[0]['mss'] = 1160
    return a == b
