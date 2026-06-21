import { Stack, Text, Title } from '@mantine/core'
import { useTranslation } from 'react-i18next'

// ComingSoon is a lightweight placeholder for sections whose real page is built
// in a later phase (profile, settings, activity). Routes mount it so the shell
// navigation works end to end before the feature lands.
export function ComingSoon({ title }: { title?: string }) {
  const { t } = useTranslation()
  return (
    <Stack>
      <Title order={3}>{title ?? t(($) => $.comingSoon.title)}</Title>
      <Text c="dimmed">{t(($) => $.comingSoon.description)}</Text>
    </Stack>
  )
}
