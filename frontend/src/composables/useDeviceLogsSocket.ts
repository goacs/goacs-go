import { onMounted, onUnmounted } from 'vue'
import { getSocket, getSocketToken } from '@/sockets/socket'
import type { LogEntry } from '@/api/types/log'

// Joins the "device.<uuid>" room (see http/socket.go) and invokes onLog for
// every "device.logged" event broadcast to it - the live-tail counterpart to
// DeviceLogsPanel's initial REST fetch.
export function useDeviceLogsSocket(uuid: string, onLog: (entry: LogEntry) => void) {
  function handleLog(entry: LogEntry) {
    if (entry.cpe_uuid === uuid) onLog(entry)
  }

  onMounted(() => {
    const socket = getSocket()
    if (!socket) return
    socket.emit('join-device', { token: getSocketToken(), uuid })
    socket.on('device.logged', handleLog)
  })

  onUnmounted(() => {
    const socket = getSocket()
    if (!socket) return
    socket.emit('leave-device', { token: getSocketToken(), uuid })
    socket.off('device.logged', handleLog)
  })
}
