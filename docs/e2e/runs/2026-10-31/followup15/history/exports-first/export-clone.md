# export-clone

Pre-boot cloned enrolled image and hash-only machine identity

Status: pass

Run: 926791b9-36ef-4051-9d44-cd7eafdc8792

Source: 9e493e8f6407dd7f091963fb7e133a11123305c3

Spec: §6.2, AT07, ENR-07

- clone_observation: {"Source":{"machine_id_sha256":"ac701114d484f446374532a5bdd721016b9fbcc1f6ee1b50427bcd24103fd136","agent_spki_sha256":"abe386e0ae91621c0df569a2b383448b4ecdac0bd14028934812c6303665294a","netbird_public_key_sha256":"3863bfe5f0e39be9267b8e3de9425008d31e2fa20ea64ac63087d56536567e43"},"Clone":{"machine_id_sha256":"ac701114d484f446374532a5bdd721016b9fbcc1f6ee1b50427bcd24103fd136","agent_spki_sha256":"abe386e0ae91621c0df569a2b383448b4ecdac0bd14028934812c6303665294a","netbird_public_key_sha256":"3863bfe5f0e39be9267b8e3de9425008d31e2fa20ea64ac63087d56536567e43"},"same_identity":true,"endpoint_rows":1,"conflicts":1,"mapping_frozen":true,"SourceStage":"enrolled","CloneStage":"enrolled"}
- snapshot: source stopped; durable state and native identity copied before clone boot; source resumed
