export const sampleDelivery = {
  name: 'Downtown Delivery Zone',
  description: 'Main delivery area for downtown customers',
  category: 'delivery_zone' as const,
  coordinates: [
    [37.7749, -122.4194],
    [37.7849, -122.4194],
    [37.7849, -122.4094],
    [37.7749, -122.4094],
    [37.7749, -122.4194],
  ],
}

export const sampleRestricted = {
  name: 'Port Restricted Yard',
  description: 'Restricted loading yard inside the downtown zone',
  category: 'restricted_zone' as const,
  coordinates: [
    [37.781, -122.416],
    [37.784, -122.416],
    [37.784, -122.411],
    [37.781, -122.411],
    [37.781, -122.416],
  ],
}

export const sampleVehicle = {
  vehicle_number: 'KA-01-AB-1234',
  driver_name: 'John Doe',
  vehicle_type: 'truck',
  phone: '+1234567890',
}

export const samplePath = [
  { latitude: 37.77, longitude: -122.4144, label: 'Posted a point outside both zones.' },
  { latitude: 37.778, longitude: -122.418, label: 'Entered the delivery zone.' },
  { latitude: 37.7825, longitude: -122.4135, label: 'Entered the restricted yard.' },
]

export function wait(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
