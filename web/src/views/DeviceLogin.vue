<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api, type DeviceLogin } from '@/api'

const route = useRoute()
const code = String(route.query.code ?? '')
const login = ref<DeviceLogin | null>(null)
const error = ref('')
const approved = ref(false)
const working = ref(false)

async function load() {
  try {
    login.value = await api.deviceLogin(code)
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function approve() {
  working.value = true
  try {
    await api.approveDeviceLogin(code)
    approved.value = true
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    working.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="sc-card" data-testid="device-login-card">
    <h1 class="sc-title">Sign in a RustDesk client</h1>

    <el-alert
      v-if="error"
      type="error"
      :closable="false"
      data-testid="device-login-error"
      :title="error"
    />

    <template v-else-if="approved">
      <el-alert
        type="success"
        :closable="false"
        data-testid="device-login-approved"
        title="Signed in. You can close this page and go back to RustDesk."
      />
    </template>

    <template v-else-if="login">
      <p class="sc-lead">
        A RustDesk client is asking to sign in as
        <strong data-testid="device-login-user">{{ login.user }}</strong>. It will be able to read
        and change this account's address book. Only continue if you started this from that client
        yourself.
      </p>
      <div class="sc-row">
        <span class="sc-row-label">Device</span>
        <span class="sc-row-value" data-testid="device-login-name">{{ login.device_name }}</span>
      </div>
      <div class="sc-row">
        <span class="sc-row-label">RustDesk ID</span>
        <span class="sc-row-value" data-testid="device-login-id">{{ login.device_id }}</span>
      </div>
      <div class="sc-row">
        <span class="sc-row-label">System</span>
        <span class="sc-row-value" data-testid="device-login-os">{{ login.device_os }}</span>
      </div>
      <div class="sc-actions approve-actions">
        <el-button
          type="primary"
          :loading="working"
          data-testid="device-login-approve"
          @click="approve"
        >
          Sign in this client
        </el-button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.approve-actions {
  margin-top: 18px;
}
</style>
