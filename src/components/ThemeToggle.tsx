import { ActionIcon, useComputedColorScheme, useMantineColorScheme } from '@mantine/core'
import { IconMoon, IconSun } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

// ThemeToggle flips between light and dark color schemes. It reads the computed
// scheme (resolving "auto" to the system value) so the first click always moves
// to the opposite of what the user currently sees. Mantine persists the choice.
export function ThemeToggle() {
  const { t } = useTranslation()
  const { setColorScheme } = useMantineColorScheme()
  const computed = useComputedColorScheme('light', { getInitialValueInEffect: true })
  const label = t(($) => $.theme.toggleLabel)

  return (
    <ActionIcon
      variant="default"
      size="lg"
      aria-label={label}
      title={label}
      onClick={() => setColorScheme(computed === 'dark' ? 'light' : 'dark')}
    >
      {computed === 'dark' ? <IconSun size={18} /> : <IconMoon size={18} />}
    </ActionIcon>
  )
}
