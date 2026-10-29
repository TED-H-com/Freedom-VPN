import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react';
import { zhTW, type TranslationKey } from './zh-TW';
import { en } from './en';

export type Locale = 'zh-TW' | 'en';

export const LOCALE_STORAGE_KEY = 'fvpn_portal_locale';

const DICTIONARIES: Record<Locale, Record<TranslationKey, string>> = {
  'zh-TW': zhTW,
  en,
};

export type TranslateFn = (
  key: TranslationKey,
  params?: Record<string, string | number>,
) => string;

interface I18nContextValue {
  locale: Locale;
  setLocale: (next: Locale) => void;
  t: TranslateFn;
}

const I18nContext = createContext<I18nContextValue | null>(null);

function readInitialLocale(): Locale {
  try {
    const saved = window.localStorage.getItem(LOCALE_STORAGE_KEY);
    if (saved === 'en' || saved === 'zh-TW') {
      return saved;
    }
  } catch {
    // Ignore storage errors
  }
  return 'zh-TW';
}

export function I18nProvider({
  children,
  initialLocale,
}: {
  children: React.ReactNode;
  initialLocale?: Locale;
}): React.ReactElement {
  const [locale, setLocaleState] = useState<Locale>(
    () => initialLocale ?? readInitialLocale(),
  );

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
    try {
      window.localStorage.setItem(LOCALE_STORAGE_KEY, next);
    } catch {
      // Ignore storage errors
    }
  }, []);

  useEffect(() => {
    if (typeof document !== 'undefined') {
      document.documentElement.lang = locale;
    }
  }, [locale]);

  const t: TranslateFn = useCallback(
    (key, params) => {
      const dict = DICTIONARIES[locale] ?? zhTW;
      let text: string = dict[key] ?? zhTW[key] ?? key;
      if (params) {
        for (const [k, v] of Object.entries(params)) {
          text = text.replaceAll(`{${k}}`, String(v));
        }
      }
      return text;
    },
    [locale],
  );

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error('useI18n must be used within an I18nProvider');
  }
  return ctx;
}

export { zhTW, en };
export type { TranslationKey };
