# export-clone

Pre-boot cloned enrolled image and hash-only machine identity

Status: pass

Run: 0a687f3a-2351-4234-b43d-882109267967

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

Spec: §6.2, AT07, ENR-07

- clone_observation: {"Source":{"machine_id_sha256":"3070269f14212259fdaafe96e69027560c54aa9bafa4fa9780e41dc5e2d613df","agent_spki_sha256":"96b5aa1abcac8e195c372670e44741b10e69dcea4c192eaaae1077987eb09918","netbird_public_key_sha256":"9288769d9627abeffc01a4f0b3fb4d4eab781cb6d159efce0601211c0f22f268"},"Clone":{"machine_id_sha256":"3070269f14212259fdaafe96e69027560c54aa9bafa4fa9780e41dc5e2d613df","agent_spki_sha256":"96b5aa1abcac8e195c372670e44741b10e69dcea4c192eaaae1077987eb09918","netbird_public_key_sha256":"9288769d9627abeffc01a4f0b3fb4d4eab781cb6d159efce0601211c0f22f268"},"same_identity":true,"endpoint_rows":1,"conflicts":1,"mapping_frozen":true,"SourceStage":"enrolled","CloneStage":"enrolled"}
- snapshot: source stopped; durable state and native identity copied before clone boot; source resumed
