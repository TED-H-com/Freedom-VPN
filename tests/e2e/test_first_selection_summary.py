import unittest
from summarize_first_selection import elapsed, data_around


def packet(lo, hi, kind=4):
    return dict(mono_lower_ns=str(lo),mono_upper_ns=str(hi),wireguard=dict(type=kind))


class Ordering(unittest.TestCase):
    def test_response_bounds_do_not_invent_order(self):
        data=[packet(1,2),packet(8,12),packet(12,13),packet(14,15),packet(1,2,1)]
        self.assertEqual(data_around(data,packet(10,12,2)),dict(total_records=4,before=1,after=1,overlaps=2))

    def test_missing_response_preserves_data_and_unknown_order(self):
        self.assertEqual(data_around([packet(1,2)],None),dict(total_records=1,before=None,after=None,overlaps=None))
        self.assertIsNone(elapsed(packet(1,2),None))

    def test_origin_delay_preserves_decimal_nanoseconds(self):
        self.assertEqual(elapsed(packet(10**18,10**18+100),packet(10**18+1000,10**18+1100)),dict(lower_ns='900',upper_ns='1100'))


if __name__=='__main__': unittest.main()
