import { APIRequestContext } from '@playwright/test';

export class APIHelper {
  constructor(private request: APIRequestContext) {}

  // Applications
  async getApps() {
    const res = await this.request.get('/api/apps');
    return res.json();
  }

  async getApp(name: string) {
    const res = await this.request.get(`/api/apps/${name}`);
    return res.json();
  }

  async getAppRedMetrics(name: string) {
    const res = await this.request.get(`/api/apps/${name}/red-metrics`);
    return res.json();
  }

  async getAppHealth(name: string) {
    const res = await this.request.get(`/api/apps/${name}/health`);
    return res.json();
  }

  async getAppObservability(name: string) {
    const res = await this.request.get(`/api/apps/${name}/observability`);
    return res.json();
  }

  // Services
  async getServices() {
    const res = await this.request.get('/api/services');
    return res.json();
  }

  async getService(name: string) {
    const res = await this.request.get(`/api/services/${name}`);
    return res.json();
  }

  // Secrets
  async getSecrets() {
    const res = await this.request.get('/api/secrets');
    return res.json();
  }

  async getSecret(name: string) {
    const res = await this.request.get(`/api/secrets/${name}`);
    return res.json();
  }

  // Clusters
  async getClusters() {
    const res = await this.request.get('/api/clusters');
    return res.json();
  }

  async getClusterHealth(id: string) {
    const res = await this.request.get(`/api/clusters/${id}/health`);
    return res.json();
  }

  // Catalog
  async getCatalog() {
    const res = await this.request.get('/api/catalog');
    return res.json();
  }

  async getVisibleCatalog() {
    const res = await this.request.get('/api/catalog/visible');
    return res.json();
  }

  // Settings
  async getSettings() {
    const res = await this.request.get('/api/settings');
    return res.json();
  }

  async getWebhooks() {
    const res = await this.request.get('/api/settings/webhooks');
    return res.json();
  }

  async getEndpoints() {
    const res = await this.request.get('/api/settings/endpoints');
    return res.json();
  }

  // Monitoring
  async getAlerts() {
    const res = await this.request.get('/api/monitoring/alerts');
    return res.json();
  }

  async getMonitoringHealth() {
    const res = await this.request.get('/api/monitoring/health');
    return res.json();
  }

  // Organizations
  async getOrgs() {
    const res = await this.request.get('/api/orgs');
    return res.json();
  }

  async getOrg(id: string) {
    const res = await this.request.get(`/api/orgs/${id}`);
    return res.json();
  }

  // Workspaces
  async getWorkspaces() {
    const res = await this.request.get('/api/workspaces');
    return res.json();
  }

  async getWorkspace(id: string) {
    const res = await this.request.get(`/api/workspaces/${id}`);
    return res.json();
  }

  // IAM
  async getUsers() {
    const res = await this.request.get('/api/users');
    return res.json();
  }

  async getPolicies() {
    const res = await this.request.get('/api/policies');
    return res.json();
  }

  async getAudit() {
    const res = await this.request.get('/api/audit');
    return res.json();
  }

  // Config
  async getConfig() {
    const res = await this.request.get('/api/config');
    return res.json();
  }

  // Topologies
  async getTopologies() {
    const res = await this.request.get('/api/topologies');
    return res.json();
  }

  async getTopology(type: string, plan: string) {
    const res = await this.request.get(`/api/topologies/${type}/${plan}`);
    return res.json();
  }
}
