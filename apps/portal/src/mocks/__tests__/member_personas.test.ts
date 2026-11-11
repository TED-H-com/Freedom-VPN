import { expect, it } from 'vitest';
import { buildMemberSelfForPersona, type MockPersona } from '../data';
import type { EntitlementState } from '../../api';

const personas = {
  anonymous: 'active',
  admin: 'active',
  admin_no_mfa: 'active',
  member_active: 'active',
  member_throttled: 'active',
  member_pending_payment: 'pending_payment',
  member_cancel_at_period_end: 'cancel_at_period_end',
  member_grace: 'grace',
  member_expired: 'expired',
  member_suspended: 'suspended',
} satisfies Record<MockPersona, EntitlementState>;

for (const persona of Object.keys(personas) as MockPersona[]) {
  it(`TestAT04_F404_MemberPersonaMatchesSelfRead_${persona}`, () => {
    const me = buildMemberSelfForPersona(persona);
    const ent = me.entitlement;
    expect(ent.state).toBe(personas[persona]);
    expect(me.limits.device_limit).toBe(ent.device_limit);
    expect(me.limits.authorized_device_count).toBeGreaterThanOrEqual(0);
    if (me.subscription) {
      expect(ent.paid_through).toBe(me.subscription.paid_through);
      expect(ent.cancel_at_period_end).toBe(me.subscription.cancel_at_period_end);
    }
    const serving = ['active', 'cancel_at_period_end', 'grace'].includes(ent.state);
    if (serving) {
      expect(me.limits.exit_count).toBeGreaterThan(0);
      const now = new Date(ent.computed_at);
      expect(Date.parse(ent.valid_from!)).toBeLessThanOrEqual(now.getTime());
      expect(Date.parse(ent.effective_service_end!)).toBeGreaterThan(now.getTime());
      expect(Date.parse(me.fair_use_reset_at!)).toBe(
        Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1),
      );
      expect(me.next_actions).toEqual(
        me.limits.authorized_device_count >= me.limits.device_limit!
          ? ['free_device_slot']
          : ['install_app', 'register_device'],
      );
    } else {
      expect(me.limits.exit_count).toBe(0);
      expect(me.fair_use_reset_at).toBeNull();
      if (ent.state === 'pending_payment') {
        expect(me.subscription).toBeNull();
        for (const value of [
          ent.plan,
          ent.device_limit,
          ent.valid_from,
          ent.effective_service_end,
          ent.paid_through,
          ent.grace_until,
        ])
          expect(value).toBeNull();
        expect(me.next_actions).toEqual(['wait_for_payment_confirmation']);
      } else {
        expect(me.next_actions).toEqual(
          ent.state === 'suspended' || ent.state === 'closed'
            ? ['contact_support']
            : ['renew_subscription'],
        );
      }
    }
    expect(ent.admin_suspended).toBe(ent.state === 'suspended');
    if (ent.state === 'grace') {
      expect(me.subscription?.provider_status).toBe('past_due');
      expect(ent.valid_from).toBe(ent.paid_through);
      expect(Date.parse(ent.grace_until!)).toBeGreaterThan(Date.parse(ent.computed_at));
    }
    if (ent.state === 'expired') {
      expect(Date.parse(ent.effective_service_end!)).toBeLessThanOrEqual(
        Date.parse(ent.computed_at),
      );
      // The last elapsed span in this fixture is the subscription's grace span.
      expect(ent.valid_from).toBe(ent.paid_through);
      expect(ent.effective_service_end).toBe(ent.grace_until);
    }
  });
}
