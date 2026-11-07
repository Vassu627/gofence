import { useState, type FormEvent } from 'react'
import { createVehicle, updateLocation } from '../api'
import { VEHICLE_TYPES } from '../categories'
import { useData } from '../context'
import { formatCoord, formatWhen } from '../format'

export default function Vehicles({ onPlace }: { onPlace: (id: string) => void }) {
  const { vehicles, fixes, refresh } = useData()
  const [number, setNumber] = useState('')
  const [driver, setDriver] = useState('')
  const [type, setType] = useState('truck')
  const [phone, setPhone] = useState('')
  const [vehicleId, setVehicleId] = useState('')
  const [lat, setLat] = useState('')
  const [lng, setLng] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [note, setNote] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function register(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError(null)
    setNote(null)
    try {
      await createVehicle({
        vehicle_number: number.trim(),
        driver_name: driver.trim(),
        vehicle_type: type,
        phone: phone.trim(),
      })
      await refresh()
      setNumber('')
      setDriver('')
      setPhone('')
      setNote('Vehicle registered.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not register the vehicle.')
    } finally {
      setBusy(false)
    }
  }

  async function locate(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError(null)
    setNote(null)
    try {
      const latitude = Number(lat)
      const longitude = Number(lng)
      if (!vehicleId || !Number.isFinite(latitude) || !Number.isFinite(longitude)) {
        throw new Error('Choose a vehicle and enter numeric coordinates.')
      }
      const result = await updateLocation({
        vehicle_id: vehicleId,
        latitude,
        longitude,
        timestamp: new Date().toISOString(),
      })
      const names = result.current_geofences.map((fence) => fence.geofence_name)
      setNote(names.length ? `Inside ${names.join(', ')}.` : 'Location saved. Outside every geofence.')
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not update the location.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6 p-4 md:p-6">
      {error && <p className="text-sm text-danger">{error}</p>}
      {note && <p className="text-sm text-signal">{note}</p>}
      <div className="grid gap-4 lg:grid-cols-2">
        <form className="space-y-3 rounded-lg border border-line bg-panel p-4" onSubmit={register}>
          <h2 className="text-lg font-semibold">Register a vehicle</h2>
          <label className="block"><span className="label">Vehicle number</span><input className="input" value={number} onChange={(event) => setNumber(event.target.value)} placeholder="KA-01-AB-1234" required /></label>
          <label className="block"><span className="label">Driver</span><input className="input" value={driver} onChange={(event) => setDriver(event.target.value)} required /></label>
          <label className="block">
            <span className="label">Type</span>
            <select className="input" value={type} onChange={(event) => setType(event.target.value)}>
              {VEHICLE_TYPES.map((item) => <option key={item} value={item}>{item}</option>)}
            </select>
          </label>
          <label className="block"><span className="label">Phone</span><input className="input" value={phone} onChange={(event) => setPhone(event.target.value)} placeholder="+1234567890" required /></label>
          <button className="btn" disabled={busy} type="submit">{busy ? 'Saving…' : 'Register'}</button>
        </form>
        <form className="space-y-3 rounded-lg border border-line bg-panel p-4" onSubmit={locate}>
          <h2 className="text-lg font-semibold">Update a location</h2>
          <label className="block">
            <span className="label">Vehicle</span>
            <select className="input" value={vehicleId} onChange={(event) => setVehicleId(event.target.value)} required>
              <option value="">Select a vehicle</option>
              {vehicles.map((vehicle) => <option key={vehicle.id} value={vehicle.id}>{vehicle.vehicle_number}</option>)}
            </select>
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="block"><span className="label">Latitude</span><input className="input font-mono" value={lat} onChange={(event) => setLat(event.target.value)} required /></label>
            <label className="block"><span className="label">Longitude</span><input className="input font-mono" value={lng} onChange={(event) => setLng(event.target.value)} required /></label>
          </div>
          <button className="btn" disabled={busy} type="submit">{busy ? 'Saving…' : 'Send fix'}</button>
        </form>
      </div>
      <div className="overflow-x-auto rounded-lg border border-line">
        <table className="w-full text-sm">
          <thead className="bg-panel text-left text-xs uppercase tracking-wider text-mist">
            <tr>
              <th className="px-3 py-2">Vehicle</th>
              <th className="px-3 py-2">Driver</th>
              <th className="px-3 py-2">Type</th>
              <th className="px-3 py-2">Phone</th>
              <th className="px-3 py-2">Location</th>
              <th className="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {vehicles.length === 0 && (
              <tr><td className="px-3 py-4 text-mist" colSpan={6}>No vehicles registered yet.</td></tr>
            )}
            {vehicles.map((vehicle) => {
              const fix = fixes[vehicle.id]
              const names = fix?.current_geofences.map((fence) => fence.geofence_name) ?? []
              return (
                <tr key={vehicle.id} className="border-t border-line">
                  <td className="px-3 py-3">
                    <div className="font-medium">{vehicle.vehicle_number}</div>
                    <div className="text-xs text-mist">{vehicle.status} · {formatWhen(vehicle.created_at)}</div>
                  </td>
                  <td className="px-3 py-3">{vehicle.driver_name}</td>
                  <td className="px-3 py-3 capitalize">{vehicle.vehicle_type}</td>
                  <td className="px-3 py-3 font-mono text-xs">{vehicle.phone}</td>
                  <td className="px-3 py-3">
                    {fix?.current_location ? (
                      <>
                        <div className="font-mono text-xs">{formatCoord(fix.current_location.latitude)}, {formatCoord(fix.current_location.longitude)}</div>
                        <div className="text-xs text-mist">{names.length ? names.join(', ') : 'Outside all geofences'}</div>
                      </>
                    ) : <span className="text-mist">No fix yet</span>}
                  </td>
                  <td className="px-3 py-3 text-right">
                    <button type="button" className="btn-ghost" onClick={() => onPlace(vehicle.id)}>Place on map</button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
