import { test, expect } from '../fixtures'
import { shoot } from '../helpers/screenshot'
import { saveClientConfig, saveToken } from '../helpers/state'
import { credsFromEnv, open, openApp } from '../helpers/syncloud'

const domain = `rustdesk.${process.env.PLAYWRIGHT_DOMAIN ?? 'bookworm.com'}`

test('signs in through Syncloud and shows the running server with its key', async ({ page }) => {
  await openApp(page)

  await expect(page.getByTestId('server-key')).toHaveText(/^[A-Za-z0-9+/]{43}=$/)
  await expect(page.getByTestId('server-address')).toHaveText(domain)
  await expect(page.getByTestId('id-status')).toHaveText('Running', { timeout: 30_000 })
  await expect(page.getByTestId('relay-status')).toHaveText('Running', { timeout: 30_000 })
  await expect(page.getByTestId('api-address')).toHaveText(`https://${domain}`)
  await expect(page.getByTestId('websocket-note')).toBeVisible()
  await shoot(page, 'server')
})

test('the copied server config decodes to the address and key on screen', async ({ page }) => {
  await openApp(page)
  const key = await page.getByTestId('server-key').innerText()

  await page.getByTestId('copy-config').click()
  const copied = await page.evaluate(() => navigator.clipboard.readText())
  const decoded = JSON.parse(
    Buffer.from(copied.split('').reverse().join(''), 'base64url').toString('utf8'),
  )

  expect(decoded).toEqual({ host: domain, relay: '', api: `https://${domain}`, key })
  saveClientConfig(copied)
  await shoot(page, 'server-config-copied')
})

test('a client signs in through Syncloud and gets a token for the address book', async ({
  page,
  request,
}) => {
  const device = { id: '100200300', uuid: 'ZTJlLWRldmljZQ==' }
  const options = await request.get('/api/login-options')
  expect(await options.json()).toEqual(['oidc/syncloud'])

  const started = await request.post('/api/oidc/auth', {
    data: { op: 'syncloud', ...device, deviceInfo: { os: 'linux', type: 'client', name: 'e2e-laptop' } },
  })
  const { code, url } = await started.json()
  const query = { params: { code, ...device } }
  expect((await (await request.get('/api/oidc/auth-query', query)).json()).error).toBe(
    'No authed oidc is found',
  )

  const approval = new URL(url)
  expect(approval.origin).toBe(`https://${domain}`)
  await open(page, approval.pathname + approval.search, 'device-login-approve')
  await expect(page.getByTestId('device-login-id')).toHaveText(device.id)
  await expect(page.getByTestId('device-login-name')).toHaveText('e2e-laptop')
  await expect(page.getByTestId('device-login-user')).toHaveText(credsFromEnv().user)
  await shoot(page, 'device-login')

  await page.getByTestId('device-login-approve').click()
  await expect(page.getByTestId('device-login-approved')).toBeVisible()
  await shoot(page, 'device-login-approved')

  const signedIn = await (await request.get('/api/oidc/auth-query', query)).json()
  expect(signedIn.type).toBe('access_token')
  expect(signedIn.user.name).toBe(credsFromEnv().user)
  saveToken(signedIn.access_token)

  const headers = { Authorization: `Bearer ${signedIn.access_token}` }
  const current = await request.post('/api/currentUser', { headers, data: device })
  expect((await current.json()).name).toBe(credsFromEnv().user)
  expect((await request.post('/api/ab/personal', { headers })).status()).toBe(404)
  expect(await (await request.get('/api/ab', { headers })).json()).toEqual({ data: '{}' })
})

test('the address book is closed without a token', async ({ request }) => {
  expect((await request.get('/api/ab')).status()).toBe(401)
  expect((await request.get('/api/ab', { headers: { Authorization: 'Bearer forged' } })).status()).toBe(401)
  expect((await request.post('/api/currentUser', { data: {} })).status()).toBe(401)
})

test('an expired client sign-in is refused', async ({ page }) => {
  await open(page, '/device?code=expired', 'device-login-error')

  await expect(page.getByTestId('device-login-approve')).toHaveCount(0)
  await shoot(page, 'device-login-expired')
})

test('logout ends the session', async ({ page }) => {
  await openApp(page)

  await page.getByTestId('logout').click()

  await expect(page.locator('#username-textfield')).toBeVisible({ timeout: 60_000 })
  await shoot(page, 'logged-out')
})
