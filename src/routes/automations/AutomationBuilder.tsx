import { Box, Stack } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  type IntegrationDataFormat,
  type OnSaveExternal,
  WorkflowBuilder,
} from '@workflowbuilder/sdk'
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  siteAutomationsGetQueryKey,
  siteAutomationsListQueryKey,
  siteAutomationsUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteAutomationResource } from '../../generated/site/types.gen.ts'
import { type ApiErrorLike, getApiErrorMessage } from '../../utils/apiErrors.ts'
import {
  AutomationDetailsFields,
  type AutomationDetailsValues,
} from './AutomationDetailsFields.tsx'
import { graphToSteps, parseSteps, serializeSteps, stepsToGraph } from './definition.ts'
import { useAutomationNodeTypes } from './nodes.tsx'

interface AutomationBuilderProps {
  slug: string
  automation: SiteAutomationResource
}

// AutomationBuilder hosts the visual flow editor. Name + trigger are edited as
// Mantine chrome above the canvas; the SDK's Save (top bar) drives onDataSave,
// which converts the graph to the linear []step definition and persists name,
// trigger and definition together through the generated update mutation.
export function AutomationBuilder({ slug, automation }: AutomationBuilderProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const nodeTypes = useAutomationNodeTypes()

  const form = useForm<AutomationDetailsValues>({
    initialValues: { name: automation.name, triggerEvent: automation.triggerEvent },
  })

  const initialGraph = useMemo(
    () => stepsToGraph(parseSteps(automation.definition)),
    [automation.definition],
  )

  const updateMutation = useMutation({
    ...siteAutomationsUpdateMutation(),
    onSuccess: async (updated) => {
      await queryClient.invalidateQueries({
        queryKey: siteAutomationsListQueryKey({ path: { workspaceSlug: slug } }),
      })
      await queryClient.invalidateQueries({
        queryKey: siteAutomationsGetQueryKey({ path: { workspaceSlug: slug, id: updated.id } }),
      })
    },
  })

  const onDataSave = useCallback<OnSaveExternal>(
    async (data: IntegrationDataFormat) => {
      const { steps, dropped } = graphToSteps({ nodes: data.nodes, edges: data.edges })
      if (dropped > 0) {
        notifications.show({
          color: 'yellow',
          title: t(($) => $.automations.branchesUnsupportedTitle),
          message: t(($) => $.automations.branchesUnsupported),
        })
      }
      const values = form.getValues()
      try {
        await updateMutation.mutateAsync({
          path: { workspaceSlug: slug, id: automation.id },
          body: {
            name: values.name.trim(),
            triggerEvent: values.triggerEvent,
            definition: serializeSteps(steps),
          },
        })
        notifications.show({
          color: 'teal',
          title: t(($) => $.notifications.successTitle),
          message: t(($) => $.notifications.automationUpdated),
        })
        return 'success'
      } catch (error) {
        notifications.show({
          color: 'red',
          title: t(($) => $.alerts.automationSaveErrorTitle),
          message: getApiErrorMessage(
            error as ApiErrorLike,
            t(($) => $.alerts.automationSaveErrorTitle),
          ),
        })
        return 'error'
      }
    },
    [slug, automation.id, form, updateMutation, t],
  )

  const integration = useMemo(() => ({ strategy: 'props' as const, onDataSave }), [onDataSave])

  return (
    <Stack>
      <AutomationDetailsFields form={form} />
      <Box pos="relative" h="calc(100vh - 220px)" w="100%">
        <WorkflowBuilder.Root
          key={automation.id}
          name={automation.name}
          layoutDirection="DOWN"
          nodeTypes={nodeTypes}
          initialNodes={initialGraph.nodes}
          initialEdges={initialGraph.edges}
          integration={integration}
        />
      </Box>
    </Stack>
  )
}
