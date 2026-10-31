import unittest
from analyze_native_capture import first_gap, match_datagrams

class Correlation(unittest.TestCase):
    def test_segmentation_is_not_loss(self):
        def row(a,n): return {'tcp':{'seq':a,'length':n}}
        self.assertIsNone(first_gap([row(100,100)],[row(100,30),row(130,70)]))
        self.assertEqual(first_gap([row(100,100)],[row(100,30),row(140,60)]),[130,140])
    def test_hash_multiset_keeps_repeated_missing_packets(self):
        rows=[{'packet':i,'ip_length':1360,'ip_sha256':'same'} for i in range(3)]
        got=match_datagrams(rows,rows[:1])
        self.assertEqual(got['sizes']['1360'],{'sent':3,'arrived':1,'absent':2})
        self.assertEqual(got['absent_sender_packets'],[1,2])

if __name__=='__main__': unittest.main()
