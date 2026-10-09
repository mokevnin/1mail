import { expect, test } from 'vitest'

import { renderWithProviders } from '../test/renderWithProviders.tsx'
import { ThemeToggle } from './ThemeToggle.tsx'

const scheme = () => document.documentElement.getAttribute('data-mantine-color-scheme')

test('toggles the color scheme on the document', async () => {
  const screen = await renderWithProviders(<ThemeToggle />)
  const button = screen.getByRole('button', { name: 'Toggle color scheme' })

  await button.click()
  const first = scheme()
  expect(['light', 'dark']).toContain(first)

  await button.click()
  expect(scheme()).not.toBe(first)
})
