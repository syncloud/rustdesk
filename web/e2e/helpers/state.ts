import * as fs from 'node:fs'
import * as path from 'node:path'

const dir = path.join('e2e', '.state')

function save(name: string, value: string): void {
  fs.mkdirSync(dir, { recursive: true })
  fs.writeFileSync(path.join(dir, name), value)
}

export function saveKey(key: string): void {
  save('key.txt', key)
}

export function savedKey(): string {
  return fs.readFileSync(path.join(dir, 'key.txt'), 'utf8')
}

export function saveClientConfig(config: string): void {
  save('config.txt', config)
}

export function saveToken(token: string): void {
  save('token.txt', token)
}

export function savedToken(): string {
  return fs.readFileSync(path.join(dir, 'token.txt'), 'utf8')
}

export function workstationId(): string {
  return fs.readFileSync(path.join('..', 'artifact', 'client', 'workstation-id.txt'), 'utf8').trim()
}
