// CSS selectors for MicroFoundry admin pages
// Centralized for easy maintenance when templates change

export const nav = {
  sidebar: 'aside',
  brand: 'aside h1',
  operationsLabel: 'text=Operations',
  settingsLabel: 'text=Settings',
  // Operations links
  dashboard: 'a[href="/"]',
  apps: 'a[href="/apps"]',
  services: 'a[href="/services"]',
  secrets: 'a[href="/secrets"]',
  monitoring: 'a[href="/monitoring"]',
  // Settings links
  clusters: 'a[href="/clusters"]',
  catalog: 'a[href="/catalog"]',
  registry: 'a[href="/settings/registry"]',
  webhooks: 'a[href="/settings/webhooks"]',
  smtp: 'a[href="/settings/smtp"]',
  endpoints: 'a[href="/settings/endpoints"]',
  users: 'a[href="/users"]',
  config: 'a[href="/config"]',
  activeClass: 'bg-gray-800',
};

export const dashboard = {
  statsGrid: '.grid.grid-cols-1',
  appCountCard: 'a[href="/apps"] .text-2xl',
  domainCard: 'text=Domain',
  namespaceCard: 'text=Namespace',
  contextCard: 'text=K8s Context',
  quickLinks: 'text=Quick Links',
  viewAppsLink: 'text=View Applications',
  platformConfigLink: 'text=Platform Configuration',
  backingServicesLink: 'text=Backing Services',
};

export const apps = {
  title: 'text=Deployed Applications',
  stateFilter: '#state-filter',
  refreshButton: 'button:has-text("Refresh")',
  tableBody: '#app-table-body',
  emptyState: 'text=No applications deployed',
  // Table headers
  headerName: 'th:has-text("Name")',
  headerState: 'th:has-text("State")',
  headerInstances: 'th:has-text("Instances")',
  headerMemory: 'th:has-text("Memory")',
  headerRoutes: 'th:has-text("Routes")',
  headerCreated: 'th:has-text("Created")',
  headerActions: 'th:has-text("Actions")',
};

export const appDetail = {
  backButton: 'a[href="/apps"]',
  appName: 'h3.text-2xl',
  stateBadge: 'span.inline-flex',
  // Tabs
  overviewTab: 'text=Overview',
  instancesTab: 'text=Instances',
  configTab: 'text=Config',
  servicesTab: 'text=Services',
  routesTab: 'text=Routes',
  logsTab: 'text=Logs',
  metricsTab: 'text=Metrics',
  performanceTab: 'text=Performance',
  tabContent: '#tab-content',
};

export const services = {
  title: 'text=Provisioned Services',
  createButton: 'button:has-text("Create Service")',
  createModal: '#create-svc-modal',
  serviceTypeSelect: '#svc-type',
  planSelect: '#svc-plan',
  serviceNameInput: '#svc-name',
  planInfo: '#plan-info',
  tableBody: '#service-table-body',
  emptyState: 'text=No services provisioned',
};

export const serviceDetail = {
  breadcrumb: 'nav.text-sm',
  serviceName: 'h2.text-2xl',
  statusBadge: 'span.inline-flex',
  connectionDetails: 'text=Connection Details',
  vcapSection: 'text=VCAP_SERVICES',
  bindForm: 'input[name="app_name"]',
};

export const catalog = {
  title: 'header h2',
  terraformBadge: '.bg-green-100, .bg-yellow-100',
  uploadTopology: 'text=Upload Topology',
  // Categories
  databases: 'h3:has-text("Databases")',
  caches: 'h3:has-text("Caches")',
  messaging: 'h3:has-text("Messaging")',
  storage: 'h3:has-text("Storage")',
  gateway: 'h3:has-text("Gateways")',
};

export const secrets = {
  title: 'header h2',
  createButton: 'a[href="/secrets/new"]',
  secretList: '#secret-list',
  filterAll: 'a:has-text("All")',
  filterService: 'a:has-text("Service")',
  filterUser: 'a:has-text("User-defined")',
};

export const secretDetail = {
  secretName: 'h2.text-2xl',
  typeBadge: 'span.inline-flex',
  revealButton: '[hx-get*="reveal"]',
  maskedValue: 'text=********',
  boundApps: 'text=Bound Applications',
  metadata: 'text=Metadata',
};

export const clusters = {
  title: 'text=Kubernetes Clusters',
  addButton: 'button:has-text("Add Cluster")',
  addForm: '#add-cluster-form',
  nameInput: '#cluster-name',
  providerInput: '#cluster-provider',
  contextInput: '#cluster-context',
  namespaceInput: '#cluster-namespace',
  domainInput: '#cluster-domain',
  regionInput: '#cluster-region',
  ingressInput: '#cluster-ingress',
  gatewayInfo: '#gateway-info',
  activeClusterRow: '.bg-blue-50',
  activeBadge: '.bg-blue-100',
};

export const settingsRegistry = {
  title: 'text=Container Registry',
  urlInput: '#reg-url',
  projectInput: '#reg-project',
  usernameInput: '#reg-username',
  passwordInput: '#reg-password',
  insecureCheckbox: 'input[name="insecure"]',
  enabledCheckbox: 'input[name="enabled"]',
  testButton: 'button:has-text("Test Connection")',
  testResult: '#test-result',
  saveButton: 'button:has-text("Save")',
  successBanner: '.bg-green-50',
};

export const settingsWebhooks = {
  title: 'header h2',
  addButton: 'button:has-text("Add Webhook")',
  addForm: '#add-webhook-form',
  nameInput: '#wh-name',
  urlInput: '#wh-url',
  eventsCheckboxes: 'input[name="events"]',
  enabledCheckbox: 'input[name="enabled"]',
};

export const settingsSMTP = {
  title: 'header h2',
  hostInput: '#smtp-host',
  portInput: '#smtp-port',
  usernameInput: '#smtp-username',
  passwordInput: '#smtp-password',
  fromInput: '#smtp-from',
  tlsCheckbox: 'input[name="tls"]',
  enabledCheckbox: 'input[name="enabled"]',
  testButton: 'button:has-text("Test Connection")',
  testResult: '#test-result',
  saveButton: 'button:has-text("Save")',
};

export const settingsEndpoints = {
  title: 'text=Service Endpoints',
  saveButton: 'button:has-text("Save")',
  successBanner: '.bg-green-50',
  testButton: 'button:has-text("Test")',
  createIngressButton: 'button:has-text("Create Ingress")',
};

export const monitoring = {
  title: 'text=Metrics & Alerts',
  grafanaButton: 'text=Open Grafana',
  componentHealth: '.grid.grid-cols-2',
  alertsContainer: '#alerts-container',
  grafanaIframe: 'iframe[src*="grafana"]',
};

export const users = {
  title: 'text=Users & Organizations',
  noAuthMessage: 'text=authentication',
  // IAM tabs
  workspacesTab: 'a:has-text("Workspaces")',
  orgsTab: 'a:has-text("Organizations")',
  usersTab: 'a:has-text("Users")',
  policiesTab: 'a:has-text("Policies")',
  auditTab: 'a:has-text("Audit Log")',
  tabContent: '#tab-content',
  // Org management
  createOrgForm: '#create-org-form',
  createOrgButton: 'button:has-text("Create")',
  // User management
  createUserForm: '#create-user-form',
  createUserButton: 'button:has-text("Create User")',
};

export const topology = {
  uploadForm: 'text=Upload Topology',
  serviceTypeSelect: 'select[name="service_type"]',
  planSelect: 'select[name="plan_id"]',
  editor: 'textarea[name*="file_content"]',
  previewButton: 'button:has-text("Preview")',
  saveButton: 'button:has-text("Save")',
};

export const config = {
  kubernetesSection: 'h3:has-text("Kubernetes")',
  githubSection: 'h3:has-text("GitHub")',
  platformSection: 'h3:has-text("Platform")',
};

export const docsPage = {
  manualTab: 'a[href="/docs?tab=manual"]',
  adminTab: 'a[href="/docs?tab=admin"]',
  architectureTab: 'a[href="/docs?tab=architecture"]',
  content: '.docs-content',
  navLink: 'a[href="/docs"]',
};
