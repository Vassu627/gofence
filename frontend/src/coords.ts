export function closeRing(points: [number, number][]): [number, number][] {
  if (points.length === 0) return []
  const first = points[0]
  const last = points[points.length - 1]
  if (first[0] === last[0] && first[1] === last[1]) return points
  return [...points, [first[0], first[1]]]
}

export function parseCoordinates(text: string): [number, number][] {
  const trimmed = text.trim()
  if (!trimmed) throw new Error('Add coordinates for the boundary.')

  let points: [number, number][]
  if (trimmed.startsWith('[')) {
    let parsed: unknown
    try {
      parsed = JSON.parse(trimmed)
    } catch {
      throw new Error('Coordinates are not valid JSON.')
    }
    if (!Array.isArray(parsed)) {
      throw new Error('Coordinates must be an array of [latitude, longitude] pairs.')
    }
    points = parsed.map((pair, index) => {
      if (!Array.isArray(pair) || pair.length !== 2) {
        throw new Error(`Coordinate ${index + 1} must be a [latitude, longitude] pair.`)
      }
      const lat = Number(pair[0])
      const lng = Number(pair[1])
      if (!Number.isFinite(lat) || !Number.isFinite(lng)) {
        throw new Error(`Coordinate ${index + 1} is not a number.`)
      }
      return [lat, lng]
    })
  } else {
    const lines = trimmed.split(/\n+/).map((line) => line.trim()).filter(Boolean)
    points = lines.map((line) => {
      const [latText, lngText] = line.split(',').map((part) => part.trim())
      const lat = Number(latText)
      const lng = Number(lngText)
      if (!Number.isFinite(lat) || !Number.isFinite(lng)) {
        throw new Error(`Could not read "${line}". Use latitude, longitude.`)
      }
      return [lat, lng]
    })
  }
  return closeRing(points)
}
