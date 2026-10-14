import { test, expect } from '../fixtures'
import { shoot } from '../helpers/screenshot'
import { savedKey, savedToken, workstationId } from '../helpers/state'
import { openApp } from '../helpers/syncloud'

test('the key clients are configured with survives the upgrade', async ({ page }) => {
  await openApp(page)

  await expect(page.getByTestId('server-key')).toHaveText(savedKey())
  await expect(page.getByTestId('id-status')).toHaveText('Running', { timeout: 30_000 })
  await expect(page.getByTestId('relay-status')).toHaveText('Running', { timeout: 30_000 })
  await shoot(page, 'server-after-upgrade')
})

test('devices, the client token and the address book survive the upgrade', async ({
  page,
  request,
}) => {
  await openApp(page)
  await expect(page.getByTestId('devices-card')).toContainText(workstationId())
  await shoot(page, 'devices-after-upgrade')

  const book = await request.get('/api/ab', {
    headers: { Authorization: `Bearer ${savedToken()}` },
  })

  expect(book.status()).toBe(200)
  const peers = JSON.parse((await book.json()).data).peers.map((peer: { id: string }) => peer.id)
  expect(peers).toContain(workstationId())
})
