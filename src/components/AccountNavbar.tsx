import { NavLink, Stack } from '@mantine/core'
import { IconArrowLeft, IconUser } from '@tabler/icons-react'
import { useMatchRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { indexRoute, profileRoute } from '../router.tsx'

// AccountNavbar is the sidebar for the workspace-independent account area:
// a way back to the workspace dashboard plus account sections (profile today).
export function AccountNavbar() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const matchRoute = useMatchRoute()

  return (
    <Stack gap="xs">
      <NavLink
        label={t(($) => $.account.backToDashboard)}
        leftSection={<IconArrowLeft size={18} />}
        onClick={() => navigate({ to: indexRoute.to })}
      />
      <NavLink
        label={t(($) => $.account.profile)}
        leftSection={<IconUser size={18} />}
        active={Boolean(matchRoute({ to: profileRoute.to }))}
        onClick={() => navigate({ to: profileRoute.to })}
      />
    </Stack>
  )
}
