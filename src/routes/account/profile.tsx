import { useTranslation } from 'react-i18next'
import { ComingSoon } from '../../components/ComingSoon.tsx'

// Placeholder — the real profile form lands in a later phase.
export function ProfilePage() {
  const { t } = useTranslation()
  return <ComingSoon title={t(($) => $.account.profile)} />
}
