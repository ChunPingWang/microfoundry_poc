package models

import "time"

type AppState string

const (
	AppStateStarted   AppState = "STARTED"
	AppStateStopped   AppState = "STOPPED"
	AppStateUploading AppState = "UPLOADING"
	AppStateStaging   AppState = "STAGING"
	AppStateDeploying AppState = "DEPLOYING"
	AppStateFailed    AppState = "FAILED"
)

type LifecycleType string

const (
	LifecycleBuildpack LifecycleType = "buildpack"
	LifecycleDocker    LifecycleType = "docker"
	LifecycleCNB       LifecycleType = "cnb"
)

// App is the central entity mapping CF Application + Process + Droplet into one struct.
type App struct {
	GUID          string            `json:"guid"`
	Name          string            `json:"name"`
	State         AppState          `json:"state"`
	LifecycleType LifecycleType     `json:"lifecycle_type"`
	Buildpacks    []string          `json:"buildpacks,omitempty"`
	DockerImage   string            `json:"docker_image,omitempty"`
	Command       string            `json:"command,omitempty"`
	Instances     int               `json:"instances"`
	MemoryMB      int               `json:"memory_mb"`
	DiskMB        int               `json:"disk_mb"`
	Port          int               `json:"port"`
	Env           map[string]string `json:"env,omitempty"`
	ImageRef      string            `json:"image_ref,omitempty"`
	HealthCheck   HealthCheck       `json:"health_check"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// AppDefaults returns an App with sensible defaults.
func AppDefaults(name string) App {
	return App{
		Name:          name,
		State:         AppStateStopped,
		LifecycleType: LifecycleBuildpack,
		Instances:     1,
		MemoryMB:      256,
		DiskMB:        1024,
		Port:          8080,
		Env:           make(map[string]string),
		HealthCheck:   HealthCheck{Type: HealthCheckPort},
	}
}

type AppStatus struct {
	App
	Routes       []Route          `json:"routes"`
	RunningCount int              `json:"running_count"`
	TotalCount   int              `json:"total_count"`
	Instances    []InstanceStatus `json:"instances,omitempty"`
}

type InstanceStatus struct {
	Index        int       `json:"index"`
	State        string    `json:"state"`
	Since        time.Time `json:"since"`
	CPU          float64   `json:"cpu"`
	MemoryMB     int       `json:"memory_mb"`
	DiskMB       int       `json:"disk_mb"`
	RestartCount int       `json:"restart_count"`
	NodeName     string    `json:"node_name,omitempty"`
	PodName      string    `json:"pod_name,omitempty"`
	ImageID      string    `json:"image_id,omitempty"`
}

// AppDetail is the comprehensive view of an app for the admin interface.
type AppDetail struct {
	Name          string            `json:"name"`
	GUID          string            `json:"guid,omitempty"`
	State         string            `json:"state"`
	Organization  string            `json:"organization"`
	Owner         string            `json:"owner,omitempty"`
	LifecycleType string            `json:"lifecycle_type"`
	Buildpacks    []string          `json:"buildpacks,omitempty"`
	ImageRef      string            `json:"image_ref,omitempty"`
	ImageDigest   string            `json:"image_digest,omitempty"`
	Command       string            `json:"command,omitempty"`
	Instances     int               `json:"instances"`
	RunningCount  int               `json:"running_count"`
	MemoryMB      int               `json:"memory_mb"`
	DiskMB        int               `json:"disk_mb"`
	CPUMillis     int               `json:"cpu_millis"`
	Port          int               `json:"port"`
	HealthCheck   HealthCheck       `json:"health_check"`
	Routes        []RouteDetail     `json:"routes"`
	Env           map[string]string `json:"env,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Services      []ServiceBindingInfo `json:"services,omitempty"`
	Secrets       []SecretInfo         `json:"secrets,omitempty"`
	InstanceList  []InstanceStatus  `json:"instance_list,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// RouteDetail is the full routing info for display.
type RouteDetail struct {
	Host     string `json:"host"`
	Domain   string `json:"domain"`
	Path     string `json:"path,omitempty"`
	URL      string `json:"url"`
	Protocol string `json:"protocol"`
}

// AppListItem is the denormalized view for the app list table.
type AppListItem struct {
	Name          string   `json:"name"`
	Organization  string   `json:"organization"`
	Owner         string   `json:"owner"`
	State         string   `json:"state"`
	LifecycleType string   `json:"lifecycle_type"`
	RunningCount  int      `json:"running_count"`
	TotalCount    int      `json:"total_count"`
	MemoryMB      int      `json:"memory_mb"`
	Routes        []string `json:"routes"`
	CreatedAt     string   `json:"created_at"`
}
