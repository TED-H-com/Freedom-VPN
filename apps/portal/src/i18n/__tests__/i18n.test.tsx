import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { en, I18nProvider, LOCALE_STORAGE_KEY, useI18n, zhTW } from '../index';

function TestConsumer(): React.ReactElement {
  const { locale, setLocale, t } = useI18n();
  return (
    <div>
      <span data-testid="current-locale">{locale}</span>
      <span data-testid="saved-phrase">{t('op.savedPendingApply')}</span>
      <button type="button" onClick={() => setLocale('en')}>
        Switch to EN
      </button>
    </div>
  );
}

describe('src/i18n (§18 Locale Parity & Spec Phrases)', () => {
  it('TestLocaleParity_ZhTWAndEnHaveIdenticalKeysAndSpecPhrases', () => {
    const zhKeys = Object.keys(zhTW).sort();
    const enKeys = Object.keys(en).sort();

    expect(zhKeys).toEqual(enKeys);

    for (const key of zhKeys) {
      const k = key as keyof typeof zhTW;
      expect(zhTW[k].trim().length).toBeGreaterThan(0);
      expect(en[k].trim().length).toBeGreaterThan(0);
    }

    // Verify exact required spec phrases in zh-TW
    expect(zhTW['op.savedPendingApply']).toBe('已儲存，等待套用');
    expect(zhTW['op.routeWithdrawnDeviceUnconfirmed']).toBe(
      '中央已撤銷路由；設備撤權尚未確認／受租約限制',
    );
    expect(zhTW['member.service.fupThrottledBanner']).toContain(
      '本期大量使用，速度已依公平使用政策調整；下次重置為……',
    );
    expect(zhTW['node.summary.online_safety_failed_not_serving']).toBe(
      '在線，隔離檢查失敗，未提供服務',
    );
    expect(zhTW['member.billing.returnPendingBadge']).toBe('確認中');
  });

  it('TestLocaleSwitch_DefaultsToZhTWAndPersistsLanguageSwitch', async () => {
    const user = userEvent.setup();
    render(
      <I18nProvider>
        <TestConsumer />
      </I18nProvider>,
    );

    expect(screen.getByTestId('current-locale')).toHaveTextContent('zh-TW');
    expect(screen.getByTestId('saved-phrase')).toHaveTextContent('已儲存，等待套用');
    expect(document.documentElement.lang).toBe('zh-TW');

    await user.click(screen.getByRole('button', { name: 'Switch to EN' }));

    expect(screen.getByTestId('current-locale')).toHaveTextContent('en');
    expect(screen.getByTestId('saved-phrase')).toHaveTextContent(
      'Saved, waiting to apply',
    );
    expect(window.localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('en');
    expect(document.documentElement.lang).toBe('en');
  });
});
