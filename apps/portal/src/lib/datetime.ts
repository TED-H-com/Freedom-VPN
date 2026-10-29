/**
 * Locale-aware timestamp formatting that renders dates in the user's selected
 * locale (`zh-TW` or `en`) alongside the canonical UTC RFC 3339 ledger timestamp (§14.1, §18.4).
 */

export function formatLocaleWithUtc(
  isoString: string | null | undefined,
  locale: 'zh-TW' | 'en' = 'zh-TW',
): string {
  if (!isoString) {
    return '—';
  }
  const date = new Date(isoString);
  if (Number.isNaN(date.getTime())) {
    return isoString;
  }
  try {
    const formatted = new Intl.DateTimeFormat(locale, {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(date);
    return `${formatted} (${isoString})`;
  } catch {
    return isoString;
  }
}
