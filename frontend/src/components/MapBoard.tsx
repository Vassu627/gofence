import { useEffect, useRef } from 'react'
import L from 'leaflet'
import { CircleMarker, MapContainer, Marker, Polygon, Polyline, Popup, TileLayer, useMap, useMapEvents } from 'react-leaflet'
import { categoryMeta } from '../categories'
import { formatCoord } from '../format'
import type { Geofence } from '../types'

export type MapMode = 'view' | 'draw' | 'place'

export type VehiclePin = {
  id: string
  label: string
  driver: string
  lat: number
  lng: number
  restricted: boolean
  detail: string
}

const normalIcon = L.divIcon({
  className: 'veh-icon',
  html: '<span class="veh-dot"></span>',
  iconSize: [18, 18],
  iconAnchor: [9, 9],
})

const hotIcon = L.divIcon({
  className: 'veh-icon',
  html: '<span class="veh-dot is-hot"></span>',
  iconSize: [18, 18],
  iconAnchor: [9, 9],
})

type Props = {
  mode: MapMode
  geofences: Geofence[]
  pins: VehiclePin[]
  draft: [number, number][]
  draftClosed: boolean
  cursor: [number, number] | null
  onMapClick: (lat: number, lng: number) => void
}

export default function MapBoard({ mode, geofences, pins, draft, draftClosed, cursor, onMapClick }: Props) {
  const focus = geofences.length
    ? geofences.flatMap((fence) => fence.coordinates)
    : pins.map((pin) => [pin.lat, pin.lng] as [number, number])
  const signature = geofences.length
    ? geofences.map((fence) => fence.id).join('|')
    : pins.map((pin) => pin.id).join('|')

  return (
    <div className="relative h-full min-h-[320px]">
      <MapContainer
        center={[37.7799, -122.4144]}
        zoom={13}
        className={`h-full w-full ${mode === 'view' ? '' : 'drawing'}`}
        scrollWheelZoom
      >
        <TileLayer
          attribution='&copy; OpenStreetMap contributors &copy; CARTO'
          url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
        />
        <Invalidate />
        <FitTo signature={signature} points={focus} />
        <MapClick enabled={mode !== 'view'} onClick={onMapClick} />
        {geofences.map((fence) => {
          const meta = categoryMeta(fence.category)
          return (
            <Polygon
              key={fence.id}
              positions={fence.coordinates}
              pathOptions={{ color: meta.color, weight: fence.category === 'restricted_zone' ? 3 : 2, fillColor: meta.color, fillOpacity: 0.22 }}
            >
              <Popup>
                <strong>{fence.name}</strong>
                <div>{meta.label}</div>
              </Popup>
            </Polygon>
          )
        })}
        {pins.map((pin) => (
          <Marker key={pin.id} position={[pin.lat, pin.lng]} icon={pin.restricted ? hotIcon : normalIcon} zIndexOffset={500}>
            <Popup>
              <strong>{pin.label}</strong>
              <div>{pin.driver}</div>
              <div>{pin.detail}</div>
              <div className="font-mono text-xs">{formatCoord(pin.lat)}, {formatCoord(pin.lng)}</div>
            </Popup>
          </Marker>
        ))}
        {!draftClosed && draft.length >= 2 && (
          <Polyline positions={draft} pathOptions={{ color: '#e0a425', weight: 2, dashArray: '4 6' }} />
        )}
        {draftClosed && draft.length >= 4 && (
          <Polygon positions={draft} pathOptions={{ color: '#e0a425', weight: 2, fillColor: '#e0a425', fillOpacity: 0.16 }} />
        )}
        {draft.map((point, index) => (
          <CircleMarker
            key={`${point[0]}-${point[1]}-${index}`}
            center={point}
            radius={index === 0 ? 6 : 4}
            pathOptions={{ color: '#e0a425', fillColor: '#e0a425', fillOpacity: 1 }}
          />
        ))}
        {mode === 'place' && cursor && (
          <CircleMarker center={cursor} radius={8} pathOptions={{ color: '#e8eee9', fillColor: '#e0a425', fillOpacity: 0.95 }} />
        )}
      </MapContainer>
      <div className="pointer-events-none absolute bottom-3 left-3 z-[800] rounded-md border border-line bg-panel/95 px-3 py-2 text-xs text-mist">
        <div className="mb-1 font-medium uppercase tracking-wider">Zones</div>
        <ul className="space-y-1">
          {['delivery_zone', 'restricted_zone', 'toll_zone', 'customer_area'].map((id) => {
            const meta = categoryMeta(id)
            return (
              <li key={id} className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 rounded-sm" style={{ background: meta.color }} />
                {meta.label}
              </li>
            )
          })}
        </ul>
      </div>
    </div>
  )
}

function MapClick({ enabled, onClick }: { enabled: boolean; onClick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(event) {
      if (!enabled) return
      onClick(event.latlng.lat, event.latlng.lng)
    },
  })
  return null
}

function FitTo({ signature, points }: { signature: string; points: [number, number][] }) {
  const map = useMap()
  const latest = useRef(points)
  latest.current = points
  useEffect(() => {
    const current = latest.current
    if (!signature || current.length === 0) return
    map.fitBounds(L.latLngBounds(current.map(([lat, lng]) => [lat, lng])), { padding: [28, 28], animate: false })
  }, [map, signature])
  return null
}

function Invalidate() {
  const map = useMap()
  useEffect(() => {
    const fix = () => map.invalidateSize()
    const timer = window.setTimeout(fix, 150)
    window.addEventListener('resize', fix)
    return () => {
      window.clearTimeout(timer)
      window.removeEventListener('resize', fix)
    }
  }, [map])
  return null
}
