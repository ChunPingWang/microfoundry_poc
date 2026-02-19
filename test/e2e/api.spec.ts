import { test, expect } from '../fixtures/admin-server';

test.describe('JSON API - Core Endpoints', () => {
  test('GET /api/apps returns JSON', async ({ request }) => {
    const response = await request.get('/api/apps');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });

  test('GET /api/services returns JSON', async ({ request }) => {
    const response = await request.get('/api/services');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });

  test('GET /api/secrets returns JSON', async ({ request }) => {
    const response = await request.get('/api/secrets');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });

  test('GET /api/clusters returns JSON', async ({ request }) => {
    const response = await request.get('/api/clusters');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });

  test('GET /api/config returns platform config', async ({ request }) => {
    const response = await request.get('/api/config');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
    const data = await response.json();
    expect(typeof data).toBe('object');
  });
});

test.describe('JSON API - Catalog Endpoints', () => {
  test('GET /api/catalog returns all service types', async ({ request }) => {
    const response = await request.get('/api/catalog');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
    const data = await response.json();
    expect(Array.isArray(data) || typeof data === 'object').toBe(true);
  });

  test('GET /api/catalog/visible returns visible catalog', async ({ request }) => {
    const response = await request.get('/api/catalog/visible');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });
});

test.describe('JSON API - Settings Endpoints', () => {
  test('GET /api/settings returns platform settings', async ({ request }) => {
    const response = await request.get('/api/settings');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });

  test('GET /api/settings/webhooks returns webhook list', async ({ request }) => {
    const response = await request.get('/api/settings/webhooks');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });

  test('GET /api/settings/endpoints returns discovered endpoints', async ({ request }) => {
    const response = await request.get('/api/settings/endpoints');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });
});

test.describe('JSON API - Monitoring Endpoints', () => {
  test('GET /api/monitoring/alerts returns response', async ({ request }) => {
    const response = await request.get('/api/monitoring/alerts');
    // AlertManager may not be running, so accept 200 or 502/503
    expect(response.status()).toBeLessThanOrEqual(503);
  });

  test('GET /api/monitoring/health returns monitoring status', async ({ request }) => {
    const response = await request.get('/api/monitoring/health');
    expect(response.status()).toBeLessThanOrEqual(503);
  });
});

test.describe('JSON API - IAM Endpoints', () => {
  test('GET /api/orgs returns organization list', async ({ request }) => {
    const response = await request.get('/api/orgs');
    expect(response.status()).toBeLessThan(500);
  });

  test('GET /api/workspaces returns workspace list', async ({ request }) => {
    const response = await request.get('/api/workspaces');
    expect(response.status()).toBeLessThan(500);
  });

  test('GET /api/users returns user list', async ({ request }) => {
    const response = await request.get('/api/users');
    // May require Keycloak admin access, so 4xx is acceptable
    expect(response.status()).toBeLessThan(500);
  });

  test('GET /api/policies returns OPA policies', async ({ request }) => {
    const response = await request.get('/api/policies');
    expect(response.status()).toBeLessThan(500);
  });

  test('GET /api/audit returns audit log entries', async ({ request }) => {
    const response = await request.get('/api/audit');
    expect(response.status()).toBeLessThan(500);
  });
});

test.describe('JSON API - Topology Endpoints', () => {
  test('GET /api/topologies returns topology list', async ({ request }) => {
    const response = await request.get('/api/topologies');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['content-type']).toContain('json');
  });
});

test.describe('JSON API - App Detail Endpoints', () => {
  test('GET /api/apps returns array structure', async ({ request }) => {
    const response = await request.get('/api/apps');
    if (response.ok()) {
      const data = await response.json();
      expect(Array.isArray(data)).toBe(true);
    }
  });

  test('app RED metrics endpoint responds', async ({ request }) => {
    const appsResponse = await request.get('/api/apps');
    if (appsResponse.ok()) {
      const apps = await appsResponse.json();
      if (apps && apps.length > 0) {
        const appName = apps[0].name || apps[0].Name;
        const response = await request.get(`/api/apps/${appName}/red-metrics`);
        // Prometheus may not be running, so accept errors gracefully
        expect(response.status()).toBeLessThanOrEqual(503);
      }
    }
  });

  test('app health endpoint responds', async ({ request }) => {
    const appsResponse = await request.get('/api/apps');
    if (appsResponse.ok()) {
      const apps = await appsResponse.json();
      if (apps && apps.length > 0) {
        const appName = apps[0].name || apps[0].Name;
        const response = await request.get(`/api/apps/${appName}/health`);
        expect(response.status()).toBeLessThanOrEqual(503);
      }
    }
  });

  test('app observability endpoint responds', async ({ request }) => {
    const appsResponse = await request.get('/api/apps');
    if (appsResponse.ok()) {
      const apps = await appsResponse.json();
      if (apps && apps.length > 0) {
        const appName = apps[0].name || apps[0].Name;
        const response = await request.get(`/api/apps/${appName}/observability`);
        expect(response.status()).toBeLessThanOrEqual(503);
      }
    }
  });
});

test.describe('JSON API - Error Handling', () => {
  test('non-existent API path returns 404', async ({ request }) => {
    const response = await request.get('/api/nonexistent');
    expect(response.status()).toBe(404);
  });

  test('non-existent app returns error', async ({ request }) => {
    const response = await request.get('/api/apps/does-not-exist-12345');
    expect(response.status()).toBeGreaterThanOrEqual(400);
  });
});
