import { test, expect } from '../fixtures'
import { shoot } from '../helpers/screenshot'
import { saveKey, savedToken, workstationId } from '../helpers/state'
import { openApp } from '../helpers/syncloud'

test('remember the key clients are configured with before the upgrade', async ({ page }) => {
  await openApp(page)

  const key = await page.getByTestId('server-key').innerText()
  expect(key).toMatch(/^[A-Za-z0-9+/]{43}=$/)
  saveKey(key)
  await shoot(page, 'server-before-upgrade')
})

test('the clients that connected are listed as devices', async ({ page }) => {
  await openApp(page)

  await expect(page.getByTestId('device')).toHaveCount(2, { timeout: 60_000 })
  await expect(page.getByTestId('devices-card')).toContainText(workstationId())
  await shoot(page, 'devices')
})

test('the signed-in client saved the workstation to the address book', async ({ request }) => {
  const book = await request.get('/api/ab', {
    headers: { Authorization: `Bearer ${savedToken()}` },
  })

  const peers = JSON.parse((await book.json()).data).peers.map((peer: { id: string }) => peer.id)
  expect(peers).toContain(workstationId())
})
