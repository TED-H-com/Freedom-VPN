import type { FairUseStatus } from '../../api';
import { useI18n, type TranslationKey } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';

/** Shared member notice: reason and recovery time come from usage (§11.6, §18.4). */
export function MemberFairUseNotice({
  fairUse,
  testId,
}: {
  fairUse: FairUseStatus | undefined;
  testId: string;
}) {
  const { t, locale } = useI18n();
  if (!fairUse) return null;

  let dated: TranslationKey;
  let undated: TranslationKey;
  switch (fairUse.reason) {
    case 'fair_use_tier':
      dated = 'member.service.fupThrottledBanner';
      undated = 'member.service.fupThrottledWithoutDate';
      break;
    case 'admin_override':
      dated = 'member.service.fupOverrideBanner';
      undated = 'member.service.fupOverrideWithoutDate';
      break;
    default:
      dated = 'member.service.fupAdjustedBanner';
      undated = 'member.service.fupAdjustedWithoutDate';
  }

  return (
    <>
      {fairUse.state === 'adjusted' && (
        <div className="alert alert-warning" role="status" data-testid={testId}>
          {fairUse.resets_at
            ? t(dated, { resetAt: formatLocaleWithUtc(fairUse.resets_at, locale) })
            : t(undated)}
        </div>
      )}
      {/* §11.2: a normal policy can also be awaiting exit acknowledgements. */}
      {fairUse.applying && (
        <div className="alert alert-info" role="status">
          {t('member.service.fupApplying')}
        </div>
      )}
    </>
  );
}
