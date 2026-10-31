import type { components, operations } from "./openapi";

type Report = components["schemas"]["ObservationReport"];
const optionalEvidence: Pick<Report, "tc_evidence">[] = [{}, { tc_evidence: null }];
type Evidence = components["schemas"]["TCReadbackEvidence"];
const generation: Evidence["candidate_generation"] = "18446744073709551615";
// @ts-expect-error Candidate generation never passes through Number.
const numericGeneration: Evidence["candidate_generation"] = 128;
const phases: Evidence["phase"][] = ["load", "pre_apply", "verify"];
// @ts-expect-error No invented phase.
const badPhase: Evidence["phase"] = "rollback";
const origins: Pick<components["schemas"]["UsageBatchRequest"], "origin_incarnation_id">[] = [
  {}, { origin_incarnation_id: "01000000-0000-4000-8000-000000000014" },
  // @ts-expect-error Absence is valid but explicit null is not.
  { origin_incarnation_id: null },
];
type Revoke = operations["revokeMemberGrant"];
const grantID: Revoke["parameters"]["path"]["grant_id"] = "01000000-0000-4000-8000-000000000029";
void [optionalEvidence, generation, numericGeneration, phases, badPhase, origins, grantID];
