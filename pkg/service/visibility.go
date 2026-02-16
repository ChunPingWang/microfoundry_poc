package service

import (
	"context"
	"encoding/json"

	"github.com/younjinjeong/microfoundry/pkg/k8s"
	"github.com/younjinjeong/microfoundry/pkg/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlanVisibilityStore manages plan visibility settings stored in a single ConfigMap.
type PlanVisibilityStore struct {
	k8sClient *k8s.Client
}

// NewPlanVisibilityStore creates a new visibility store.
func NewPlanVisibilityStore(client *k8s.Client) *PlanVisibilityStore {
	return &PlanVisibilityStore{k8sClient: client}
}

// GetAll returns the full visibility map.
func (s *PlanVisibilityStore) GetAll(ctx context.Context) (*models.PlanVisibility, error) {
	cm, err := s.k8sClient.Clientset.CoreV1().ConfigMaps(s.k8sClient.Namespace).Get(
		ctx, models.PlanVisibilityConfigMapName, metav1.GetOptions{},
	)
	if err != nil {
		if errors.IsNotFound(err) {
			return &models.PlanVisibility{Plans: map[string]bool{}}, nil
		}
		return nil, err
	}

	vis := &models.PlanVisibility{Plans: map[string]bool{}}
	if data, ok := cm.Data["visibility"]; ok {
		if err := json.Unmarshal([]byte(data), vis); err != nil {
			return &models.PlanVisibility{Plans: map[string]bool{}}, nil
		}
	}
	return vis, nil
}

// SetPlanVisibility sets visibility for a single plan.
func (s *PlanVisibilityStore) SetPlanVisibility(ctx context.Context, serviceTypeID, planID string, visible bool) error {
	key := models.PlanKey(serviceTypeID, planID)
	return s.update(ctx, func(vis *models.PlanVisibility) {
		if visible {
			vis.Plans[key] = true
		} else {
			delete(vis.Plans, key)
		}
	})
}

// SetServiceVisibility sets visibility for all plans of a service type.
func (s *PlanVisibilityStore) SetServiceVisibility(ctx context.Context, serviceTypeID string, visible bool, catalog []models.ServiceType) error {
	return s.update(ctx, func(vis *models.PlanVisibility) {
		for _, svc := range catalog {
			if svc.ID == serviceTypeID {
				for _, plan := range svc.Plans {
					key := models.PlanKey(serviceTypeID, plan.ID)
					if visible {
						vis.Plans[key] = true
					} else {
						delete(vis.Plans, key)
					}
				}
				break
			}
		}
	})
}

// IsVisible returns whether a specific plan is visible to end users.
func (s *PlanVisibilityStore) IsVisible(ctx context.Context, serviceTypeID, planID string) bool {
	vis, err := s.GetAll(ctx)
	if err != nil {
		return false
	}
	return vis.Plans[models.PlanKey(serviceTypeID, planID)]
}

// VisibleCatalog returns only service types/plans that are marked visible.
func (s *PlanVisibilityStore) VisibleCatalog(ctx context.Context) ([]models.ServiceType, error) {
	vis, err := s.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	catalog := Catalog()
	var result []models.ServiceType
	for _, svc := range catalog {
		var visiblePlans []models.ServicePlan
		for _, plan := range svc.Plans {
			if vis.Plans[models.PlanKey(svc.ID, plan.ID)] {
				visiblePlans = append(visiblePlans, plan)
			}
		}
		if len(visiblePlans) > 0 {
			filtered := svc
			filtered.Plans = visiblePlans
			result = append(result, filtered)
		}
	}
	return result, nil
}

// update applies a mutation function to the visibility data with optimistic concurrency.
func (s *PlanVisibilityStore) update(ctx context.Context, mutate func(*models.PlanVisibility)) error {
	cmAPI := s.k8sClient.Clientset.CoreV1().ConfigMaps(s.k8sClient.Namespace)

	cm, err := cmAPI.Get(ctx, models.PlanVisibilityConfigMapName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// Create new ConfigMap
		vis := &models.PlanVisibility{Plans: map[string]bool{}}
		mutate(vis)
		data, _ := json.Marshal(vis)
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      models.PlanVisibilityConfigMapName,
				Namespace: s.k8sClient.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "microfoundry",
					"microfoundry.io/component":    "plan-visibility",
				},
			},
			Data: map[string]string{
				"visibility": string(data),
			},
		}
		_, err = cmAPI.Create(ctx, cm, metav1.CreateOptions{})
		return err
	}

	// Update existing ConfigMap
	vis := &models.PlanVisibility{Plans: map[string]bool{}}
	if raw, ok := cm.Data["visibility"]; ok {
		json.Unmarshal([]byte(raw), vis)
	}
	if vis.Plans == nil {
		vis.Plans = map[string]bool{}
	}
	mutate(vis)
	data, _ := json.Marshal(vis)
	cm.Data["visibility"] = string(data)
	_, err = cmAPI.Update(ctx, cm, metav1.UpdateOptions{})
	return err
}
