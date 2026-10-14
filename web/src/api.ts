export type Server = {
  address: string
  key: string
  client_config: string
  id_running: boolean
  relay_running: boolean
}

export type Device = {
  id: string
  hostname: string
  os: string
  username: string
  version: string
  owner: string
  last_seen: string
}

export type DeviceLogin = {
  device_id: string
  device_name: string
  device_os: string
  user: string
}

async function parse<T>(response: Response): Promise<T> {
  if (response.status === 401) {
    const here = window.location.pathname + window.location.search
    window.location.href = `/auth/login?next=${encodeURIComponent(here)}`
    throw new Error('unauthenticated')
  }
  if (!response.ok) {
    let message = `${response.status}`
    try {
      const body = await response.json()
      if (body?.error) message = body.error
    } catch {
      message = `${response.status}`
    }
    throw new Error(message)
  }
  return response.json() as Promise<T>
}

export const api = {
  server: () => fetch('/api/server').then(parse<Server>),
  devices: () => fetch('/api/devices').then(parse<Device[]>),
  deviceLogin: (code: string) =>
    fetch(`/api/device-login?code=${encodeURIComponent(code)}`).then(parse<DeviceLogin>),
  approveDeviceLogin: (code: string) =>
    fetch('/api/device-login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    }).then(parse<Record<string, never>>),
}
