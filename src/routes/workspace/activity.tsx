import { useTranslation } from 'react-i18next'
import { ComingSoon } from '../../components/ComingSoon.tsx'

// Placeholder — the real events list lands in a later phase.
export function ActivityPage() {
  const { t } = useTranslation()
  return <ComingSoon title={t(($) => $.nav.activity)} />
}
