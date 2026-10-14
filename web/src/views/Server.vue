<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { api, type Device, type Server } from '@/api'
import { notify } from '@/notify'

const server = ref<Server | null>(null)
const devices = ref<Device[]>([])
let timer: ReturnType<typeof setInterval> | undefined

async function load() {
  try {
    server.value = await api.server()
    devices.value = await api.devices()
  } catch (e) {
    notify.error((e as Error).message)
  }
}

async function copy(value: string, what: string) {
  try {
    await navigator.clipboard.writeText(value)
    notify.success(`${what} copied`)
  } catch (e) {
    notify.error((e as Error).message)
  }
}

function seen(device: Device): string {
  return new Date(device.last_seen).toLocaleString()
}

onMounted(() => {
  load()
  timer = setInterval(load, 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="sc-card" data-testid="server-card">
    <h1 class="sc-title">RustDesk Server</h1>
    <p class="sc-lead">
      Your own ID and relay server. Point RustDesk clients at it and remote sessions stop going
      through the public RustDesk servers.
    </p>

    <template v-if="server">
      <div class="sc-row">
        <span class="sc-row-label">ID server</span>
        <span class="sc-row-value" data-testid="id-status">
          <el-tag :type="server.id_running ? 'success' : 'danger'" size="small">
            {{ server.id_running ? 'Running' : 'Stopped' }}
          </el-tag>
        </span>
      </div>
      <div class="sc-row">
        <span class="sc-row-label">Relay server</span>
        <span class="sc-row-value" data-testid="relay-status">
          <el-tag :type="server.relay_running ? 'success' : 'danger'" size="small">
            {{ server.relay_running ? 'Running' : 'Stopped' }}
          </el-tag>
        </span>
      </div>
    </template>
  </div>

  <div v-if="server" class="sc-card">
    <h2 class="sc-section-title">Connect a client</h2>
    <p class="sc-lead">
      In the RustDesk client open Settings, Network, ID/Relay server. Either paste the server
      config from the clipboard there, or fill in the ID server, the API server and the key by
      hand and leave the relay server empty.
    </p>
    <p class="sc-lead" data-testid="websocket-note">
      Then tick Use WebSocket on the same page. Clients reach this server over HTTPS only, so
      there are no ports to forward, and every session is relayed through this device.
    </p>

    <el-alert
      v-if="!server.key"
      type="warning"
      :closable="false"
      data-testid="key-missing"
      title="The server has not generated its key yet."
    />

    <template v-else>
      <div class="sc-row">
        <span class="sc-row-label">ID server</span>
        <span class="sc-row-value copyable">
          <span data-testid="server-address">{{ server.address }}</span>
          <el-button size="small" data-testid="copy-address" @click="copy(server.address, 'ID server')">
            Copy
          </el-button>
        </span>
      </div>
      <div class="sc-row">
        <span class="sc-row-label">API server</span>
        <span class="sc-row-value copyable">
          <span data-testid="api-address">https://{{ server.address }}</span>
          <el-button
            size="small"
            data-testid="copy-api"
            @click="copy(`https://${server.address}`, 'API server')"
          >
            Copy
          </el-button>
        </span>
      </div>
      <div class="sc-row">
        <span class="sc-row-label">Key</span>
        <span class="sc-row-value copyable">
          <span data-testid="server-key" class="mono">{{ server.key }}</span>
          <el-button size="small" data-testid="copy-key" @click="copy(server.key, 'Key')">
            Copy
          </el-button>
        </span>
      </div>
      <div class="sc-actions config-actions">
        <el-button
          type="primary"
          data-testid="copy-config"
          @click="copy(server.client_config, 'Server config')"
        >
          Copy server config
        </el-button>
      </div>
    </template>
  </div>

  <div v-if="server" class="sc-card" data-testid="devices-card">
    <h2 class="sc-section-title">Devices</h2>
    <p class="sc-lead" data-testid="account-note">
      Clients pointed at this server show up here. To keep one address book on all of them, open
      the account login inside the RustDesk client and choose Continue with Syncloud.
    </p>
    <p v-if="devices.length === 0" class="sc-lead" data-testid="devices-empty">
      No client has connected yet.
    </p>
    <div v-for="device in devices" :key="device.id" class="sc-row" data-testid="device">
      <span class="sc-row-label">
        <span data-testid="device-name">{{ device.hostname || device.id }}</span>
        <span class="device-id mono" data-testid="device-id">{{ device.id }}</span>
      </span>
      <span class="sc-row-value device-details">
        <el-tag v-if="device.owner" size="small" data-testid="device-owner">{{ device.owner }}</el-tag>
        <span data-testid="device-os">{{ device.os }}</span>
        <span class="device-seen" data-testid="device-seen">{{ seen(device) }}</span>
      </span>
    </div>
  </div>
</template>

<style scoped>
.copyable {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  min-width: 0;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
  word-break: break-all;
}

.device-id {
  margin-left: 10px;
  color: var(--sc-muted);
}

.device-details {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 10px;
  min-width: 0;
}

.device-seen {
  color: var(--sc-muted);
  font-size: 13px;
}

.config-actions {
  margin-top: 18px;
}
</style>
