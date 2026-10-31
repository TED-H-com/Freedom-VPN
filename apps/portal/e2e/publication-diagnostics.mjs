// Keep diagnostics as free text; absence of a message never proves success.
export function publicationGateDiagnostics(operation) {
  if (operation?.kind !== 'node.publish') return [];
  return (operation.steps ?? [])
    .filter(
      (step) =>
        step.status === 'pending' &&
        (step.name === 'agent_policy' || step.name === 'data_path_probe'),
    )
    .map((step) => ({
      name: step.name,
      status: step.status,
      error_code: step.error_code ?? null,
      message: step.message ?? null,
      diagnostic: [step.error_code, step.message].filter(Boolean).join(': '),
    }));
}
