import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { getVehicleLocation, listAlerts, listGeofences, listVehicles } from './api'
import { useLiveAlerts } from './hooks/useLiveAlerts'
import type { AlertRule, Geofence, Vehicle, VehicleLocation } from './types'

export type Snapshot = {
  geofences: Geofence[]
  vehicles: Vehicle[]
  alertRules: AlertRule[]
  fixes: Record<string, VehicleLocation>
}

type DataValue = Snapshot & {
  loading: boolean
  banner: string | null
  revision: number
  refresh: () => Promise<Snapshot>
  live: ReturnType<typeof useLiveAlerts>
}

const DataContext = createContext<DataValue | null>(null)

export function DataProvider({ children }: { children: ReactNode }) {
  const live = useLiveAlerts()
  const [geofences, setGeofences] = useState<Geofence[]>([])
  const [vehicles, setVehicles] = useState<Vehicle[]>([])
  const [alertRules, setAlertRules] = useState<AlertRule[]>([])
  const [fixes, setFixes] = useState<Record<string, VehicleLocation>>({})
  const [loading, setLoading] = useState(true)
  const [banner, setBanner] = useState<string | null>(null)
  const [revision, setRevision] = useState(0)
  const first = useRef(true)

  const refresh = useCallback(async () => {
    try {
      const [fencePage, vehiclePage, alertPage] = await Promise.all([
        listGeofences(),
        listVehicles(),
        listAlerts(),
      ])
      const located = await Promise.all(
        vehiclePage.vehicles.map(async (vehicle) => {
          try {
            return [vehicle.id, await getVehicleLocation(vehicle.id)] as const
          } catch {
            return null
          }
        }),
      )
      const nextFixes: Record<string, VehicleLocation> = {}
      located.forEach((entry) => {
        if (entry) nextFixes[entry[0]] = entry[1]
      })
      const snapshot = {
        geofences: fencePage.geofences,
        vehicles: vehiclePage.vehicles,
        alertRules: alertPage.alerts,
        fixes: nextFixes,
      }
      setGeofences(snapshot.geofences)
      setVehicles(snapshot.vehicles)
      setAlertRules(snapshot.alertRules)
      setFixes(snapshot.fixes)
      setBanner(null)
      setRevision((value) => value + 1)
      return snapshot
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Could not load data'
      setBanner(message)
      throw err instanceof Error ? err : new Error(message)
    } finally {
      if (first.current) {
        first.current = false
        setLoading(false)
      }
    }
  }, [])

  useEffect(() => {
    refresh().catch(() => {})
  }, [refresh])

  const latest = live.events[0]?.event_id
  useEffect(() => {
    if (!latest) return
    refresh().catch(() => {})
  }, [latest, refresh])

  return (
    <DataContext.Provider value={{ geofences, vehicles, alertRules, fixes, loading, banner, revision, refresh, live }}>
      {children}
    </DataContext.Provider>
  )
}

export function useData() {
  const value = useContext(DataContext)
  if (!value) throw new Error('useData must be used inside DataProvider')
  return value
}
