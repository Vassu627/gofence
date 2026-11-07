import { useEffect, useState } from 'react'
import { websocketURL } from '../api'
import type { AlertEvent } from '../types'

export type Connection = 'connecting' | 'live' | 'offline'

export function useLiveAlerts() {
  const [events, setEvents] = useState<AlertEvent[]>([])
  const [toasts, setToasts] = useState<AlertEvent[]>([])
  const [status, setStatus] = useState<Connection>('connecting')

  useEffect(() => {
    let stopped = false
    let socket: WebSocket | null = null
    let attempt = 0
    let retry = 0
    const timeouts = new Set<number>()

    const connect = () => {
      if (stopped) return
      setStatus('connecting')
      socket = new WebSocket(websocketURL())
      socket.onopen = () => {
        attempt = 0
        if (!stopped) setStatus('live')
      }
      socket.onmessage = (message) => {
        if (stopped) return
        try {
          const event = JSON.parse(String(message.data)) as AlertEvent
          if (!event?.event_id) return
          setEvents((prev) => [event, ...prev].slice(0, 50))
          setToasts((prev) => [event, ...prev].slice(0, 3))
          const timeout = window.setTimeout(() => {
            setToasts((prev) => prev.filter((item) => item.event_id !== event.event_id))
            timeouts.delete(timeout)
          }, 8000)
          timeouts.add(timeout)
        } catch {
          // Ignore malformed socket payloads.
        }
      }
      socket.onclose = () => {
        if (stopped) return
        setStatus('offline')
        const delay = Math.min(10000, 500 * 2 ** attempt)
        attempt += 1
        retry = window.setTimeout(connect, delay)
      }
    }

    connect()
    return () => {
      stopped = true
      window.clearTimeout(retry)
      timeouts.forEach((timeout) => window.clearTimeout(timeout))
      socket?.close()
    }
  }, [])

  return {
    events,
    toasts,
    status,
    dismissToast: (id: string) => setToasts((prev) => prev.filter((item) => item.event_id !== id)),
    clear: () => setEvents([]),
  }
}
