"""D-164 diagnostic export and recovery-context attribution, never permission."""
import calendar
import re


def utc_instant(value):
    """Comparable UTC components with exact nanoseconds, without float time."""
    if not isinstance(value, str):
        raise ValueError('UTC timestamp must be a string')
    match = re.fullmatch(r'([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]{1,9}))?Z', value)
    if not match:
        raise ValueError('invalid UTC timestamp')
    year, month, day, hour, minute, second = map(int, match.groups()[:6])
    if not (1 <= month <= 12 and 1 <= day <= calendar.monthrange(year, month)[1]
            and hour < 24 and minute < 60 and second < 60):
        raise ValueError('invalid UTC calendar time')
    return year, month, day, hour, minute, second, int((match[7] or '').ljust(9, '0'))


def valid_read(value):
    if not isinstance(value, dict) or set(value) != {'read_at', 'source', 'candidate_generation', 'verified', 'diagnostic'}:
        return False
    generation = value['candidate_generation']
    if not (value['source'] in ('apply', 'live') and type(value['verified']) is bool
            and isinstance(generation, str) and re.fullmatch(r'0|[1-9][0-9]{0,19}', generation)
            and int(generation) <= 18446744073709551615
            and isinstance(value['diagnostic'], str) and len(value['diagnostic']) <= 500):
        return False
    try:
        utc_instant(value['read_at'])
        value['diagnostic'].encode('utf-8')
    except (ValueError, UnicodeError):
        return False
    return True


_NO_UPPER_BOUND = object()


def within_window(value, lower, upper=_NO_UPPER_BOUND):
    """Existing [lower, upper) recovery bounds, at full UTC precision."""
    try:
        at = utc_instant(value)
        return at >= utc_instant(lower) and (upper is _NO_UPPER_BOUND or at < utc_instant(upper))
    except ValueError:
        return False


def qos_detail(report):
    checks = report.get('checks')
    if not isinstance(checks, list):
        return ''
    return next((c['detail'] for c in checks if isinstance(c, dict) and c.get('name') == 'qos'
                 and isinstance(c.get('detail'), str)), '')


def legacy_sources(report):
    # Preserve the historical compatibility rule, labelled separately from
    # structured data. Never parse the opaque serving_read.diagnostic string.
    detail = qos_detail(report)
    return [source for source in ('apply', 'live')
            if 'source='+source in detail and 'verified=true' in detail]


def export_read(row):
    report = row['report']
    presence = 'absent' if 'serving_read' not in report else 'null' if report['serving_read'] is None else 'value'
    result = dict(report_id=report.get('report_id'), node_id=report.get('node_id'),
                  node_incarnation_id=report.get('node_incarnation_id'),
                  observation_at=report.get('observed_at'), received_at=row.get('received_at'),
                  status=row['status'], presence=presence)
    if presence != 'absent':
        result['serving_read'] = report['serving_read']
    if presence == 'value':
        result['interpretation_basis'] = 'structured' if valid_read(report['serving_read']) else 'invalid_structured'
    else:
        result.update(interpretation_basis='legacy_qos',
                      legacy_interpretation=dict(qos_detail=qos_detail(report), verified_sources=legacy_sources(report)))
    return result


def recovery_source_matches(report, source, removal_end):
    """Call only inside the existing clean-TC set; not same-operation identity.

    TC and complete serving read timestamps have different capture boundaries.
    Candidate and the existing removal lower bound constrain recovery context;
    no equality, age, upper-bound, or diagnostic-text rule is introduced.
    """
    value = report.get('serving_read')
    if value is None:
        return source in legacy_sources(report)
    if not valid_read(value):
        return False
    tc = report.get('tc_evidence') or {}
    if value['source'] != source or value['verified'] is not True or value['candidate_generation'] != tc.get('candidate_generation'):
        return False
    return within_window(value['read_at'], removal_end)
