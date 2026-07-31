// go-socket.io v1.4.x (backend/http/socket.go) only speaks Engine.IO v3, so
// this deliberately pins socket.io-client to v2.x - v3/v4 dropped EIO3
// support and cannot connect to it at all (fails with 403/502 handshake
// errors, not a graceful downgrade).
import io from 'socket.io-client'

type Socket = SocketIOClient.Socket

let socket: Socket | null = null
let socketToken: string | null = null

export function connectSocket(token: string): Socket {
  if (socket) {
    socket.disconnect()
  }

  socketToken = token
  socket = io(import.meta.env.VITE_SOCKET_URL, {
    transports: ['websocket'],
    autoConnect: true,
  })

  return socket
}

export function disconnectSocket(): void {
  socket?.disconnect()
  socket = null
  socketToken = null
}

export function getSocket(): Socket | null {
  return socket
}

// Stashed for join-device/leave-device payloads (see useDeviceLogsSocket) -
// go-socket.io has no built-in per-channel auth, so the server validates
// this token itself on each join attempt.
export function getSocketToken(): string | null {
  return socketToken
}
