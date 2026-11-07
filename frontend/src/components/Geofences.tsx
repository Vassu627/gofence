import { useEffect, useState, type FormEvent } from 'react'
import { createAlert, createGeofence, listGeofences } from '../api'
import { CATEGORIES, categoryMeta } from '../categories'
import { parseCoordinates } from '../coords'
import { useData } from '../context'
import { formatWhen } from '../format'
import type { Category, Geofence } from '../types'

export default function Geofences({ onShowMap }: { onShowMap: () => void }) {
  const { geofences, refresh, revision } = useData()
  const [category, setCategory] = useState('')
  const [filtered, setFiltered] = useState<Geofence[] | null>(null)
  const [filterError, setFilterError] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [fenceCategory, setFenceCategory] = useState<Category>('delivery_zone')
  const [coordinates, setCoordinates] = useState('')
  const [arm, setArm] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [note, setNote] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!category) {
      setFiltered(null)
      return
    }
    let cancel = false
    listGeofences(category)
      .then((page) => { if (!cancel) setFiltered(page.geofences) })
      .catch((err: unknown) => { if (!cancel) setFilterError(err instanceof Error ? err.message : 'Could not filter geofences') })
    return () => { cancel = true }
  }, [category, revision])

  const shown = category ? filtered ?? [] : geofences

  function chooseCategory(next: Category) {
    setFenceCategory(next)
    if (next === 'restricted_zone') setArm(true)
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setNote(null)
    setBusy(true)
    try {
      const created = await createGeofence({
        name: name.trim(),
        description: description.trim(),
        category: fenceCategory,
        coordinates: parseCoordinates(coordinates),
      })
      if (arm) await createAlert({ geofence_id: created.id, event_type: 'both' })
      await refresh()
      setName('')
      setDescription('')
      setCoordinates('')
      setArm(false)
      setNote(arm ? 'Geofence created and alert armed.' : 'Geofence created.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not create the geofence.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto grid max-w-6xl gap-6 p-4 md:p-6 lg:grid-cols-[320px_1fr]">
      <form className="space-y-3 rounded-lg border border-line bg-panel p-4" onSubmit={onSubmit}>
        <h2 className="text-lg font-semibold">New geofence</h2>
        <p className="text-sm text-mist">Paste JSON pairs or one latitude, longitude per line. The ring is closed for you.</p>
        {error && <p className="text-sm text-danger">{error}</p>}
        {note && <p className="text-sm text-signal">{note}</p>}
        <label className="block"><span className="label">Name</span><input className="input" value={name} onChange={(event) => setName(event.target.value)} required /></label>
        <label className="block"><span className="label">Description</span><textarea className="input min-h-20" value={description} onChange={(event) => setDescription(event.target.value)} /></label>
        <label className="block">
          <span className="label">Category</span>
          <select className="input" value={fenceCategory} onChange={(event) => chooseCategory(event.target.value as Category)}>
            {CATEGORIES.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
          </select>
        </label>
        <label className="block">
          <span className="label">Coordinates</span>
          <textarea className="input min-h-36 font-mono text-xs" value={coordinates} onChange={(event) => setCoordinates(event.target.value)} placeholder={'37.7749, -122.4194\n37.7849, -122.4194\n37.7849, -122.4094\n37.7749, -122.4094'} />
        </label>
        <label className="flex items-start gap-2 text-sm">
          <input type="checkbox" className="mt-1" checked={arm} onChange={(event) => setArm(event.target.checked)} />
          <span>Also create an entry and exit alert for every vehicle.</span>
        </label>
        <button className="btn w-full" disabled={busy} type="submit">{busy ? 'Saving…' : 'Create geofence'}</button>
        <button type="button" className="btn-ghost w-full" onClick={onShowMap}>Draw on the map instead</button>
      </form>
      <section>
        <div className="mb-3 flex items-center justify-between gap-3">
          <h2 className="text-lg font-semibold">All geofences</h2>
          <select className="input w-48" value={category} onChange={(event) => { setFilterError(null); setCategory(event.target.value) }}>
            <option value="">All categories</option>
            {CATEGORIES.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
          </select>
        </div>
        {filterError && <p className="mb-3 text-sm text-danger">{filterError}</p>}
        <div className="space-y-3">
          {shown.length === 0 && <p className="text-sm text-mist">No geofences in this view yet.</p>}
          {shown.map((fence) => {
            const meta = categoryMeta(fence.category)
            return (
              <article key={fence.id} className="rounded-lg border border-line bg-panel p-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h3 className="font-semibold">{fence.name}</h3>
                    <p className="text-sm text-mist">{fence.description || 'No description'}</p>
                  </div>
                  <span className="rounded-full px-2 py-1 text-xs" style={{ color: meta.color, border: `1px solid ${meta.color}` }}>{meta.label}</span>
                </div>
                <p className="mt-3 font-mono text-xs text-mist">{fence.coordinates.length} points · {formatWhen(fence.created_at)} · {fence.id}</p>
              </article>
            )
          })}
        </div>
      </section>
    </div>
  )
}
