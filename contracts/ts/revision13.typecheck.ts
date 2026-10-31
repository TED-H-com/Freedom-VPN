import type { components } from "./openapi";

// D-089: existing callers can omit source even though its schema has a default.
// Inlining that default makes openapi-typescript require the property.
type ResultSchema = "UpdateResult" | "NodeUpdateSummary" | "UpdateTarget" | "UpdateJobSummary";
const legacyResults: {
  [Name in ResultSchema]: Pick<components["schemas"][Name], "source">;
} = {
  UpdateResult: {},
  NodeUpdateSummary: {},
  UpdateTarget: {},
  UpdateJobSummary: {},
};
void legacyResults;

const installer: components["schemas"]["UpdateResultSource"] = "installer";
const central: components["schemas"]["UpdateResultSource"] = "central";
// @ts-expect-error D-089 rejects unknown sources.
const unknown: components["schemas"]["UpdateResultSource"] = "unknown";
// @ts-expect-error Omission is allowed; explicit null is not.
const nullSource: components["schemas"]["UpdateResult"]["source"] = null;
void [installer, central, unknown, nullSource];

// D-088: omission and false remain distinct from a frozen incarnation.
const signals: Pick<components["schemas"]["ObservationAccepted"], "identity_frozen">[] = [
  {},
  { identity_frozen: false },
  { identity_frozen: true },
  // @ts-expect-error The freeze signal is non-nullable.
  { identity_frozen: null },
];
void signals;
