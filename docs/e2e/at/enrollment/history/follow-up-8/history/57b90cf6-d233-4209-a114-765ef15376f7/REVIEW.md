# Follow-up 8 diagnostic deadline review

The original AT06/AT07 records remain unchanged. The installed node served with
the separate hookless marker and four rendered chains; native traffic increased
upload/download counters by 446/306 bytes, with a TCP echo and UDP reply.
The enrollment harness then waited only 45 seconds for upload. The production
cadence is 60 seconds, and usage AT20's `waitForUploadedTraffic` allows 90 seconds
for 10-second sampling plus upload and scheduling margin.

This is a harness deadline error, not evidence of a product upload failure.
The real timeout remains recorded. A corrected run must pass all five composed
AT07 rows, `BeforePlusHeldByteEquality`, `OriginalPayloadAndHistoricalAttribution`,
`OriginACKAndIndependentCurrentACK`, retired certificate denial and both probe
processes before these exact timeout/process failures stop blocking. Other
failures cannot use this review. The corrected probe retains its CLI poll result.

Review also found that revoke disables publication. Fresh post-reinstall traffic
requires explicit API publication after accounting checks; no database changes
or continued serving after revocation are expected.
