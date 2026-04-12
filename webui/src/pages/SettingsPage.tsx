import { useTranslation } from 'react-i18next'
import { setStoredLang } from '../i18n'
import DataSourcesSection from '../settings/DataSourcesSection'
import LayoutHardwareSection from '../settings/LayoutHardwareSection'
import { useAppStore } from '../store/appStore'

export default function SettingsPage() {
  const { t, i18n } = useTranslation()
  const { saveSettings, setStatus } = useAppStore()

  return (
    <div className="max-w-5xl mx-auto p-6 space-y-8">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">{t('settingsPage.title')}</h1>
        <button
          type="button"
          onClick={() => saveSettings().catch((e) => setStatus(String(e)))}
          className="px-4 py-2 rounded bg-emerald-700 hover:bg-emerald-600 text-sm shrink-0"
        >
          {t('settingsPage.saveAll')}
        </button>
      </div>

      <section className="space-y-3 border border-zinc-800 rounded-lg p-4">
        <h2 className="text-sm font-medium text-zinc-400">{t('settingsPage.language')}</h2>
        <div className="flex gap-4 text-sm">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="lang"
              checked={i18n.language === 'zh'}
              onChange={() => setStoredLang('zh')}
            />
            {t('settingsPage.langZh')}
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="lang"
              checked={i18n.language === 'en'}
              onChange={() => setStoredLang('en')}
            />
            {t('settingsPage.langEn')}
          </label>
        </div>
      </section>

      <DataSourcesSection />
      <LayoutHardwareSection />
    </div>
  )
}
