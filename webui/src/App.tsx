import { useTranslation } from 'react-i18next'
import { NavLink, Route, Routes, Navigate } from 'react-router-dom'
import EditorPage from './pages/EditorPage'
import SettingsPage from './pages/SettingsPage'
import { useAppStore } from './store/appStore'

const navCls = ({ isActive }: { isActive: boolean }) =>
  `px-3 py-2 text-sm rounded-md ${isActive ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-400 hover:text-zinc-200'}`

export default function App() {
  const { t } = useTranslation()
  const status = useAppStore((s) => s.status)

  return (
    <div className="flex flex-col h-full">
      <header className="shrink-0 flex items-center gap-2 px-3 py-2 border-b border-zinc-800 bg-zinc-900/90">
        <span className="font-semibold text-zinc-200 mr-4">NasPanel Pro</span>
        <nav className="flex gap-1">
          <NavLink to="/editor" className={navCls}>
            {t('nav.editor')}
          </NavLink>
          <NavLink to="/settings" className={navCls}>
            {t('nav.settings')}
          </NavLink>
        </nav>
        {status ? (
          <span className="ml-auto text-xs text-emerald-400/90 truncate max-w-md">{status}</span>
        ) : null}
      </header>
      <main className="flex-1 min-h-0 overflow-auto">
        <Routes>
          <Route path="/" element={<Navigate to="/editor" replace />} />
          <Route path="/connections" element={<Navigate to="/settings" replace />} />
          <Route path="/editor" element={<EditorPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </main>
    </div>
  )
}
