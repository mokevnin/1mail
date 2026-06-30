import {
  Alert,
  Button,
  Card,
  Checkbox,
  Group,
  Loader,
  NumberInput,
  Select,
  Stack,
  Text,
  TextInput,
  Title,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { useQuery } from '@tanstack/react-query'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import {
  siteIntegrationsCreateMutation,
  siteIntegrationsDeleteMutation,
  siteIntegrationsListOptions,
  siteIntegrationsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteIntegrationConfigInput } from '../../generated/site/types.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'

type ProviderKind = 'smtp' | 'ses'

// Flat form state; the provider-specific config object is assembled on submit
// from whichever fields the selected provider uses.
interface IntegrationFormValues {
  name: string
  provider: ProviderKind
  isDefault: boolean
  host: string
  port: number
  username: string
  password: string
  region: string
  accessKeyId: string
  secretAccessKey: string
  endpoint: string
  from: string
  fromName: string
}

const INITIAL_VALUES: IntegrationFormValues = {
  name: '',
  provider: 'smtp',
  isDefault: false,
  host: '',
  port: 587,
  username: '',
  password: '',
  region: '',
  accessKeyId: '',
  secretAccessKey: '',
  endpoint: '',
  from: '',
  fromName: '',
}

function buildConfig(values: IntegrationFormValues): SiteIntegrationConfigInput {
  if (values.provider === 'ses') {
    return {
      kind: 'ses',
      region: values.region.trim(),
      accessKeyId: values.accessKeyId.trim(),
      secretAccessKey: values.secretAccessKey,
      endpoint: values.endpoint.trim() || null,
      from: values.from.trim(),
      fromName: values.fromName.trim() || null,
    }
  }
  return {
    kind: 'smtp',
    host: values.host.trim(),
    port: values.port,
    username: values.username.trim() || null,
    password: values.password || null,
    from: values.from.trim(),
    fromName: values.fromName.trim() || null,
  }
}

export function IntegrationsSection({ slug }: { slug: string }) {
  const { t } = useTranslation()
  const confirmDelete = useDeleteConfirmation()

  const queryKey = siteIntegrationsListQueryKey({ path: { workspaceSlug: slug } })
  const integrationsQuery = useQuery(siteIntegrationsListOptions({ path: { workspaceSlug: slug } }))

  const form = useForm<IntegrationFormValues>({ initialValues: INITIAL_VALUES })

  const createMutation = useResourceMutation({
    mutation: siteIntegrationsCreateMutation(),
    invalidate: [queryKey],
    successMessage: t(($) => $.settings.integrations.saved),
    errorTitle: t(($) => $.settings.integrations.createError),
    onDone: () => form.reset(),
  })

  const deleteMutation = useResourceMutation({
    mutation: siteIntegrationsDeleteMutation(),
    invalidate: [queryKey],
    errorTitle: t(($) => $.settings.integrations.deleteError),
  })

  const onDelete = (id: string) => {
    confirmDelete({
      title: t(($) => $.settings.integrations.deleteConfirmTitle),
      description: t(($) => $.settings.integrations.deleteConfirmDescription),
      onConfirm: () => deleteMutation.mutate({ path: { workspaceSlug: slug, id } }),
    })
  }

  const isSes = form.values.provider === 'ses'
  const records = integrationsQuery.data ?? []

  return (
    <Card withBorder>
      <Title order={4} mb="xs">
        {t(($) => $.settings.integrations.title)}
      </Title>
      <Text c="dimmed" size="sm" mb="md">
        {t(($) => $.settings.integrations.description)}
      </Text>

      {integrationsQuery.isError ? (
        <Alert color="red" title={t(($) => $.settings.integrations.loadError)} mb="md">
          {t(($) => $.settings.integrations.loadError)}
        </Alert>
      ) : null}

      <form
        onSubmit={form.onSubmit((values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
            body: {
              name: values.name.trim(),
              isDefault: values.isDefault,
              config: buildConfig(values),
            },
          }),
        )}
      >
        <Stack maw={520} mb="md">
          <Group grow>
            <TextInput
              label={t(($) => $.settings.integrations.nameLabel)}
              required
              {...form.getInputProps('name')}
            />
            <Select
              label={t(($) => $.settings.integrations.providerLabel)}
              data={[
                { value: 'smtp', label: 'SMTP' },
                { value: 'ses', label: 'Amazon SES' },
              ]}
              allowDeselect={false}
              {...form.getInputProps('provider')}
            />
          </Group>

          {isSes ? (
            <>
              <Group grow>
                <TextInput
                  label={t(($) => $.settings.integrations.fields.region)}
                  required
                  {...form.getInputProps('region')}
                />
                <TextInput
                  label={t(($) => $.settings.integrations.fields.accessKeyId)}
                  required
                  {...form.getInputProps('accessKeyId')}
                />
                <TextInput
                  label={t(($) => $.settings.integrations.fields.secretAccessKey)}
                  type="password"
                  required
                  {...form.getInputProps('secretAccessKey')}
                />
              </Group>
              <TextInput
                label={t(($) => $.settings.integrations.fields.endpoint)}
                description={t(($) => $.settings.integrations.fields.endpointHint)}
                placeholder="https://postbox.cloud.yandex.net"
                {...form.getInputProps('endpoint')}
              />
            </>
          ) : (
            <>
              <Group grow>
                <TextInput
                  label={t(($) => $.settings.integrations.fields.host)}
                  required
                  {...form.getInputProps('host')}
                />
                <NumberInput
                  label={t(($) => $.settings.integrations.fields.port)}
                  required
                  {...form.getInputProps('port')}
                />
              </Group>
              <Group grow>
                <TextInput
                  label={t(($) => $.settings.integrations.fields.username)}
                  {...form.getInputProps('username')}
                />
                <TextInput
                  label={t(($) => $.settings.integrations.fields.password)}
                  type="password"
                  {...form.getInputProps('password')}
                />
              </Group>
            </>
          )}

          <Group grow>
            <TextInput
              label={t(($) => $.settings.integrations.fields.from)}
              required
              {...form.getInputProps('from')}
            />
            <TextInput
              label={t(($) => $.settings.integrations.fields.fromName)}
              {...form.getInputProps('fromName')}
            />
          </Group>

          <Checkbox
            label={t(($) => $.settings.integrations.makeDefault)}
            {...form.getInputProps('isDefault', { type: 'checkbox' })}
          />

          <Group justify="flex-end">
            <Button type="submit" loading={createMutation.isPending}>
              {t(($) => $.settings.integrations.create)}
            </Button>
          </Group>
        </Stack>
      </form>

      {integrationsQuery.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        idAccessor="id"
        columns={[
          { accessor: 'name' },
          { accessor: 'provider', title: t(($) => $.settings.integrations.providerLabel) },
          {
            accessor: 'isDefault',
            title: t(($) => $.settings.integrations.default),
            render: (record) => (record.isDefault ? t(($) => $.settings.integrations.default) : ''),
          },
          {
            accessor: 'actions',
            title: '',
            render: (record) => (
              <Button
                size="compact-sm"
                color="red"
                variant="light"
                onClick={() => onDelete(record.id)}
              >
                {t(($) => $.settings.integrations.delete)}
              </Button>
            ),
          },
        ]}
        noRecordsText={t(($) => $.settings.integrations.empty)}
      />
    </Card>
  )
}
