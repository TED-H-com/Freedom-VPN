// TEST ONLY: slow runners need margin to observe active; not a product minimum.
export const MEMBER_SCREEN_GRANT_WINDOW_MS = 5000;
export const MEMBER_SCREEN_BUDGET = {
  newMember: 15_000,
  pending: 5000,
  grant: 10_000,
  grantUpstream: 8000,
  expiryAllowance: 5000,
  navigation: 5000,
  renderPerLocale: 5000,
  capturePerLocale: 5000,
  teardown: 15_000,
  teardownRequest: 4000,
  expiryTest: 90_000,
};
// Expiry body upper bound: new member 15 + pending 5 + grant 10 + active 5
// + expiry 10 (valid_until + 5s allowance) + navigation 5 + 2 renders * 5
// + 2 optional captures * 5 = 70s. Teardown has a separate 15s fixture slot:
// conservative total 85s < 90s for this test alone (suite default stays 45s).
// Active/expiry windows overlap, so this deliberately overcounts elapsed time.
