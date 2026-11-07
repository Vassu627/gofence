export type Category = 'delivery_zone' | 'restricted_zone' | 'toll_zone' | 'customer_area'
export type EventType = 'entry' | 'exit' | 'both'
export type Crossing = 'entry' | 'exit'

export type Geofence = {
  id: string
  name: string
  description: string
  coordinates: [number, number][]
  category: Category
  created_at: string
}

export type Vehicle = {
  id: string
  vehicle_number: string
  driver_name: string
  vehicle_type: string
  phone: string
  status: string
  created_at: string
}

export type InsideFence = {
  geofence_id: string
  geofence_name: string
  status: string
}

export type OccupiedFence = {
  geofence_id: string
  geofence_name: string
  category: Category
}

export type VehicleLocation = {
  vehicle_id: string
  vehicle_number: string
  current_location: {
    latitude: number
    longitude: number
    timestamp: string
  } | null
  current_geofences: OccupiedFence[]
}

export type LocationUpdate = {
  vehicle_id: string
  location_updated: boolean
  current_geofences: InsideFence[]
  time_ns?: string
}

export type AlertRule = {
  alert_id: string
  geofence_id: string
  geofence_name: string
  vehicle_id: string | null
  vehicle_number: string | null
  event_type: EventType
  status: string
  created_at: string
}

export type Violation = {
  id: string
  vehicle_id: string
  vehicle_number: string
  geofence_id: string
  geofence_name: string
  event_type: Crossing
  latitude: number
  longitude: number
  timestamp: string
}

export type AlertEvent = {
  event_id: string
  event_type: Crossing
  timestamp: string
  vehicle: {
    vehicle_id: string
    vehicle_number: string
    driver_name: string
  }
  geofence: {
    geofence_id: string
    geofence_name: string
    category: Category
  }
  location: {
    latitude: number
    longitude: number
  }
}
