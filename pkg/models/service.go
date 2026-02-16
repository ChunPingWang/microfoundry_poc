package models

// ServiceBindingInfo represents a service bound to an app (display only).
type ServiceBindingInfo struct {
	Name   string `json:"name"`
	Type   string `json:"type"`   // managed, user-provided
	Status string `json:"status"` // bound, binding, failed
}

// SecretInfo represents K8s Secret metadata (no values exposed).
type SecretInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // Opaque, kubernetes.io/tls, etc.
	KeyCount  int    `json:"key_count"`
	CreatedAt string `json:"created_at"`
}
