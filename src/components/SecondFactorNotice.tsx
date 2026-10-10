import { Alert, Card, Group, Stack, Text, Title } from '@mantine/core'
import { useTranslation } from 'react-i18next'

import type { SiteWorkspaceResource } from '../generated/site/types.gen.ts'
import { securityRoute } from '../router.tsx'
import { formatDate } from '../utils/datetime.ts'
import { ButtonLink } from './RouterLink.tsx'

// Where the signed-in User stands under a Workspace's Two-factor requirement (ADR
// 0020): no deadline applies (no requirement, or they have a Second factor), they
// are in grace until a deadline, or their grace is over and the Workspace is
// withheld until they enroll.
export type SecondFactorStanding =
  | { kind: 'clear' }
  | { kind: 'grace'; endsAt: string }
  | { kind: 'blocked' }

export function secondFactorStanding(
  workspace: Pick<SiteWorkspaceResource, 'secondFactorGraceEndsAt'> | undefined,
  now: number,
): SecondFactorStanding {
  const endsAt = workspace?.secondFactorGraceEndsAt
  if (!endsAt) return { kind: 'clear' }
  return Date.parse(endsAt) <= now ? { kind: 'blocked' } : { kind: 'grace', endsAt }
}

// SecondFactorGraceBanner reminds the User, on every page of the Workspace, of the
// date by which they must set up a Second factor.
export function SecondFactorGraceBanner({ endsAt }: { endsAt: string }) {
  const { t } = useTranslation()
  return (
    <Alert color="yellow" title={t(($) => $.secondFactorRequirement.graceTitle)} mb="md">
      <Stack gap="xs" align="flex-start">
        <Text size="sm">
          {t(($) => $.secondFactorRequirement.graceDescription, { date: formatDate(endsAt) })}
        </Text>
        <ButtonLink to={securityRoute.to} size="xs" variant="light" color="yellow">
          {t(($) => $.secondFactorRequirement.setUp)}
        </ButtonLink>
      </Stack>
    </Alert>
  )
}

// SecondFactorBlocked replaces the Workspace's pages once the grace is over: every
// request to the Workspace is refused until the User enrolls a Second factor.
export function SecondFactorBlocked() {
  const { t } = useTranslation()
  return (
    <Card withBorder maw={560} mx="auto" mt="xl" p="xl">
      <Stack>
        <Title order={3}>{t(($) => $.secondFactorRequirement.blockedTitle)}</Title>
        <Text c="dimmed">{t(($) => $.secondFactorRequirement.blockedDescription)}</Text>
        <Group>
          <ButtonLink to={securityRoute.to}>{t(($) => $.secondFactorRequirement.setUp)}</ButtonLink>
        </Group>
      </Stack>
    </Card>
  )
}
