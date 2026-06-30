import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
import translation from '../locales/en/translation.json' with { type: 'json' }

// The app runs on a DEDICATED i18next instance rather than the global default.
// @workflowbuilder/sdk calls i18next.init() on the global default at import time,
// which rebuilds the resource store and wipes our `translation` namespace. Owning
// our own instance keeps the app's strings independent of the SDK. Components must
// reach it through the <I18nextProvider> in main.tsx (the SDK subtree is wrapped in
// its own provider bound back to the global instance it initialises).
const i18n = i18next.createInstance()

void i18n.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: 'en',
  resources: {
    en: {
      translation,
    },
  },
  interpolation: {
    escapeValue: false,
  },
})

export { i18n }
