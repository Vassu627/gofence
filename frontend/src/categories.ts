import type { Category } from './types'

export const CATEGORIES: { id: Category; label: string; color: string }[] = [
  { id: 'delivery_zone', label: 'Delivery zone', color: '#2f9e8f' },
  { id: 'restricted_zone', label: 'Restricted zone', color: '#e15a45' },
  { id: 'toll_zone', label: 'Toll zone', color: '#e0b15a' },
  { id: 'customer_area', label: 'Customer area', color: '#4c8dff' },
]

export const VEHICLE_TYPES = ['truck', 'car', 'van', 'motorcycle']

export function categoryMeta(id: string) {
  return CATEGORIES.find((item) => item.id === id) ?? { id, label: id, color: '#93a099' }
}
