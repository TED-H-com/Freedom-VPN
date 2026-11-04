SELECT jsonb_build_object('read_at',clock_timestamp(),'node_id','01a1a6fb-5e05-741c-ac02-cd000d4036be',
 'window_start','2026-11-04T14:32:49.364796+00:00','window_end','2026-11-04T14:56:48.636097+00:00',
 'rows',coalesce((SELECT jsonb_agg(jsonb_build_object(
   'report_id',report_id,'incarnation_id',incarnation_id,
   'received_at',received_at,'observed_at',observed_at,
   'generation',applied_generation::text,'member_forwarding_enabled',member_forwarding_enabled)
   ORDER BY received_at,id)
 FROM node_observations WHERE node_id='01a1a6fb-5e05-741c-ac02-cd000d4036be'::uuid
 AND received_at>='2026-11-04T14:32:49.364796+00:00'::timestamptz AND received_at<'2026-11-04T14:56:48.636097+00:00'::timestamptz),'[]'::jsonb));
