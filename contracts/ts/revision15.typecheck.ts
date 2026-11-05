import type { components } from "./openapi";

type Report = components["schemas"]["ObservationReport"];
type Detail = components["schemas"]["NodeDetail"];
type ServingRead = components["schemas"]["ServingReadEvidence"];

const reportServingReads: Pick<Report, "serving_read">[] = [
  {},
  { serving_read: null },
  {
    serving_read: {
      read_at: "2026-10-28T15:00:25Z",
      source: "apply",
      candidate_generation: "128",
      verified: true,
      diagnostic: 'read_at=2026-10-28T15:00:25Z source=apply candidate=128 verified=true {"generation":"128","phase":"verify","reads":2,"objects":4,"truncated":false,"sha256":"dc031120dc666039402e235c0416643a9d76d09c41ef57f4b2cfef65378033a1"}',
    },
  },
  {
    serving_read: {
      read_at: "2026-10-28T15:00:26Z",
      source: "live",
      candidate_generation: "9007199254740993",
      verified: true,
      diagnostic: 'read_at=2026-10-28T15:00:26Z source=live candidate=9007199254740993 verified=true {"generation":"9007199254740993","phase":"verify","reads":2,"objects":4,"truncated":false,"sha256":"dc031120dc666039402e235c0416643a9d76d09c41ef57f4b2cfef65378033a1"}',
    },
  },
];

const detailServingReads: Pick<Detail, "serving_read">[] = [
  {},
  { serving_read: null },
  {
    serving_read: {
      read_at: "2026-10-28T15:00:28Z",
      source: "live",
      candidate_generation: "18446744073709551615",
      verified: false,
      diagnostic: 'read_at=2026-10-28T15:00:28Z source=live candidate=18446744073709551615 verified=false {"generation":"18446744073709551615","phase":"verify","reads":2,"objects":5,"truncated":false,"sha256":"6d78e62925fc23e6d6864f986fa220987447329d5b109c7b71aa0958a2d57c91","rejected":{"interface":"fv0","attachment":"ingress","objects":[{"kind":"flower","handle":"0x1","parent":"ffff:fff1","actions":["gact"]}]}}',
    },
  },
];

const sources: ServingRead["source"][] = ["apply", "live"];
// @ts-expect-error Only apply and live are valid ServingReadEvidence sources.
const badSource: ServingRead["source"] = "rollback";

const candidateGen: ServingRead["candidate_generation"] = "18446744073709551615";
// @ts-expect-error Candidate generation must be a decimal string, never a Number.
const badCandidateGen: ServingRead["candidate_generation"] = 128;

const badErrorCode: ServingRead = {
  read_at: "2026-10-28T15:00:25Z",
  source: "apply",
  candidate_generation: "128",
  verified: true,
  diagnostic: "read_at=2026-10-28T15:00:25Z source=apply candidate=128 verified=true {}",
  // @ts-expect-error ServingReadEvidence has no error_code property (D-164).
  error_code: null,
};

void [reportServingReads, detailServingReads, sources, badSource, candidateGen, badCandidateGen, badErrorCode];
