# export-clone

Pre-boot cloned enrolled image and hash-only machine identity

Status: pass

Run: 12d10041-9d33-4fbb-a4b0-9d2a21bce785

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §6.2, AT07, ENR-07

- clone_observation: {"Source":{"machine_id_sha256":"5acba1b7936dd985a7b31462044eeaeba12adcd02fcd8140b2e35348c8f255ee","agent_spki_sha256":"8f780631d027a5ea9b866b6bb0b035db692a18072a8021a811f8ba7bd809be46","netbird_public_key_sha256":"c27d840b63ec21025f61f3a515d8ce4bc0d1032ffe334f40dbf48e3d2d81fe55"},"Clone":{"machine_id_sha256":"5acba1b7936dd985a7b31462044eeaeba12adcd02fcd8140b2e35348c8f255ee","agent_spki_sha256":"8f780631d027a5ea9b866b6bb0b035db692a18072a8021a811f8ba7bd809be46","netbird_public_key_sha256":"c27d840b63ec21025f61f3a515d8ce4bc0d1032ffe334f40dbf48e3d2d81fe55"},"same_identity":true,"endpoint_rows":1,"conflicts":0,"mapping_frozen":false,"SourceStage":"enrolled","CloneStage":"enrolled"}
- snapshot: source stopped; durable state and native identity copied before clone boot; source resumed
