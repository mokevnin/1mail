import { useTranslation } from 'react-i18next'
import { ComingSoon } from '../../components/ComingSoon.tsx'

// Placeholder — workspace settings (rename, tracking snippet, API keys) land in
// later phases.
export function SettingsPage() {
  const { t } = useTranslation()
  return <ComingSoon title={t(($) => $.nav.settings)} />
}
