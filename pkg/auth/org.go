package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	orgLabelKey   = "microfoundry.io/type"
	orgLabelValue = "organization"
	orgPrefix     = "mf-org-"
)

// Organization represents a multi-tenant org in MicroFoundry.
type Organization struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	OwnerEmail string `json:"owner_email"`
	CreatedAt  string `json:"created_at"`
	Namespace  string `json:"namespace"`
}

// OrgMember represents a user's membership in an organization.
type OrgMember struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"` // "admin", "member", "viewer"
	JoinedAt string `json:"joined_at"`
}

// OrgStore persists organizations as ConfigMaps in the platform namespace.
type OrgStore struct {
	clientset kubernetes.Interface
	namespace string // platform namespace (e.g., "microfoundry")
}

// NewOrgStore creates an org store backed by K8s ConfigMaps.
func NewOrgStore(clientset kubernetes.Interface, namespace string) *OrgStore {
	return &OrgStore{clientset: clientset, namespace: namespace}
}

// List returns all organizations.
func (s *OrgStore) List(ctx context.Context) ([]Organization, error) {
	cms, err := s.clientset.CoreV1().ConfigMaps(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: orgLabelKey + "=" + orgLabelValue,
	})
	if err != nil {
		return nil, fmt.Errorf("listing orgs: %w", err)
	}

	var orgs []Organization
	for _, cm := range cms.Items {
		if members := cm.Data["members"]; members != "" {
			continue // skip member ConfigMaps
		}
		var org Organization
		if data := cm.Data["metadata"]; data != "" {
			if err := json.Unmarshal([]byte(data), &org); err != nil {
				continue
			}
			orgs = append(orgs, org)
		}
	}
	return orgs, nil
}

// Get returns a single organization by ID.
func (s *OrgStore) Get(ctx context.Context, orgID string) (*Organization, error) {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, orgPrefix+orgID, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting org %q: %w", orgID, err)
	}

	var org Organization
	if err := json.Unmarshal([]byte(cm.Data["metadata"]), &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// Create creates a new organization and its K8s namespace.
func (s *OrgStore) Create(ctx context.Context, name, ownerEmail string) (*Organization, error) {
	id := sanitizeID(name)

	org := Organization{
		ID:         id,
		Name:       name,
		OwnerEmail: ownerEmail,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Namespace:  "mf-" + id,
	}

	orgData, _ := json.Marshal(org)

	// Create org metadata ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      orgPrefix + id,
			Namespace: s.namespace,
			Labels:    map[string]string{orgLabelKey: orgLabelValue},
		},
		Data: map[string]string{
			"metadata": string(orgData),
		},
	}

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("organization %q already exists", name)
		}
		return nil, fmt.Errorf("creating org: %w", err)
	}

	// Create members ConfigMap with owner
	owner := OrgMember{
		Email:    ownerEmail,
		Name:     ownerEmail, // will be updated on login
		Role:     "admin",
		JoinedAt: org.CreatedAt,
	}
	membersData, _ := json.Marshal([]OrgMember{owner})

	membersCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      orgPrefix + id + "-members",
			Namespace: s.namespace,
			Labels:    map[string]string{orgLabelKey: orgLabelValue},
		},
		Data: map[string]string{
			"members": string(membersData),
		},
	}
	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Create(ctx, membersCM, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("creating members: %w", err)
	}

	// Create org namespace for workload isolation
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: org.Namespace,
			Labels: map[string]string{
				orgLabelKey:              orgLabelValue,
				"microfoundry.io/org-id": id,
			},
		},
	}
	if _, err := s.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("creating namespace: %w", err)
	}

	return &org, nil
}

// Delete removes an organization, its members, and its namespace.
func (s *OrgStore) Delete(ctx context.Context, orgID string) error {
	// Delete member ConfigMap
	_ = s.clientset.CoreV1().ConfigMaps(s.namespace).Delete(ctx, orgPrefix+orgID+"-members", metav1.DeleteOptions{})

	// Delete org ConfigMap
	if err := s.clientset.CoreV1().ConfigMaps(s.namespace).Delete(ctx, orgPrefix+orgID, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("deleting org: %w", err)
	}

	// Delete org namespace (deletes all contained resources)
	_ = s.clientset.CoreV1().Namespaces().Delete(ctx, "mf-"+orgID, metav1.DeleteOptions{})

	return nil
}

// ListMembers returns all members of an organization.
func (s *OrgStore) ListMembers(ctx context.Context, orgID string) ([]OrgMember, error) {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, orgPrefix+orgID+"-members", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting members: %w", err)
	}

	var members []OrgMember
	if err := json.Unmarshal([]byte(cm.Data["members"]), &members); err != nil {
		return nil, err
	}
	return members, nil
}

// AddMember adds a user to an organization.
func (s *OrgStore) AddMember(ctx context.Context, orgID, email, name, role string) error {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, orgPrefix+orgID+"-members", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting members: %w", err)
	}

	var members []OrgMember
	if err := json.Unmarshal([]byte(cm.Data["members"]), &members); err != nil {
		return err
	}

	// Check if already a member
	for _, m := range members {
		if m.Email == email {
			return fmt.Errorf("user %q is already a member", email)
		}
	}

	members = append(members, OrgMember{
		Email:    email,
		Name:     name,
		Role:     role,
		JoinedAt: time.Now().UTC().Format(time.RFC3339),
	})

	data, _ := json.Marshal(members)
	cm.Data["members"] = string(data)

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("updating members: %w", err)
	}
	return nil
}

// RemoveMember removes a user from an organization.
func (s *OrgStore) RemoveMember(ctx context.Context, orgID, email string) error {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, orgPrefix+orgID+"-members", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting members: %w", err)
	}

	var members []OrgMember
	if err := json.Unmarshal([]byte(cm.Data["members"]), &members); err != nil {
		return err
	}

	var updated []OrgMember
	for _, m := range members {
		if m.Email != email {
			updated = append(updated, m)
		}
	}

	if len(updated) == len(members) {
		return fmt.Errorf("user %q is not a member", email)
	}

	data, _ := json.Marshal(updated)
	cm.Data["members"] = string(data)

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("updating members: %w", err)
	}
	return nil
}

// SetMemberRole changes a member's role.
func (s *OrgStore) SetMemberRole(ctx context.Context, orgID, email, role string) error {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, orgPrefix+orgID+"-members", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting members: %w", err)
	}

	var members []OrgMember
	if err := json.Unmarshal([]byte(cm.Data["members"]), &members); err != nil {
		return err
	}

	found := false
	for i, m := range members {
		if m.Email == email {
			members[i].Role = role
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user %q is not a member", email)
	}

	data, _ := json.Marshal(members)
	cm.Data["members"] = string(data)

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("updating members: %w", err)
	}
	return nil
}

// GetUserOrgs returns all organizations a user belongs to.
func (s *OrgStore) GetUserOrgs(ctx context.Context, email string) ([]Organization, error) {
	allOrgs, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var userOrgs []Organization
	for _, org := range allOrgs {
		members, err := s.ListMembers(ctx, org.ID)
		if err != nil {
			continue
		}
		for _, m := range members {
			if m.Email == email {
				userOrgs = append(userOrgs, org)
				break
			}
		}
	}
	return userOrgs, nil
}

// EnsureDefaultOrg creates a default org for a user if they have none.
func (s *OrgStore) EnsureDefaultOrg(ctx context.Context, email, name string) (*Organization, error) {
	orgs, err := s.GetUserOrgs(ctx, email)
	if err != nil {
		return nil, err
	}
	if len(orgs) > 0 {
		return &orgs[0], nil
	}

	// Create a default org named after the user's email prefix
	orgName := email
	if idx := strings.Index(email, "@"); idx > 0 {
		orgName = email[:idx]
	}

	return s.Create(ctx, orgName, email)
}

// sanitizeID converts a name to a valid K8s resource name fragment.
func sanitizeID(name string) string {
	id := strings.ToLower(name)
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, id)
	// Trim leading/trailing dashes
	id = strings.Trim(id, "-")
	if len(id) > 40 {
		id = id[:40]
	}
	if id == "" {
		id = "default"
	}
	return id
}
