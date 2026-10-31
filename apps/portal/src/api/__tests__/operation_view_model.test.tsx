import { act, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  apiClient,
  buildOperationStepViewModel,
  isOperationTerminal,
  type OperationStep,
} from '..';
import { OperationBanner } from '../../components/common/OperationBanner';
import { I18nProvider } from '../../i18n';
import { PUBLICATION_GATE_OPERATION } from '../../mocks/data';

describe('D-074 / D-107 publication gate diagnostics (§5.2, §14.8)', () => {
  it.each([
    'stale_generation',
    'check_failed',
    'forwarding_disabled',
    'tunnel_not_passed',
    'isolation_unconfirmed',
    'readiness_unconfirmed',
    'serving_checks_unconfirmed',
    'lease_unconfirmed',
    'future diagnostic text',
  ])('preserves the free-text cause %s without changing pending status', (cause) => {
    const op = structuredClone(PUBLICATION_GATE_OPERATION);
    const gate = op.steps.find((step) => step.name === 'data_path_probe')!;
    gate.message = `member_forwarding_unconfirmed (${cause})`;
    const view = buildOperationStepViewModel(gate);
    expect(view.diagnostic).toBe(`NODE_NOT_SERVICEABLE: ${gate.message}`);
    expect(view.status).toBe('pending');
    expect(view.observed_at).toBeNull();
    expect(op.status).toBe('running');
    expect(isOperationTerminal(op.status)).toBe(false);
    expect(op.error).toBeNull();
    expect(op.completed_at).toBeNull();
  });

  it.each([null, undefined, ''])(
    'missing message %s never passes the gate',
    (message) => {
      const gate = PUBLICATION_GATE_OPERATION.steps.find(
        (step) => step.name === 'data_path_probe',
      )!;
      // Simulate nullable and omitted wire fields without widening generated types.
      const wireStep: OperationStep = JSON.parse(JSON.stringify({ ...gate, message }));
      expect(buildOperationStepViewModel(wireStep)).toMatchObject({
        status: 'pending',
        observed_at: null,
        diagnostic: 'NODE_NOT_SERVICEABLE',
      });
      expect(
        buildOperationStepViewModel({ ...wireStep, error_code: null }),
      ).toMatchObject({ status: 'pending', diagnostic: '' });
    },
  );

  it.each(['agent_policy', 'data_path_probe'])(
    'keeps %s diagnostic visible through polling timeouts, then accepts fresh completion',
    async (name) => {
      vi.useFakeTimers();
      try {
        const op = structuredClone(PUBLICATION_GATE_OPERATION);
        const gate = op.steps.find((step) => step.name === 'data_path_probe')!;
        op.steps = [{ ...gate, name }];
        const poll = vi
          .spyOn(apiClient, 'getOperation')
          .mockRejectedValue(new Error('timeout'));
        const terminal = vi.fn();
        render(
          <I18nProvider>
            <OperationBanner operation={op} onTerminal={terminal} />
          </I18nProvider>,
        );
        await act(() => vi.advanceTimersByTimeAsync(30_000));
        const banner = screen.getByTestId('operation-banner');
        expect(banner).toHaveAttribute('data-operation-status', 'running');
        expect(banner).toHaveClass('alert-info');
        expect(within(banner).getByTestId(`op-step-${name}`)).toHaveTextContent(
          'NODE_NOT_SERVICEABLE: member_forwarding_unconfirmed (lease_unconfirmed)',
        );
        expect(terminal).not.toHaveBeenCalled();
        expect(poll).toHaveBeenCalled();

        poll.mockResolvedValue({
          ...op,
          status: 'succeeded',
          completed_at: '2026-10-31T08:00:00Z',
          steps: [
            { ...gate, name, status: 'succeeded', error_code: null, message: null },
          ],
        });
        await act(() => vi.advanceTimersByTimeAsync(2100));
        expect(banner).toHaveAttribute('data-operation-status', 'succeeded');
        expect(terminal).toHaveBeenCalledOnce();
      } finally {
        vi.useRealTimers();
        vi.restoreAllMocks();
      }
    },
  );
});
