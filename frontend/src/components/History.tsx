import { useEffect, useState, type FormEvent } from 'react'
import { listViolations } from '../api'
import { useData } from '../context'
import { formatCoord, formatWhen } from '../format'
import type { Violation } from '../types'

const limit = 25

export default function History() {
  const { vehicles, geofences, revision } = useData()
  const [vehicleId, setVehicleId] = useState('')
  const [geofenceId, setGeofenceId] = useState('')
  const [start, setStart] = useState('')
  const [end, setEnd] = useState('')
  const [rows, setRows] = useState<Violation[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [nonce, setNonce] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    let cancel = false
    const startDate = start ? new Date(start) : null
    const endDate = end ? new Date(end) : null
    if ((startDate && Number.isNaN(startDate.getTime())) || (endDate && Number.isNaN(endDate.getTime()))) {
      setError('Enter a valid date range.')
      setBusy(false)
      return
    }
    setBusy(true)
    listViolations({
      vehicle_id: vehicleId || undefined,
      geofence_id: geofenceId || undefined,
      start_date: startDate ? startDate.toISOString() : undefined,
      end_date: endDate ? endDate.toISOString() : undefined,
      limit,
      offset,
    }).then((page) => {
      if (cancel) return
      setRows(page.violations)
      setTotal(page.total_count)
      setError(null)
    }).catch((err: unknown) => {
      if (!cancel) setError(err instanceof Error ? err.message : 'Could not load history')
    }).finally(() => {
      if (!cancel) setBusy(false)
    })
    return () => { cancel = true }
  }, [offset, nonce, revision])

  function apply(event: FormEvent) {
    event.preventDefault()
    setOffset(0)
    setNonce((value) => value + 1)
  }

  const from = total === 0 ? 0 : offset + 1
  const to = offset + rows.length

  return (
    <div className="mx-auto max-w-6xl space-y-4 p-4 md:p-6">
      <form className="grid gap-3 rounded-lg border border-line bg-panel p-4 md:grid-cols-5" onSubmit={apply}>
        <label className="block">
          <span className="label">Vehicle</span>
          <select className="input" value={vehicleId} onChange={(event) => setVehicleId(event.target.value)}>
            <option value="">All</option>
            {vehicles.map((vehicle) => <option key={vehicle.id} value={vehicle.id}>{vehicle.vehicle_number}</option>)}
          </select>
        </label>
        <label className="block">
          <span className="label">Geofence</span>
          <select className="input" value={geofenceId} onChange={(event) => setGeofenceId(event.target.value)}>
            <option value="">All</option>
            {geofences.map((fence) => <option key={fence.id} value={fence.id}>{fence.name}</option>)}
          </select>
        </label>
        <label className="block"><span className="label">From</span><input className="input" type="datetime-local" value={start} onChange={(event) => setStart(event.target.value)} /></label>
        <label className="block"><span className="label">To</span><input className="input" type="datetime-local" value={end} onChange={(event) => setEnd(event.target.value)} /></label>
        <div className="flex items-end"><button className="btn w-full" type="submit">Apply</button></div>
      </form>
      {error && <p className="text-sm text-danger">{error}</p>}
      <div className="overflow-x-auto rounded-lg border border-line">
        <table className="w-full text-sm">
          <thead className="bg-panel text-left text-xs uppercase tracking-wider text-mist">
            <tr>
              <th className="px-3 py-2">When</th>
              <th className="px-3 py-2">Vehicle</th>
              <th className="px-3 py-2">Geofence</th>
              <th className="px-3 py-2">Event</th>
              <th className="px-3 py-2">Position</th>
            </tr>
          </thead>
          <tbody>
            {busy && rows.length === 0 && <tr><td className="px-3 py-4 text-mist" colSpan={5}>Loading crossings…</td></tr>}
            {!busy && rows.length === 0 && <tr><td className="px-3 py-4 text-mist" colSpan={5}>No crossings yet. Update a vehicle location to record entry and exit events.</td></tr>}
            {rows.map((row) => (
              <tr key={row.id} className="border-t border-line">
                <td className="px-3 py-3">{formatWhen(row.timestamp)}</td>
                <td className="px-3 py-3">{row.vehicle_number}</td>
                <td className="px-3 py-3">{row.geofence_name}</td>
                <td className={`px-3 py-3 capitalize ${row.event_type === 'entry' ? 'text-amber' : 'text-glide'}`}>{row.event_type}</td>
                <td className="px-3 py-3 font-mono text-xs">{formatCoord(row.latitude)}, {formatCoord(row.longitude)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex items-center justify-between text-sm text-mist">
        <span>{busy ? 'Loading…' : `Showing ${from}–${to} of ${total}`}</span>
        <div className="flex gap-2">
          <button type="button" className="btn-ghost" disabled={offset === 0 || busy} onClick={() => setOffset((value) => Math.max(0, value - limit))}>Previous</button>
          <button type="button" className="btn-ghost" disabled={busy || offset + rows.length >= total} onClick={() => setOffset((value) => value + limit)}>Next</button>
        </div>
      </div>
    </div>
  )
}
