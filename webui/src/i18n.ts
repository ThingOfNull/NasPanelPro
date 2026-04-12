import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en'
import zh from './locales/zh'

const STORAGE_KEY = 'naspanel_lang'

export function readStoredLang(): 'en' | 'zh' {
  const s = localStorage.getItem(STORAGE_KEY)
  if (s === 'en' || s === 'zh') {
    return s
  }
  const nav = navigator.language?.toLowerCase() ?? ''
  if (nav.startsWith('en')) {
    return 'en'
  }
  return 'zh'
}

export function setStoredLang(lng: 'en' | 'zh'): void {
  localStorage.setItem(STORAGE_KEY, lng)
  void i18n.changeLanguage(lng)
}

export async function initI18n(): Promise<void> {
  await i18n.use(initReactI18next).init({
    resources: {
      en: { translation: en },
      zh: { translation: zh },
    },
    lng: readStoredLang(),
    fallbackLng: 'zh',
    interpolation: { escapeValue: false },
  })
}

export default i18n
