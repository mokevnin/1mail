import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from '../locales/en/translation.json' with { type: 'json' }
import es from '../locales/es/translation.json' with { type: 'json' }
import ru from '../locales/ru/translation.json' with { type: 'json' }

// Locale is a property of the instance (a SaaS deployment runs per country),
// fixed at startup — there is no in-app language switcher and no browser
// detection. The backend substitutes window.__APP_LOCALE__ into index.html at
// serve time (see internal/server/spa.go); Vite does the same in dev. We read it
// synchronously here, so no extra request is needed at boot.
export const SUPPORTED_LOCALES = ['en', 'ru', 'es'] as const
export type Locale = (typeof SUPPORTED_LOCALES)[number]

export function resolveLocale(): Locale {
  const injected = (window as { __APP_LOCALE__?: string }).__APP_LOCALE__
  return SUPPORTED_LOCALES.includes(injected as Locale) ? (injected as Locale) : 'en'
}

// The app runs on a DEDICATED i18next instance rather than the global default.
// @workflowbuilder/sdk calls i18next.init() on the global default at import time,
// which rebuilds the resource store and wipes our `translation` namespace. Owning
// our own instance keeps the app's strings independent of the SDK. Components must
// reach it through the <I18nextProvider> in main.tsx (the SDK subtree is wrapped in
// its own provider bound back to the global instance it initialises).
const i18n = i18next.createInstance()

// index.html ships a static lang="en" (so the a11y lint accepts a real code);
// reflect the actual instance locale onto <html lang> once we know it.
document.documentElement.lang = resolveLocale()

void i18n.use(initReactI18next).init({
  lng: resolveLocale(),
  fallbackLng: 'en',
  resources: {
    en: { translation: en },
    ru: { translation: ru },
    es: { translation: es },
  },
  interpolation: {
    escapeValue: false,
  },
})

export { i18n }
