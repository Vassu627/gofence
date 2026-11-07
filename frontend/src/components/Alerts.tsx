import { useEffect, useState, type FormEvent } from 'react'
import { createAlert, listAlerts } from '../api'
import { categoryMeta } from '../categories'
import { useData } from '../context'
import { formatWhen } from '../format'
import type { AlertRule, EventType } from '../types'

export default function Alerts() {
  const { geofences, vehicles, refresh, revision } = useData()
  const [fenceFilter, setFenceFilter] = useState('')
  const [vehicleFilter, setVehicleFilter] = useState('')
  const [rows, setRows] = useState<AlertRule[]>([])
  const [geofenceId, setGeofenceId] = useState('')
  const [vehicleId, setVehicleId] = useState('')
  const [eventType, setEventType] = useState<EventType>('entry')
  const [error, setError] = useState<string | null>(null)
  const [note, setNote] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    let cancel = false
    listAlerts({
      geofence_id: fenceFilter || undefined,
      vehicle_id: vehicleFilter || undefined,
    }).then((page) => {
      if (!cancel) setRows(page.alerts)
    }).catch((err: unknown) => {
      if (!cancel) setError(err instanceof Error ? err.message : 'Could not load alert rules')
    })
    return () => { cancel = true }
  }, [fenceFilter, vehicleFilter, revision])

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError(null)
    setNote(null)
    try {
      if (!geofenceId) throw new Error('Choose a geofence.')
      await createAlert({
        geofence_id: geofenceId,
        event_type: eventType,
        ...(vehicleId ? { vehicle_id: vehicleId } : {}),
      })
      await refresh()
      setNote('Alert rule saved.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save the rule.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6 p-4 md:p-6">
      <form className="grid gap-3 rounded-lg border border-line bg-panel p-4 md:grid-cols-4" onSubmit={onSubmit}>
        <label className="block md:col-span-1">
          <span className="label">Geofence</span>
          <select className="input" value={geofenceId} onChange={(event) => setGeofenceId(event.target.value)}>
            <option value="">Select</option>
            {geofences.map((fence) => <option key={fence.id} value={fence.id}>{fence.name}</option>)}
          </select>
        </label>
        <label className="block">
          <span className="label">Vehicle</span>
          <select className="input" value={vehicleId} onChange={(event) => setVehicleId(event.target.value)}>
            <option value="">All vehicles</option>
            {vehicles.map((vehicle) => <option key={vehicle.id} value={vehicle.id}>{vehicle.vehicle_number}</option>)}
          </select>
        </label>
        <label className="block">
          <span className="label">Event</span>
          <select className="input" value={eventType} onChange={(event) => setEventType(event.target.value as EventType)}>
            <option value="entry">Entry</option>
            <option value="exit">Exit</option>
            <option value="both">Entry and exit</option>
          </select>
        </label>
        <div className="flex items-end">
          <button className="btn w-full" disabled={busy} type="submit">{busy ? 'Saving…' : 'Save rule'}</button>
        </div>
        {error && <p className="text-sm text-danger md:col-span-4">{error}</p>}
        {note && <p className="text-sm text-signal md:col-span-4">{note}</p>}
      </form>
      <div className="flex flex-wrap gap-2">
        <select className="input w-56" value={fenceFilter} onChange={(event) => setFenceFilter(event.target.value)}>
          <option value="">All geofences</option>
          {geofences.map((fence) => <option key={fence.id} value={fence.id}>{fence.name}</option>)}
        </select>
        <select className="input w-56" value={vehicleFilter} onChange={(event) => setVehicleFilter(event.target.value)}>
          <option value="">All rule targets</option>
          {vehicles.map((vehicle) => <option key={vehicle.id} value={vehicle.id}>{vehicle.vehicle_number}</option>)}
        </select>
      </div>
      <div className="overflow-x-auto rounded-lg border border-line">
        <table className="w-full text-sm">
          <thead className="bg-panel text-left text-xs uppercase tracking-wider text-mist">
            <tr>
              <th className="px-3 py-2">Geofence</th>
              <th className="px-3 py-2">Vehicle</th>
              <th className="px-3 py-2">Event</th>
              <th className="px-3 py-2">Created</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && <tr><td className="px-3 py-4 text-mist" colSpan={4}>No alert rules yet.</td></tr>}
            {rows.map((rule) => (
              <tr key={rule.alert_id} className="border-t border-line">
                <td className="px-3 py-3">
                  <div>{rule.geofence_name}</div>
                  <div className="text-xs" style={{ color: categoryMeta(geofences.find((fence) => fence.id === rule.geofence_id)?.category || '').color }}>
                    {rule.status}
                  </div>
                </td>
                <td className="px-3 py-3">{rule.vehicle_number || 'All vehicles'}</td>
                <td className="px-3 py-3 capitalize">{rule.event_type}</td>
                <td className="px-3 py-3 text-mist">{formatWhen(rule.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
