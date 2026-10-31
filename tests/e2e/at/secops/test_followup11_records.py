"""Evidence gaps must not become transport passes or SECOPS-21 closure."""
import copy
import unittest

from followup11_records import protocol_cutoff, generation_status


class Followup11VerdictTests(unittest.TestCase):
    def fixture(self):
        def sample(replies, attempts):
            return dict(status_read_ok=True,counter_read_ok=True,
                        status=dict(running=True,tcp_echoes=replies,udp_replies=replies,
                                    tcp_attempts=attempts,udp_attempts=attempts),
                        counters=dict(Present=True,upload_bytes=str(replies*100),download_bytes=str(replies*100)))
        return dict(positive_samples=[sample(1,1),sample(5,5)],cutoff_samples=[sample(9,10),sample(9,20)])

    def test_active_attempts_and_positive_counters_required(self):
        row=self.fixture()
        for protocol in ('TCP','UDP'):
            self.assertEqual(protocol_cutoff(row,protocol),'passed')
            stopped=copy.deepcopy(row);stopped['cutoff_samples'][1]['status']['running']=False
            self.assertEqual(protocol_cutoff(stopped,protocol),'blocked')
            absent=copy.deepcopy(row);absent['positive_samples'][0]['counters']['Present']=False
            self.assertEqual(protocol_cutoff(absent,protocol),'blocked')
        late=copy.deepcopy(row);late['cutoff_samples'][1]['status']['udp_replies']=10
        self.assertEqual(protocol_cutoff(late,'UDP'),'failed')
        self.assertEqual(protocol_cutoff(late,'TCP'),'passed')
        quiet_sender=copy.deepcopy(row);quiet_sender['cutoff_samples'][1]['status']['tcp_attempts']=10
        self.assertEqual(protocol_cutoff(quiet_sender,'TCP'),'blocked')

    def test_no_d099_row_means_pending_even_with_compiler_ack(self):
        live={'D-095':{'passed':True},'SECOPS-21':{
            'offers':[{'generation':'15'}],'observations':[{'applied':'15'}],
            'retained_at_t1':{'applied':'14'},'manual_generation_adjustment':False,'stale_generation_log_count':0,
            'first_http_offer':{'first_response_proven':True,'generation':'15'},
            'first_offer_local_store':{'accepted':True,'generation':'15'}}}
        self.assertEqual(generation_status({'d099_roadmap_present':False},live),'open — D-099 pending')
        self.assertEqual(generation_status({'d099_roadmap_present':True},live),'closed')
        del live['SECOPS-21']['first_offer_local_store']
        self.assertEqual(generation_status({'d099_roadmap_present':True},live),'open')
        live['SECOPS-21']['first_offer_local_store']={'accepted':True,'generation':'16'}
        self.assertEqual(generation_status({'d099_roadmap_present':True},live),'open')
