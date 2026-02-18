package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	wsLabelKey   = "microfoundry.io/type"
	wsLabelValue = "workspace"
	wsPrefix     = "mf-ws-"
)

// Workspace represents a company-level grouping that contains organizations.
type Workspace struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	OwnerEmail string `json:"owner_email"`
	CreatedAt  string `json:"created_at"`
}

// WorkspaceMember represents a user's membership in a workspace.
type WorkspaceMember struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"` // "admin", "member", "viewer"
	JoinedAt string `json:"joined_at"`
}

// WorkspaceStore persists workspaces as ConfigMaps in the platform namespace.
type WorkspaceStore struct {
	clientset kubernetes.Interface
	namespace string
}

// NewWorkspaceStore creates a workspace store backed by K8s ConfigMaps.
func NewWorkspaceStore(clientset kubernetes.Interface, namespace string) *WorkspaceStore {
	return &WorkspaceStore{clientset: clientset, namespace: namespace}
}

// List returns all workspaces.
func (s *WorkspaceStore) List(ctx context.Context) ([]Workspace, error) {
	cms, err := s.clientset.CoreV1().ConfigMaps(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: wsLabelKey + "=" + wsLabelValue,
	})
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}

	var workspaces []Workspace
	for _, cm := range cms.Items {
		if members := cm.Data["members"]; members != "" {
			continue // skip member ConfigMaps
		}
		var ws Workspace
		if data := cm.Data["metadata"]; data != "" {
			if err := json.Unmarshal([]byte(data), &ws); err != nil {
				continue
			}
			workspaces = append(workspaces, ws)
		}
	}
	return workspaces, nil
}

// Get returns a single workspace by ID.
func (s *WorkspaceStore) Get(ctx context.Context, wsID string) (*Workspace, error) {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, wsPrefix+wsID, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting workspace %q: %w", wsID, err)
	}

	var ws Workspace
	if err := json.Unmarshal([]byte(cm.Data["metadata"]), &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

// Create creates a new workspace.
func (s *WorkspaceStore) Create(ctx context.Context, name, ownerEmail string) (*Workspace, error) {
	id := sanitizeID(name)

	ws := Workspace{
		ID:         id,
		Name:       name,
		OwnerEmail: ownerEmail,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	wsData, _ := json.Marshal(ws)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wsPrefix + id,
			Namespace: s.namespace,
			Labels:    map[string]string{wsLabelKey: wsLabelValue},
		},
		Data: map[string]string{
			"metadata": string(wsData),
		},
	}

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("workspace %q already exists", name)
		}
		return nil, fmt.Errorf("creating workspace: %w", err)
	}

	// Create members ConfigMap with owner
	owner := WorkspaceMember{
		Email:    ownerEmail,
		Name:     ownerEmail,
		Role:     "admin",
		JoinedAt: ws.CreatedAt,
	}
	membersData, _ := json.Marshal([]WorkspaceMember{owner})

	membersCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wsPrefix + id + "-members",
			Namespace: s.namespace,
			Labels:    map[string]string{wsLabelKey: wsLabelValue},
		},
		Data: map[string]string{
			"members": string(membersData),
		},
	}
	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Create(ctx, membersCM, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("creating workspace members: %w", err)
	}

	return &ws, nil
}

// Delete removes a workspace and its member list.
func (s *WorkspaceStore) Delete(ctx context.Context, wsID string) error {
	_ = s.clientset.CoreV1().ConfigMaps(s.namespace).Delete(ctx, wsPrefix+wsID+"-members", metav1.DeleteOptions{})

	if err := s.clientset.CoreV1().ConfigMaps(s.namespace).Delete(ctx, wsPrefix+wsID, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}
	return nil
}

// ListMembers returns all members of a workspace.
func (s *WorkspaceStore) ListMembers(ctx context.Context, wsID string) ([]WorkspaceMember, error) {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, wsPrefix+wsID+"-members", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting workspace members: %w", err)
	}

	var members []WorkspaceMember
	if err := json.Unmarshal([]byte(cm.Data["members"]), &members); err != nil {
		return nil, err
	}
	return members, nil
}

// AddMember adds a user to a workspace.
func (s *WorkspaceStore) AddMember(ctx context.Context, wsID, email, name, role string) error {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, wsPrefix+wsID+"-members", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting workspace members: %w", err)
	}

	var members []WorkspaceMember
	if err := json.Unmarshal([]byte(cm.Data["members"]), &members); err != nil {
		return err
	}

	for _, m := range members {
		if m.Email == email {
			return fmt.Errorf("user %q is already a workspace member", email)
		}
	}

	members = append(members, WorkspaceMember{
		Email:    email,
		Name:     name,
		Role:     role,
		JoinedAt: time.Now().UTC().Format(time.RFC3339),
	})

	data, _ := json.Marshal(members)
	cm.Data["members"] = string(data)

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("updating workspace members: %w", err)
	}
	return nil
}

// RemoveMember removes a user from a workspace.
func (s *WorkspaceStore) RemoveMember(ctx context.Context, wsID, email string) error {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, wsPrefix+wsID+"-members", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting workspace members: %w", err)
	}

	var members []WorkspaceMember
	if err := json.Unmarshal([]byte(cm.Data["members"]), &members); err != nil {
		return err
	}

	var updated []WorkspaceMember
	for _, m := range members {
		if m.Email != email {
			updated = append(updated, m)
		}
	}

	if len(updated) == len(members) {
		return fmt.Errorf("user %q is not a workspace member", email)
	}

	data, _ := json.Marshal(updated)
	cm.Data["members"] = string(data)

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("updating workspace members: %w", err)
	}
	return nil
}

// SetMemberRole changes a workspace member's role.
func (s *WorkspaceStore) SetMemberRole(ctx context.Context, wsID, email, role string) error {
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, wsPrefix+wsID+"-members", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting workspace members: %w", err)
	}

	var members []WorkspaceMember
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
		return fmt.Errorf("user %q is not a workspace member", email)
	}

	data, _ := json.Marshal(members)
	cm.Data["members"] = string(data)

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("updating workspace members: %w", err)
	}
	return nil
}

// GetUserWorkspaces returns all workspaces a user belongs to.
func (s *WorkspaceStore) GetUserWorkspaces(ctx context.Context, email string) ([]Workspace, error) {
	allWS, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var userWS []Workspace
	for _, ws := range allWS {
		members, err := s.ListMembers(ctx, ws.ID)
		if err != nil {
			continue
		}
		for _, m := range members {
			if m.Email == email {
				userWS = append(userWS, ws)
				break
			}
		}
	}
	return userWS, nil
}

// EnsureDefaultWorkspace creates a default workspace for a user if they have none.
func (s *WorkspaceStore) EnsureDefaultWorkspace(ctx context.Context, email, name string) (*Workspace, error) {
	workspaces, err := s.GetUserWorkspaces(ctx, email)
	if err != nil {
		return nil, err
	}
	if len(workspaces) > 0 {
		return &workspaces[0], nil
	}

	wsName := "default"
	if name != "" {
		wsName = name
	}

	return s.Create(ctx, wsName, email)
}
