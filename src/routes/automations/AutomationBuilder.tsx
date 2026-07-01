import { Box, Stack } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  type IntegrationDataFormat,
  type OnSaveExternal,
  WorkflowBuilder,
} from '@workflowbuilder/sdk'
// Imported as a raw string (not auto-injected): the SDK sheet carries unlayered
// global resets and .mantine-* overrides that would leak app-wide. We mount it
// only while the builder is on screen and remove it on unmount, so it never
// touches the rest of the app (e.g. the dashboard's body scroll).
import builderStyles from '@workflowbuilder/sdk/style.css?inline'
import i18next from 'i18next'
import { useCallback, useLayoutEffect, useMemo } from 'react'
import { I18nextProvider, useTranslation } from 'react-i18next'
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
import { graphToSteps, stepsToGraph } from './definition.ts'
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

  // Scope the SDK stylesheet to the builder's lifetime: present while mounted,
  // gone everywhere else. useLayoutEffect injects it before paint to avoid an
  // unstyled flash of the canvas.
  useLayoutEffect(() => {
    const style = document.createElement('style')
    style.dataset.workflowBuilder = ''
    style.textContent = builderStyles
    document.head.appendChild(style)
    return () => style.remove()
  }, [])

  const form = useForm<AutomationDetailsValues>({
    initialValues: { name: automation.name, triggerEvent: automation.triggerEvent },
  })

  const initialGraph = useMemo(() => stepsToGraph(automation.steps), [automation.steps])

  const updateMutation = useMutation({
    ...siteAutomationsUpdateMutation(),
    onSuccess: async (updated) => {
      await queryClient.invalidateQueries({
        queryKey: siteAutomationsListQueryKey({ path: { slug: slug } }),
      })
      await queryClient.invalidateQueries({
        queryKey: siteAutomationsGetQueryKey({ path: { slug: slug, id: updated.id } }),
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
          path: { slug: slug, id: automation.id },
          body: {
            name: values.name.trim(),
            triggerEvent: values.triggerEvent,
            steps,
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
        {/* The SDK initialises and reads the global i18next instance for its own
            strings. The app tree runs on a private instance (see main.tsx), so the
            builder subtree is bound back to the global one here. */}
        <I18nextProvider i18n={i18next}>
          <WorkflowBuilder.Root
            key={automation.id}
            name={automation.name}
            layoutDirection="DOWN"
            nodeTypes={nodeTypes}
            initialNodes={initialGraph.nodes}
            initialEdges={initialGraph.edges}
            integration={integration}
          />
        </I18nextProvider>
      </Box>
    </Stack>
  )
}
