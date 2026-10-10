import { Stack, Text, Title } from '@mantine/core'
import { useTranslation } from 'react-i18next'

// The console home: empty until the Workspace list lands.
export function ConsoleHomePage() {
  const { t } = useTranslation()
  return (
    <Stack>
      <Title order={2}>{t(($) => $.console.home.title)}</Title>
      <Text c="dimmed">{t(($) => $.console.home.description)}</Text>
    </Stack>
  )
}
