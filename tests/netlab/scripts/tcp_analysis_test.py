"""TEST ONLY correlation failure and sequence-space regressions."""
import unittest
from tcp_analysis import classify, correlate, failed_reads, validate_read_coverage


def segment(n, seq, flags, length=0, ack=0):
    return dict(packet=n,wall_ns=str(100000+n*10000),interface='fv-wg0',direction='out',
                source_port=40000,destination_port=9001,seq=seq,ack=ack,flags=flags,
                flags_text=[],data_length=length,ip_id=n)


class TCPAnalysisTests(unittest.TestCase):
    def test_successful_reads_require_handshake_and_received_byte_coverage(self):
        request={'origin_out_fv-wg0':{},'node_in_fv-fh':{},'node_out_fv-ih':{},'responder_in_fv-ii':{}}
        reply={'responder_out_fv-ii':{},'node_in_fv-ih':{},'node_out_fv-fh':{},'origin_in_fv-wg0':{'wall_ns':'10000'}}
        syn=dict(direction='request',kind='SYN',stages=request,data_length=0)
        synack=dict(direction='reply',kind='SYN-ACK',stages=reply,data_length=0)
        data=dict(direction='reply',kind='data',stages=reply,data_length=4,relative_seq=1,flags=['ACK'])
        read=dict(event='tcp_read',sequence=1,size=4,wall_ns='20000')
        validate_read_coverage([read],[syn,synack,data])
        for packets in ([],[syn,data],[syn,synack],[syn,synack,dict(data,relative_seq=2)]):
            with self.assertRaises(ValueError):validate_read_coverage([read],packets)
        with self.assertRaises(ValueError):
            validate_read_coverage([dict(read,wall_ns='1000')],[syn,synack,data])

    def test_retransmitted_bytes_do_not_double_count_read_coverage(self):
        request={p:{} for p in ('origin_out_fv-wg0','node_in_fv-fh','node_out_fv-ih','responder_in_fv-ii')}
        reply={p:{'wall_ns':'10000'} for p in ('responder_out_fv-ii','node_in_fv-ih','node_out_fv-fh','origin_in_fv-wg0')}
        packets=[dict(direction='request',kind='SYN',stages=request,data_length=0),
                 dict(direction='reply',kind='SYN-ACK',stages=reply,data_length=0)]
        data=dict(direction='reply',kind='data',stages=reply,data_length=4,relative_seq=1,flags=['ACK'])
        reads=[dict(event='tcp_read',sequence=1,size=4,wall_ns='20000'),dict(event='tcp_read',sequence=2,size=1,wall_ns='30000')]
        with self.assertRaises(ValueError):validate_read_coverage(reads,packets+[data,data])
        validate_read_coverage(reads,packets+[data,dict(data,relative_seq=4,data_length=2)])

    def test_failed_read_keeps_packet_and_guard_context_without_size(self):
        send=dict(event='tcp_send',sequence=7,wall_ns='100',mono_ns='10')
        read=dict(event='tcp_read',sequence=7,wall_ns='200',mono_ns='110')
        packets=[dict(number=1,wall_ns='90'),dict(number=2,wall_ns='150')]
        windows=[dict(generation='1001',install_start=120,verify_end=180)]
        for fields in ({},{'size':0},{'outcome':'TimeoutError'}):
            with self.subTest(fields=fields):
                result=failed_reads([send,dict(read,**fields)],packets,windows)
                self.assertEqual(len(result),1)
                self.assertEqual(result[0]['segments_during_read'],[2])
                self.assertEqual(result[0]['guard_windows'],['1001'])
                self.assertEqual(result[0]['elapsed_ns'],100)
        self.assertEqual(failed_reads([send,dict(read,size=4)],packets,windows),[])
        for events in ([read],[send,send,read]):
            with self.assertRaises(ValueError):failed_reads(events,packets,windows)

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
