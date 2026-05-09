import { Text } from '@mantine/core'
import { modals } from '@mantine/modals'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

type DeleteConfirmationOptions = {
  title?: ReactNode
  description?: ReactNode
  onConfirm: () => void
}

export function useDeleteConfirmation() {
  const { t } = useTranslation()

  return ({ title, description, onConfirm }: DeleteConfirmationOptions) => {
    modals.openConfirmModal({
      title: title ?? t(($) => $.confirmations.deleteTitle),
      children: <Text>{description ?? t(($) => $.confirmations.deleteDescription)}</Text>,
      labels: {
        cancel: t(($) => $.actions.cancel),
        confirm: t(($) => $.actions.delete),
      },
      confirmProps: { color: 'red' },
      onConfirm,
    })
  }
}
