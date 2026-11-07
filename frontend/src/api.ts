import type {
  AlertRule,
  Category,
  EventType,
  Geofence,
  LocationUpdate,
  Vehicle,
  VehicleLocation,
  Violation,
} from './types'

const rawUrl = (import.meta.env.VITE_API_URL as string | undefined)?.trim() || 'http://localhost:8080'

export const API_URL = (/^https?:\/\//.test(rawUrl) ? rawUrl : `http://${rawUrl}`).replace(/\/$/, '')

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export function websocketURL() {
  const url = new URL(API_URL)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.pathname = '/ws/alerts'
  url.search = ''
  const key = (import.meta.env.VITE_API_KEY as string | undefined)?.trim()
  if (key) url.searchParams.set('api_key', key)
  return url.toString()
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  const key = (import.meta.env.VITE_API_KEY as string | undefined)?.trim()
  if (key) headers.set('X-API-Key', key)
  let response: Response
  try {
    response = await fetch(`${API_URL}${path}`, { ...init, headers })
  } catch {
    throw new ApiError(0, `Cannot reach the API at ${API_URL}. Start the backend and refresh.`)
  }
  const text = await response.text()
  let data: { error?: string } = {}
  if (text) {
    try {
      data = JSON.parse(text) as { error?: string }
    } catch {
      data = {}
    }
  }
  if (!response.ok) throw new ApiError(response.status, data.error || `Request failed (${response.status})`)
  return data as T
}

export async function listGeofences(category?: string) {
  const query = new URLSearchParams()
  if (category) query.set('category', category)
  const suffix = query.toString() ? `?${query}` : ''
  const data = await req<{ geofences: Geofence[] }>(`/geofences${suffix}`)
  return { ...data, geofences: data.geofences ?? [] }
}

export function createGeofence(body: {
  name: string
  description?: string
  coordinates: number[][]
  category: Category
}) {
  return req<{ id: string; name: string; status: string }>('/geofences', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export async function listVehicles() {
  const data = await req<{ vehicles: Vehicle[] }>('/vehicles')
  return { ...data, vehicles: data.vehicles ?? [] }
}

export function createVehicle(body: {
  vehicle_number: string
  driver_name: string
  vehicle_type: string
  phone: string
}) {
  return req<{ id: string; vehicle_number: string; status: string }>('/vehicles', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function updateLocation(body: {
  vehicle_id: string
  latitude: number
  longitude: number
  timestamp: string
}) {
  return req<LocationUpdate>('/vehicles/location', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getVehicleLocation(id: string) {
  return req<VehicleLocation>(`/vehicles/location/${encodeURIComponent(id)}`)
}

export async function listAlerts(filter?: { geofence_id?: string; vehicle_id?: string }) {
  const query = new URLSearchParams()
  if (filter?.geofence_id) query.set('geofence_id', filter.geofence_id)
  if (filter?.vehicle_id) query.set('vehicle_id', filter.vehicle_id)
  const suffix = query.toString() ? `?${query}` : ''
  const data = await req<{ alerts: AlertRule[] }>(`/alerts${suffix}`)
  return { ...data, alerts: data.alerts ?? [] }
}

export function createAlert(body: { geofence_id: string; vehicle_id?: string; event_type: EventType }) {
  return req<{ alert_id: string; status: string }>('/alerts/configure', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export async function listViolations(filter: {
  vehicle_id?: string
  geofence_id?: string
  start_date?: string
  end_date?: string
  limit?: number
  offset?: number
}) {
  const query = new URLSearchParams()
  Object.entries(filter).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value))
  })
  const suffix = query.toString() ? `?${query}` : ''
  const data = await req<{ violations: Violation[]; total_count: number }>(`/violations/history${suffix}`)
  return { ...data, violations: data.violations ?? [], total_count: data.total_count ?? 0 }
}
