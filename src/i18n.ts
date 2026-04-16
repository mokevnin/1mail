import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import translation from '../locales/en/translation.json' with { type: 'json' }

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
