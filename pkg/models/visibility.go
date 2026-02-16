package models

const PlanVisibilityConfigMapName = "mf-plan-visibility"

// PlanVisibility tracks which service plans are exposed to end users.
// Plans not present in the map default to hidden (secure by default).
type PlanVisibility struct {
	Plans map[string]bool `json:"plans"`
}

// PlanKey returns the canonical map key for a service type + plan combination.
func PlanKey(serviceTypeID, planID string) string {
	return serviceTypeID + "/" + planID
}
