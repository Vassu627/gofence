package model

func ValidCategory(s string) bool {
	switch s {
	case "delivery_zone", "restricted_zone", "toll_zone", "customer_area":
		return true
	default:
		return false
	}
}

func ValidEvent(s string) bool {
	switch s {
	case "entry", "exit", "both":
		return true
	default:
		return false
	}
}

type Geofence struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Coordinates [][]float64 `json:"coordinates"`
	Category    string      `json:"category"`
	CreatedAt   string      `json:"created_at"`
}

type Vehicle struct {
	ID            string `json:"id"`
	VehicleNumber string `json:"vehicle_number"`
	DriverName    string `json:"driver_name"`
	VehicleType   string `json:"vehicle_type"`
	Phone         string `json:"phone"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

type InsideGeofence struct {
	GeofenceID   string `json:"geofence_id"`
	GeofenceName string `json:"geofence_name"`
	Status       string `json:"status"`
}

type LocationUpdate struct {
	VehicleID        string           `json:"vehicle_id"`
	LocationUpdated  bool             `json:"location_updated"`
	CurrentGeofences []InsideGeofence `json:"current_geofences"`
	Alerts           []AlertMessage   `json:"-"`
}

type Fix struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp string  `json:"timestamp"`
}

type OccupiedGeofence struct {
	GeofenceID   string `json:"geofence_id"`
	GeofenceName string `json:"geofence_name"`
	Category     string `json:"category"`
}

type VehicleLocation struct {
	VehicleID        string             `json:"vehicle_id"`
	VehicleNumber    string             `json:"vehicle_number"`
	CurrentLocation  *Fix               `json:"current_location"`
	CurrentGeofences []OccupiedGeofence `json:"current_geofences"`
}

type AlertRule struct {
	AlertID       string  `json:"alert_id"`
	GeofenceID    string  `json:"geofence_id"`
	GeofenceName  string  `json:"geofence_name"`
	VehicleID     *string `json:"vehicle_id"`
	VehicleNumber *string `json:"vehicle_number"`
	EventType     string  `json:"event_type"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
}

type Violation struct {
	ID            string  `json:"id"`
	VehicleID     string  `json:"vehicle_id"`
	VehicleNumber string  `json:"vehicle_number"`
	GeofenceID    string  `json:"geofence_id"`
	GeofenceName  string  `json:"geofence_name"`
	EventType     string  `json:"event_type"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Timestamp     string  `json:"timestamp"`
}

type ViolationPage struct {
	Violations []Violation `json:"violations"`
	TotalCount int         `json:"total_count"`
}

type AlertMessage struct {
	EventID   string        `json:"event_id"`
	EventType string        `json:"event_type"`
	Timestamp string        `json:"timestamp"`
	Vehicle   AlertVehicle  `json:"vehicle"`
	Geofence  AlertGeofence `json:"geofence"`
	Location  AlertLocation `json:"location"`
}

type AlertVehicle struct {
	VehicleID     string `json:"vehicle_id"`
	VehicleNumber string `json:"vehicle_number"`
	DriverName    string `json:"driver_name"`
}

type AlertGeofence struct {
	GeofenceID   string `json:"geofence_id"`
	GeofenceName string `json:"geofence_name"`
	Category     string `json:"category"`
}

type AlertLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
