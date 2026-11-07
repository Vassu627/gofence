const base = (process.env.API_URL || 'http://localhost:8080').replace(/\/$/, '')

function assertTime(body, label) {
  if (typeof body.time_ns !== 'string' || !/^\d+$/.test(body.time_ns)) {
    throw new Error(`${label} is missing a numeric time_ns: ${JSON.stringify(body)}`)
  }
}

async function call(method, path, body) {
  const response = await fetch(`${base}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await response.text()
  let data = {}
  try {
    data = text ? JSON.parse(text) : {}
  } catch {
    data = { raw: text }
  }
  if (!response.ok) {
    throw new Error(`${method} ${path} -> ${response.status} ${JSON.stringify(data)}`)
  }
  assertTime(data, `${method} ${path}`)
  return data
}

const suffix = Date.now().toString().slice(-8)
const fence = await call('POST', '/geofences', {
  name: `Smoke Zone ${suffix}`,
  description: 'Created by scripts/smoke.mjs',
  category: 'delivery_zone',
  coordinates: [
    [37.7749, -122.4194],
    [37.7849, -122.4194],
    [37.7849, -122.4094],
    [37.7749, -122.4094],
    [37.7749, -122.4194],
  ],
})
if (!fence.id || fence.status !== 'active') throw new Error('geofence create response is incomplete')

const vehicle = await call('POST', '/vehicles', {
  vehicle_number: `SM-${suffix}`,
  driver_name: 'Smoke Driver',
  vehicle_type: 'van',
  phone: '+15550001111',
})
await call('POST', '/alerts/configure', {
  geofence_id: fence.id,
  vehicle_id: vehicle.id,
  event_type: 'entry',
})

const outsideAt = new Date(Date.now() - 5000).toISOString()
const outside = await call('POST', '/vehicles/location', {
  vehicle_id: vehicle.id,
  latitude: 37.77,
  longitude: -122.4144,
  timestamp: outsideAt,
})
if (outside.location_updated !== true || outside.current_geofences.length !== 0) {
  throw new Error(`expected the first fix to be outside, got ${JSON.stringify(outside)}`)
}

const inside = await call('POST', '/vehicles/location', {
  vehicle_id: vehicle.id,
  latitude: 37.778,
  longitude: -122.418,
  timestamp: new Date().toISOString(),
})
if (inside.current_geofences.length !== 1 || inside.current_geofences[0].status !== 'inside') {
  throw new Error(`expected one containing geofence, got ${JSON.stringify(inside)}`)
}

const history = await call('GET', `/violations/history?vehicle_id=${encodeURIComponent(vehicle.id)}&limit=10`)
const entry = history.violations.find((row) => row.event_type === 'entry' && row.geofence_id === fence.id)
if (!entry) throw new Error(`entry was not stored: ${JSON.stringify(history)}`)

const current = await call('GET', `/vehicles/location/${encodeURIComponent(vehicle.id)}`)
if (!current.current_location || current.current_geofences.length !== 1) {
  throw new Error(`current location is wrong: ${JSON.stringify(current)}`)
}

await call('GET', '/geofences?category=delivery_zone')
await call('GET', '/vehicles')
await call('GET', `/alerts?vehicle_id=${encodeURIComponent(vehicle.id)}`)
await call('GET', '/health')

console.log(`smoke ok vehicle=${vehicle.id} geofence=${fence.id}`)
