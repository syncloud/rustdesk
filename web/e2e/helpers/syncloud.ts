import { expect, type Page } from '@playwright/test'

export type SyncloudCreds = {
  user: string
  password: string
}

export function credsFromEnv(): SyncloudCreds {
  const user = process.env.DEVICE_USER
  const password = process.env.DEVICE_PASSWORD
  if (!user || !password) {
    throw new Error('DEVICE_USER and DEVICE_PASSWORD must be set')
  }
  return { user, password }
}

export async function openApp(page: Page): Promise<void> {
  await open(page, '/', 'server-key')
}

export async function open(page: Page, path: string, readyTestId: string): Promise<void> {
  const creds = credsFromEnv()
  const username = page.locator('#username-textfield')
  const consent = page.locator('#accept-button')
  const app = page.getByTestId(readyTestId)

  await page.goto(path)
  await expect(username.or(app)).toBeVisible({ timeout: 60_000 })
  if (await app.isVisible()) return

  await username.fill(creds.user)
  await page.locator('#password-textfield').fill(creds.password)
  await page.locator('#sign-in-button').click()

  await expect(consent.or(app)).toBeVisible({ timeout: 60_000 })
  if (await consent.isVisible()) {
    await consent.click()
  }
  await expect(app).toBeVisible({ timeout: 60_000 })
}
