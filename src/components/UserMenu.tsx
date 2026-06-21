import { Avatar, Group, Menu, Text, UnstyledButton } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconChevronDown, IconLogout, IconSettings, IconUser } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { loginRoute, profileRoute, settingsRoute } from '../router.tsx'

// UserMenu is the header account dropdown: profile, workspace settings, logout.
// `slug` is the active workspace (absent in the account layout); the workspace
// settings item only renders when one is set. Logout hits go-pkgz/auth at
// /auth/logout (outside the generated /site client), which clears the JWT
// cookie; we then drop cached queries and route to login.
export function UserMenu({ slug, label }: { slug?: string; label?: string }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const handleLogout = async () => {
    try {
      await fetch('/auth/logout')
    } catch {
      notifications.show({
        color: 'red',
        title: t(($) => $.userMenu.logoutErrorTitle),
        message: t(($) => $.userMenu.logoutErrorTitle),
      })
      return
    }
    queryClient.clear()
    await navigate({ to: loginRoute.to })
  }

  return (
    <Menu position="bottom-end" withinPortal>
      <Menu.Target>
        <UnstyledButton>
          <Group gap="xs">
            <Avatar radius="xl" size={32} color="blue">
              {label ? label.charAt(0).toUpperCase() : <IconUser size={18} />}
            </Avatar>
            {label ? <Text size="sm">{label}</Text> : null}
            <IconChevronDown size={16} />
          </Group>
        </UnstyledButton>
      </Menu.Target>

      <Menu.Dropdown>
        <Menu.Item
          leftSection={<IconUser size={16} />}
          onClick={() => navigate({ to: profileRoute.to })}
        >
          {t(($) => $.userMenu.profile)}
        </Menu.Item>
        {slug ? (
          <Menu.Item
            leftSection={<IconSettings size={16} />}
            onClick={() => navigate({ to: settingsRoute.to, params: { slug } })}
          >
            {t(($) => $.userMenu.workspaceSettings)}
          </Menu.Item>
        ) : null}
        <Menu.Divider />
        <Menu.Item color="red" leftSection={<IconLogout size={16} />} onClick={handleLogout}>
          {t(($) => $.userMenu.logout)}
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  )
}
