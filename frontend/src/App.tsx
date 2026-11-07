import { useEffect, useState } from 'react'
import Alerts from './components/Alerts'
import Geofences from './components/Geofences'
import History from './components/History'
import Operations from './components/Operations'
import Vehicles from './components/Vehicles'
import type { MapMode } from './components/MapBoard'
import { API_URL } from './api'
import { DataProvider, useData } from './context'
import { categoryMeta } from './categories'
import { formatCoord, formatWhen } from './format'
import type { AlertEvent } from './types'

type Tab = 'map' | 'geofences' | 'vehicles' | 'alerts' | 'history'

const tabs: { id: Tab; label: string }[] = [
  { id: 'map', label: 'Live map' },
  { id: 'geofences', label: 'Geofences' },
  { id: 'vehicles', label: 'Vehicles' },
  { id: 'alerts', label: 'Alert rules' },
  { id: 'history', label: 'Crossings' },
]

export default function App() {
  return (
    <DataProvider>
      <Shell />
    </DataProvider>
  )
}

function Shell() {
  const { banner, live, vehicles } = useData()
  const [tab, setTab] = useState<Tab>('map')
  const [mode, setMode] = useState<MapMode>('view')
  const [activeVehicle, setActiveVehicle] = useState('')
  const [unread, setUnread] = useState(0)
  const latest = live.events[0]?.event_id

  useEffect(() => {
    if (!activeVehicle && vehicles[0]) setActiveVehicle(vehicles[0].id)
  }, [vehicles, activeVehicle])

  useEffect(() => {
    if (tab === 'map' || tab === 'alerts') setUnread(0)
  }, [tab])

  useEffect(() => {
    if (!latest) return
    if (tab === 'map' || tab === 'alerts') return
    setUnread((count) => count + 1)
  }, [latest, tab])

  function placeVehicle(id: string) {
    setActiveVehicle(id)
    setMode('place')
    setTab('map')
  }

  return (
    <div className="flex h-screen bg-ink">
      <aside className="hidden w-60 shrink-0 flex-col border-r border-line bg-panel md:flex">
        <div className="border-b border-line px-4 py-5">
          <div className="text-xs uppercase tracking-[0.18em] text-amber">Control</div>
          <h1 className="text-xl font-semibold">Geofence</h1>
        </div>
        <nav className="flex flex-1 flex-col gap-1 p-3">
          {tabs.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setTab(item.id)}
              className={`flex items-center justify-between rounded-md px-3 py-2 text-left text-sm ${tab === item.id ? 'bg-ink text-paper' : 'text-mist hover:bg-ink/60 hover:text-paper'}`}
            >
              {item.label}
              {item.id === 'alerts' && unread > 0 && (
                <span className="rounded-full bg-amber px-1.5 text-xs font-semibold text-ink">{unread}</span>
              )}
            </button>
          ))}
        </nav>
        <div className="border-t border-line px-4 py-3 text-xs text-mist">
          <div className="mb-1 flex items-center gap-2">
            <span className={`h-2 w-2 rounded-full ${live.status === 'live' ? 'bg-signal' : live.status === 'connecting' ? 'bg-amber' : 'bg-danger'}`} />
            {live.status === 'live' ? 'Alerts live' : live.status === 'connecting' ? 'Connecting' : 'Reconnecting'}
          </div>
          <div className="truncate font-mono">{API_URL}</div>
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center justify-between gap-3 border-b border-line px-4 py-3">
          <div>
            <h2 className="text-lg font-semibold">{tabs.find((item) => item.id === tab)?.label}</h2>
            <p className="text-xs text-mist">Boundaries, vehicles, and live crossings</p>
          </div>
          <div className="flex items-center gap-2 text-xs text-mist md:hidden">
            <span className={`h-2 w-2 rounded-full ${live.status === 'live' ? 'bg-signal' : 'bg-amber'}`} />
            {live.status}
          </div>
        </header>
        <nav className="flex gap-2 overflow-x-auto border-b border-line px-3 py-2 md:hidden">
          {tabs.map((item) => (
            <button key={item.id} type="button" onClick={() => setTab(item.id)} className={`shrink-0 rounded-full px-3 py-1 text-sm ${tab === item.id ? 'bg-amber text-ink' : 'border border-line text-mist'}`}>
              {item.label}
            </button>
          ))}
        </nav>
        {banner && <div className="border-b border-danger/40 bg-danger/10 px-4 py-2 text-sm text-danger">{banner}</div>}
        <main className={tab === 'map' ? 'min-h-0 flex-1 overflow-hidden' : 'min-h-0 flex-1 overflow-auto'}>
          {tab === 'map' && (
            <Operations mode={mode} setMode={setMode} activeVehicle={activeVehicle} setActiveVehicle={setActiveVehicle} />
          )}
          {tab === 'geofences' && <Geofences onShowMap={() => { setMode('draw'); setTab('map') }} />}
          {tab === 'vehicles' && <Vehicles onPlace={placeVehicle} />}
          {tab === 'alerts' && <Alerts />}
          {tab === 'history' && <History />}
        </main>
      </div>
      <Toasts events={live.toasts} onDismiss={live.dismissToast} />
    </div>
  )
}

function Toasts({ events, onDismiss }: { events: AlertEvent[]; onDismiss: (id: string) => void }) {
  if (events.length === 0) return null
  return (
    <div aria-live="polite" className="pointer-events-none fixed right-4 top-4 z-[1000] flex w-[min(100%-2rem,22rem)] flex-col gap-2">
      {events.map((event) => {
        const meta = categoryMeta(event.geofence.category)
        return (
          <article key={event.event_id} className={`pointer-events-auto rounded-lg border border-line bg-panel p-3 shadow-lg ${event.event_type === 'entry' ? 'border-l-4 border-l-amber' : 'border-l-4 border-l-glide'}`}>
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-sm font-semibold capitalize">{event.event_type} · {event.vehicle.vehicle_number}</p>
                <p className="text-sm text-mist">{event.vehicle.driver_name}</p>
              </div>
              <button type="button" className="text-xs text-mist hover:text-paper" onClick={() => onDismiss(event.event_id)}>Dismiss</button>
            </div>
            <p className="mt-2 text-sm">{event.geofence.geofence_name}</p>
            <p className="text-xs" style={{ color: meta.color }}>{meta.label}</p>
            <p className="mt-1 font-mono text-xs text-mist">{formatCoord(event.location.latitude)}, {formatCoord(event.location.longitude)} · {formatWhen(event.timestamp)}</p>
          </article>
        )
      })}
    </div>
  )
}
