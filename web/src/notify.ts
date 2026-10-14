import { ElMessage } from 'element-plus'

const options = { showClose: true, duration: 3500, offset: 74 }

export const notify = {
  success: (message: string) => ElMessage({ type: 'success', message, ...options }),
  error: (message: string) => ElMessage({ type: 'error', message, ...options }),
  info: (message: string) => ElMessage({ type: 'info', message, ...options }),
}
