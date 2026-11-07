import { useState, type FormEvent } from 'react'
import { ApiError, createAlert, createGeofence, createVehicle, updateLocation } from '../api'
import { CATEGORIES } from '../categories'
import { closeRing } from '../coords'
import { useData } from '../context'
import { formatCoord, formatWhen } from '../format'
import { sampleDelivery, samplePath, sampleRestricted, sampleVehicle, wait } from '../sample'
import type { Category } from '../types'
import MapBoard, { type MapMode, type VehiclePin } from './MapBoard'

type Props = {
  mode: MapMode
  setMode: (mode: MapMode) => void
  activeVehicle: string
  setActiveVehicle: (id: string) => void
}

export default function Operations({ mode, setMode, activeVehicle, setActiveVehicle }: Props) {
  const { geofences, vehicles, fixes, loading, refresh, live } = useData()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [category, setCategory] = useState<Category>('delivery_zone')
  const [arm, setArm] = useState(false)
  const [draft, setDraft] = useState<[number, number][]>([])
  const [draftClosed, setDraftClosed] = useState(false)
  const [lat, setLat] = useState('')
  const [lng, setLng] = useState('')
  const [stamp, setStamp] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [note, setNote] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [demoBusy, setDemoBusy] = useState(false)
  const [demoLog, setDemoLog] = useState<string[]>([])

  const pins: VehiclePin[] = vehicles.flatMap((vehicle) => {
    const fix = fixes[vehicle.id]
    if (!fix?.current_location) return []
    const names = fix.current_geofences.map((fence) => fence.geofence_name)
    return [{
      id: vehicle.id,
      label: vehicle.vehicle_number,
      driver: vehicle.driver_name,
      lat: fix.current_location.latitude,
      lng: fix.current_location.longitude,
      restricted: fix.current_geofences.some((fence) => fence.category === 'restricted_zone'),
      detail: names.length ? names.join(', ') : 'Outside all geofences',
    }]
  })

  const cursor = lat && lng && Number.isFinite(Number(lat)) && Number.isFinite(Number(lng))
    ? [Number(lat), Number(lng)] as [number, number]
    : null

  function onMapClick(nextLat: number, nextLng: number) {
    if (mode === 'draw') {
      if (draftClosed) return
      setDraft((points) => [...points, [nextLat, nextLng]])
      setError(null)
      return
    }
    if (mode === 'place') {
      setLat(nextLat.toFixed(6))
      setLng(nextLng.toFixed(6))
      setError(null)
    }
  }

  function chooseCategory(next: Category) {
    setCategory(next)
    if (next === 'restricted_zone') setArm(true)
  }

  async function saveFence(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setNote(null)
    if (!name.trim()) {
      setError('Name the geofence.')
      return
    }
    if (!draftClosed || draft.length < 4) {
      setError('Add at least three corners, then close the shape.')
      return
    }
    setBusy(true)
    try {
      const created = await createGeofence({
        name: name.trim(),
        description: description.trim(),
        category,
        coordinates: closeRing(draft),
      })
      if (arm) await createAlert({ geofence_id: created.id, event_type: 'both' })
      await refresh()
      setName('')
      setDescription('')
      setDraft([])
      setDraftClosed(false)
      setArm(false)
      setNote(arm ? 'Geofence saved and an alert was armed for every vehicle.' : 'Geofence saved.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save the geofence.')
    } finally {
      setBusy(false)
    }
  }

  async function saveLocation(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setNote(null)
    if (!activeVehicle) {
      setError('Register a vehicle first.')
      return
    }
    const latitude = Number(lat)
    const longitude = Number(lng)
    if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
      setError('Enter a latitude and longitude, or click the map.')
      return
    }
    const when = stamp ? new Date(stamp) : new Date()
    if (Number.isNaN(when.getTime())) {
      setError('Timestamp is invalid.')
      return
    }
    setBusy(true)
    try {
      const result = await updateLocation({
        vehicle_id: activeVehicle,
        latitude,
        longitude,
        timestamp: when.toISOString(),
      })
      const names = result.current_geofences.map((fence) => fence.geofence_name)
      setNote(names.length ? `Inside ${names.join(', ')}.` : 'Location saved. The vehicle is outside every geofence.')
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not update the location.')
    } finally {
      setBusy(false)
    }
  }

  function useBrowserLocation() {
    if (!navigator.geolocation) {
      setError('This browser does not provide a location.')
      return
    }
    navigator.geolocation.getCurrentPosition(
      (position) => {
        setLat(position.coords.latitude.toFixed(6))
        setLng(position.coords.longitude.toFixed(6))
        setError(null)
      },
      () => setError('Location permission was denied.'),
      { enableHighAccuracy: true, timeout: 8000 },
    )
  }

  async function replay() {
    setDemoBusy(true)
    setDemoLog([])
    setError(null)
    const lines: string[] = []
    const push = (line: string) => {
      lines.push(line)
      setDemoLog([...lines])
    }
    try {
      if (!geofences.some((fence) => fence.name === sampleDelivery.name)) {
        const created = await createGeofence(sampleDelivery)
        await createAlert({ geofence_id: created.id, event_type: 'entry' })
        push('Created the delivery zone and an entry alert.')
      } else {
        push('Using the existing delivery zone.')
      }
      if (!geofences.some((fence) => fence.name === sampleRestricted.name)) {
        const created = await createGeofence(sampleRestricted)
        await createAlert({ geofence_id: created.id, event_type: 'both' })
        push('Created the restricted yard and armed entry and exit alerts.')
      } else {
        push('Using the existing restricted yard.')
      }
      try {
        await createVehicle(sampleVehicle)
        push('Registered KA-01-AB-1234.')
      } catch (err) {
        if (!(err instanceof ApiError) || err.status !== 409) throw err
        push('Using the existing sample vehicle.')
      }

      const snapshot = await refresh()
      const delivery = snapshot.geofences.find((fence) => fence.name === sampleDelivery.name)
      const restricted = snapshot.geofences.find((fence) => fence.name === sampleRestricted.name)
      const vehicle = snapshot.vehicles.find((item) => item.vehicle_number === sampleVehicle.vehicle_number)
      if (!delivery || !restricted || !vehicle) throw new Error('Sample data did not save.')

      const deliveryArmed = snapshot.alertRules.some((rule) =>
        rule.geofence_id === delivery.id && !rule.vehicle_id && (rule.event_type === 'entry' || rule.event_type === 'both'))
      if (!deliveryArmed) {
        await createAlert({ geofence_id: delivery.id, event_type: 'entry' })
        push('Armed an entry alert on the delivery zone.')
      }
      const restrictedArmed = snapshot.alertRules.some((rule) =>
        rule.geofence_id === restricted.id && !rule.vehicle_id && (rule.event_type === 'both' || rule.event_type === 'entry'))
      if (!restrictedArmed) {
        await createAlert({ geofence_id: restricted.id, event_type: 'both' })
        push('Armed entry and exit alerts on the restricted yard.')
      }

      let base = Date.now()
      const previous = snapshot.fixes[vehicle.id]?.current_location?.timestamp
      if (previous) {
        const parsed = new Date(previous).getTime()
        if (!Number.isNaN(parsed) && parsed >= base) base = parsed + 1000
      }
      for (let index = 0; index < samplePath.length; index += 1) {
        const step = samplePath[index]
        await updateLocation({
          vehicle_id: vehicle.id,
          latitude: step.latitude,
          longitude: step.longitude,
          timestamp: new Date(base + index * 1000).toISOString(),
        })
        push(step.label)
        await wait(700)
      }
      setActiveVehicle(vehicle.id)
      await refresh()
      push('Route complete. Entry alerts should be in the live feed.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'The sample route failed.')
    } finally {
      setDemoBusy(false)
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col lg:flex-row">
      <section className="order-2 max-h-[52vh] overflow-auto border-line p-4 lg:order-1 lg:max-h-none lg:w-[400px] lg:border-r">
        <div className="mb-4 flex rounded-md border border-line p-1">
          {(['view', 'draw', 'place'] as const).map((id) => (
            <button
              key={id}
              type="button"
              onClick={() => setMode(id)}
              className={`flex-1 rounded-md px-3 py-1.5 text-sm capitalize ${mode === id ? 'bg-amber font-semibold text-ink' : 'text-mist hover:text-paper'}`}
            >
              {id}
            </button>
          ))}
        </div>
        {loading && <p className="mb-3 text-sm text-mist">Loading yard data…</p>}
        {error && <p className="mb-3 text-sm text-danger">{error}</p>}
        {note && <p className="mb-3 text-sm text-signal">{note}</p>}

        {mode === 'view' && (
          <div className="space-y-3 text-sm">
            <p className="text-mist">Choose Draw to trace a boundary, or Place to drop a vehicle on the map.</p>
            <ul className="divide-y divide-line rounded-md border border-line">
              {vehicles.length === 0 && <li className="px-3 py-3 text-mist">No vehicles yet.</li>}
              {vehicles.map((vehicle) => {
                const fix = fixes[vehicle.id]
                return (
                  <li key={vehicle.id} className="px-3 py-2">
                    <div className="font-medium">{vehicle.vehicle_number}</div>
                    <div className="text-mist">{vehicle.driver_name}</div>
                    <div className="font-mono text-xs text-mist">
                      {fix?.current_location
                        ? `${formatCoord(fix.current_location.latitude)}, ${formatCoord(fix.current_location.longitude)}`
                        : 'No location yet'}
                    </div>
                  </li>
                )
              })}
            </ul>
          </div>
        )}

        {mode === 'draw' && (
          <form className="space-y-3" onSubmit={saveFence}>
            <p className="text-sm text-mist">Click the map to drop corners. Three corners minimum, then close the shape.</p>
            <label className="block">
              <span className="label">Name</span>
              <input className="input" value={name} onChange={(event) => setName(event.target.value)} />
            </label>
            <label className="block">
              <span className="label">Description</span>
              <textarea className="input min-h-20" value={description} onChange={(event) => setDescription(event.target.value)} />
            </label>
            <label className="block">
              <span className="label">Category</span>
              <select className="input" value={category} onChange={(event) => chooseCategory(event.target.value as Category)}>
                {CATEGORIES.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
              </select>
            </label>
            <label className="flex items-start gap-2 text-sm">
              <input type="checkbox" className="mt-1" checked={arm} onChange={(event) => setArm(event.target.checked)} />
              <span>Create an entry and exit alert for every vehicle. Leave this on for restricted zones.</span>
            </label>
            <p className="font-mono text-xs text-mist">{draft.length} point{draft.length === 1 ? '' : 's'}{draftClosed ? ' · closed' : ''}</p>
            <div className="flex flex-wrap gap-2">
              <button type="button" className="btn-ghost" disabled={draft.length < 3 || draftClosed} onClick={() => { setDraft((points) => closeRing(points)); setDraftClosed(true) }}>Close shape</button>
              <button type="button" className="btn-ghost" disabled={draft.length === 0} onClick={() => { setDraftClosed(false); setDraft((points) => points.slice(0, -1)) }}>Undo</button>
              <button type="button" className="btn-ghost" disabled={draft.length === 0} onClick={() => { setDraft([]); setDraftClosed(false) }}>Clear</button>
            </div>
            <button className="btn w-full" disabled={busy} type="submit">{busy ? 'Saving…' : 'Save geofence'}</button>
          </form>
        )}

        {mode === 'place' && (
          <form className="space-y-3" onSubmit={saveLocation}>
            <p className="text-sm text-mist">Click the map to set the position, or type coordinates.</p>
            <label className="block">
              <span className="label">Vehicle</span>
              <select className="input" value={activeVehicle} onChange={(event) => setActiveVehicle(event.target.value)}>
                <option value="">Select a vehicle</option>
                {vehicles.map((vehicle) => (
                  <option key={vehicle.id} value={vehicle.id}>{vehicle.vehicle_number} · {vehicle.driver_name}</option>
                ))}
              </select>
            </label>
            <div className="grid grid-cols-2 gap-2">
              <label className="block">
                <span className="label">Latitude</span>
                <input className="input font-mono" value={lat} onChange={(event) => setLat(event.target.value)} inputMode="decimal" />
              </label>
              <label className="block">
                <span className="label">Longitude</span>
                <input className="input font-mono" value={lng} onChange={(event) => setLng(event.target.value)} inputMode="decimal" />
              </label>
            </div>
            <label className="block">
              <span className="label">Timestamp</span>
              <input className="input" type="datetime-local" step="1" value={stamp} onChange={(event) => setStamp(event.target.value)} />
            </label>
            <p className="text-xs text-mist">Leave the timestamp blank to use the current time.</p>
            <div className="flex gap-2">
              <button type="button" className="btn-ghost" onClick={useBrowserLocation}>Use my location</button>
              <button className="btn flex-1" disabled={busy} type="submit">{busy ? 'Saving…' : 'Update location'}</button>
            </div>
          </form>
        )}

        <div className="mt-6 border-t border-line pt-4">
          <button type="button" className="btn w-full" disabled={demoBusy} onClick={replay}>
            {demoBusy ? 'Replaying route…' : 'Replay sample route'}
          </button>
          <p className="mt-2 text-xs text-mist">Creates a downtown delivery zone, a restricted yard, a truck, and drives it across both boundaries.</p>
          {demoLog.length > 0 && (
            <ol className="mt-3 space-y-1 text-sm text-mist">
              {demoLog.map((line, index) => <li key={`${index}-${line}`}>{line}</li>)}
            </ol>
          )}
        </div>

        <div className="mt-6">
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-sm font-semibold">Live feed</h2>
            {live.events.length > 0 && <button type="button" className="text-xs text-mist hover:text-paper" onClick={live.clear}>Clear</button>}
          </div>
          <ul className="space-y-2">
            {live.events.length === 0 && <li className="text-sm text-mist">Alerts appear here as soon as a rule matches.</li>}
            {live.events.slice(0, 6).map((event) => (
              <li key={event.event_id} className="rounded-md border border-line px-3 py-2 text-sm">
                <div className="font-medium">{event.vehicle.vehicle_number} · {event.event_type}</div>
                <div className="text-mist">{event.geofence.geofence_name}</div>
                <div className="text-xs text-mist">{formatWhen(event.timestamp)}</div>
              </li>
            ))}
          </ul>
        </div>
      </section>
      <section className="order-1 h-[48vh] min-h-[320px] lg:order-2 lg:h-auto lg:min-h-0 lg:flex-1">
        <MapBoard
          mode={mode}
          geofences={geofences}
          pins={pins}
          draft={draft}
          draftClosed={draftClosed}
          cursor={cursor}
          onMapClick={onMapClick}
        />
      </section>
    </div>
  )
}
