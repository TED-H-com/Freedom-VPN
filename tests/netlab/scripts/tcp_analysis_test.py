"""TEST ONLY correlation failure and sequence-space regressions."""
import unittest
from tcp_analysis import classify, correlate


def segment(n, seq, flags, length=0, ack=0):
    return dict(packet=n,wall_ns=str(100000+n*10000),interface='fv-wg0',direction='out',
                source_port=40000,destination_port=9001,seq=seq,ack=ack,flags=flags,
                flags_text=[],data_length=length,ip_id=n)


class TCPAnalysisTests(unittest.TestCase):
    def test_overlapping_retransmission_and_wrap(self):
        origin=[segment(1,4294967294,2),segment(2,4294967295,24,4),segment(3,1,24,2),segment(4,3,16,ack=12),segment(5,3,16,ack=12)]
        segments,unmatched=correlate({'origin':origin},[])
        self.assertEqual(unmatched,[])
        self.assertEqual([s['retransmission'] for s in segments],[False,False,True,False,False])
        self.assertEqual(segments[2]['retransmits_segments'],[2])
        self.assertEqual(segments[2]['relative_seq'],3)
        self.assertTrue(segments[-1]['repeat_ack'])

    def test_missing_handshake_ambiguous_identity_and_duplicate_stage(self):
        with self.assertRaises(ValueError): correlate({'origin':[segment(1,3,24,4)]},[])
        one=segment(1,10,2); two=dict(one,packet=2)
        with self.assertRaises(ValueError): correlate({'origin':[one,two]},[])
        node=dict(one,interface='fv-fh',direction='in')
        with self.assertRaises(ValueError): correlate({'origin':[one],'node':[node,dict(node,packet=2)]},[])

    def test_copies_at_hops_are_one_segment_and_unmatched_is_explicit(self):
        one=segment(1,10,2); node=dict(one,interface='fv-fh',direction='in')
        other=dict(node,packet=2,seq=900)
        segments,unmatched=correlate({'origin':[one],'node':[node,other]},[])
        self.assertEqual(len(segments),1)
        self.assertEqual(len(segments[0]['stages']),2)
        self.assertEqual(len(unmatched),1)

    def test_every_bracket_match_retained(self):
        w=dict(generation='1',install_start=10000,install_end=20000,release_start=21000,release_end=30000,verify_start=31000,verify_end=50000)
        self.assertEqual(classify(0,[w]),[])
        self.assertEqual([h['phase'] for h in classify(15000,[w])],['install'])
        self.assertEqual([h['phase'] for h in classify(20500,[w])],['install','guard_certain','release'])
        self.assertEqual([h['phase'] for h in classify(40000,[w])],['verify'])
        self.assertEqual(len(classify(15000,[w,w])),2)


if __name__=='__main__':unittest.main()
