import { expect, test } from 'vitest'

import type {
  SiteSecondFactorConfirmEnrollmentData,
  SiteSecondFactorDisableData,
  SiteSecondFactorEnrollment,
  SiteSecondFactorGetStatusData,
  SiteSecondFactorRegenerateRecoveryCodesData,
  SiteSecondFactorStartEnrollmentData,
  SiteSecondFactorStatus,
} from '../../generated/site/types.gen.ts'
import { securityRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { SecurityPage } from './security.tsx'

type Op = 'start' | 'confirm' | 'regenerate' | 'disable'
type Call = { op: Op; body: unknown }
type Handler = (req: Request) => Response | Promise<Response>

const enrollment: SiteSecondFactorEnrollment = {
  secret: 'JBSWY3DPEHPK3PXP',
  otpauthUri: 'otpauth://totp/1mail:info@1mail.com?secret=JBSWY3DPEHPK3PXP&issuer=1mail',
  qrCode:
    'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
}
const codes = ['aaaaa-bbbbb', 'ccccc-ddddd'] as const

// Serves the Second factor endpoints with a status the mutations move, and records
// every mutating call.
function serveSecurity(
  calls: Call[],
  initial: SiteSecondFactorStatus,
  overrides: Partial<Record<Op, Handler>> = {},
) {
  let status = initial
  const serve =
    (op: Op, fallback: Handler): Handler =>
    async (req) => {
      const text = await req.clone().text()
      calls.push({ op, body: text ? JSON.parse(text) : null })
      return (overrides[op] ?? fallback)(req)
    }
  mockClientRoutes([
    route<SiteSecondFactorGetStatusData>('GET', '/me/second-factor', {}, () =>
      jsonResponse(status),
    ),
    route<SiteSecondFactorStartEnrollmentData>(
      'POST',
      '/me/second-factor/enrollment',
      {},
      serve('start', () => {
        status = { enabled: false, pending: true, recoveryCodesRemaining: 0 }
        return jsonResponse(enrollment)
      }),
    ),
    route<SiteSecondFactorConfirmEnrollmentData>(
      'POST',
      '/me/second-factor/enrollment/confirm',
      {},
      serve('confirm', () => {
        status = { enabled: true, pending: false, recoveryCodesRemaining: codes.length }
        return jsonResponse({ codes })
      }),
    ),
    route<SiteSecondFactorRegenerateRecoveryCodesData>(
      'POST',
      '/me/second-factor/recovery-codes',
      {},
      serve('regenerate', () => jsonResponse({ codes })),
    ),
    route<SiteSecondFactorDisableData>(
      'POST',
      '/me/second-factor/disable',
      {},
      serve('disable', () => {
        status = { enabled: false, pending: false, recoveryCodesRemaining: 0 }
        return new Response(null, { status: 204 })
      }),
    ),
  ])
}

const off: SiteSecondFactorStatus = { enabled: false, pending: false, recoveryCodesRemaining: 0 }
const on: SiteSecondFactorStatus = { enabled: true, pending: false, recoveryCodesRemaining: 3 }

const renderPage = () => renderWithRouter(<SecurityPage />, routeMount(securityRoute))

test('enrolls an authenticator app and shows the recovery codes once', async () => {
  const calls: Call[] = []
  serveSecurity(calls, off)
  const { screen } = await renderPage()

  await expect.element(screen.getByText('Off', { exact: true })).toBeInTheDocument()
  await screen.getByRole('button', { name: 'Set up authenticator app' }).click()

  await expect
    .element(screen.getByRole('img', { name: 'QR code for the authenticator app' }))
    .toBeInTheDocument()
  await expect.element(screen.getByText(enrollment.secret)).toBeInTheDocument()

  await screen.getByLabelText(/^Code from the app/).fill(' 123456 ')
  await screen.getByRole('button', { name: 'Turn on' }).click()

  await expect.element(screen.getByText('Save your recovery codes')).toBeInTheDocument()
  for (const code of codes) {
    await expect.element(screen.getByText(code)).toBeInTheDocument()
  }
  expect(calls).toEqual([
    { op: 'start', body: null },
    { op: 'confirm', body: { code: '123456' } },
  ])

  await screen.getByRole('button', { name: 'I have saved them' }).click()
  expect(screen.getByText(codes[0]).elements()).toHaveLength(0)
  await expect.element(screen.getByText('On', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('2 recovery codes left')).toBeInTheDocument()
})

test('reports a wrong confirmation code and keeps the setup open', async () => {
  serveSecurity([], off, {
    confirm: () => jsonResponse({ status: 422, detail: 'the code is not valid' }, { status: 422 }),
  })
  const { screen } = await renderPage()

  await screen.getByRole('button', { name: 'Set up authenticator app' }).click()
  await screen.getByLabelText(/^Code from the app/).fill('000000')
  await screen.getByRole('button', { name: 'Turn on' }).click()

  await expect
    .element(screen.getByText('Could not turn on two-factor authentication'))
    .toBeInTheDocument()
  await expect.element(screen.getByText(enrollment.secret)).toBeInTheDocument()
  expect(screen.getByText('Save your recovery codes').elements()).toHaveLength(0)
})

test('regenerates the recovery codes with the password', async () => {
  const calls: Call[] = []
  serveSecurity(calls, on)
  const { screen } = await renderPage()

  await expect.element(screen.getByText('3 recovery codes left')).toBeInTheDocument()
  await screen
    .getByLabelText(/^Current password/)
    .first()
    .fill('pw-123456')
  await screen.getByRole('button', { name: 'Generate new codes' }).click()

  await expect.element(screen.getByText(codes[1])).toBeInTheDocument()
  expect(calls).toEqual([{ op: 'regenerate', body: { currentPassword: 'pw-123456' } }])
})

test('disables the second factor with the password and a code', async () => {
  const calls: Call[] = []
  serveSecurity(calls, on)
  const { screen } = await renderPage()

  await screen
    .getByLabelText(/^Current password/)
    .last()
    .fill('pw-123456')
  await screen.getByLabelText(/^Code from the app or a recovery code/).fill('654321')
  await screen.getByRole('button', { name: 'Turn off two-factor authentication' }).click()

  await expect.element(screen.getByText('Two-factor authentication is off')).toBeInTheDocument()
  expect(calls).toEqual([{ op: 'disable', body: { currentPassword: 'pw-123456', code: '654321' } }])
  await expect
    .element(screen.getByRole('button', { name: 'Set up authenticator app' }))
    .toBeInTheDocument()
})
