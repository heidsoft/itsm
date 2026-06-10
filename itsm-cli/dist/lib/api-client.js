import { loadCredentials } from './credentials.js';
const DEFAULT_BASE_URL = 'http://localhost:8090';
async function request(baseURL, method, endpoint, data, params) {
    const cred = loadCredentials();
    const url = new URL(`${baseURL}/api/v1${endpoint}`);
    if (params) {
        Object.entries(params).forEach(([k, v]) => {
            if (v !== undefined)
                url.searchParams.set(k, String(v));
        });
    }
    const headers = { 'Content-Type': 'application/json' };
    if (cred?.token)
        headers['Authorization'] = `Bearer ${cred.token}`;
    const res = await fetch(url.toString(), {
        method,
        headers,
        body: data ? JSON.stringify(data) : undefined,
    });
    if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`);
    }
    const json = await res.json();
    if (json.code !== 0) {
        throw new Error(json.message || 'API error');
    }
    return json.data;
}
export class ApiClient {
    constructor(baseURL = DEFAULT_BASE_URL) {
        this.baseURL = baseURL;
    }
    // ---------- Auth ----------
    async login(loginData) {
        const url = `${this.baseURL}/api/v1/auth/login`;
        const res = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(loginData),
        });
        if (!res.ok)
            throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        if (json.code !== 0)
            throw new Error(json.message || 'Login failed');
        return json.data;
    }
    async logout() { await request(this.baseURL, 'POST', '/auth/logout'); }
    async getUserInfo() { return request(this.baseURL, 'GET', '/auth/userinfo'); }
    // ---------- Tickets ----------
    async listTickets(params) {
        return request(this.baseURL, 'GET', '/tickets', undefined, params);
    }
    async getTicket(id) {
        return request(this.baseURL, 'GET', `/tickets/${id}`);
    }
    async createTicket(data) {
        return request(this.baseURL, 'POST', '/tickets', data);
    }
    async searchTickets(query, params) {
        return request(this.baseURL, 'GET', '/tickets/search', undefined, { ...params, search: query });
    }
    async getOverdueTickets() {
        return request(this.baseURL, 'GET', '/tickets/overdue');
    }
    async globalSearch(q, page) {
        return request(this.baseURL, 'GET', '/search', undefined, { q, page });
    }
    // ---------- Incidents ----------
    async listIncidents(params) {
        return request(this.baseURL, 'GET', '/incidents', undefined, params);
    }
    async getIncident(id) {
        return request(this.baseURL, 'GET', `/incidents/${id}`);
    }
    async aiTriage(payload) {
        return request(this.baseURL, 'POST', '/ai/triage', payload);
    }
    // ---------- Changes ----------
    async listChanges(params) {
        return request(this.baseURL, 'GET', '/changes', undefined, params);
    }
    async getChange(id) {
        return request(this.baseURL, 'GET', `/changes/${id}`);
    }
    // ---------- CMDB ----------
    async listCIs(params) {
        return request(this.baseURL, 'GET', '/cmdb/cis', undefined, params);
    }
    async getCI(id) {
        return request(this.baseURL, 'GET', `/cmdb/cis/${id}`);
    }
    // ---------- Knowledge ----------
    async listKnowledge(params) {
        return request(this.baseURL, 'GET', '/knowledge/articles', undefined, params);
    }
    async searchKnowledge(q) {
        return request(this.baseURL, 'GET', '/knowledge/articles', undefined, { search: q, pageSize: 20 });
    }
    async getKnowledgeArticle(id) {
        return request(this.baseURL, 'GET', `/knowledge/articles/${id}`);
    }
    // ---------- SLA ----------
    async getSLAStats() {
        return request(this.baseURL, 'GET', '/sla/stats');
    }
    async getSLAOverdue() {
        return request(this.baseURL, 'GET', '/sla/overdue');
    }
    // ---------- Workflow ----------
    async listProcessInstances(params) {
        return request(this.baseURL, 'GET', '/workflow/instances', undefined, params);
    }
    async getProcessInstance(id) {
        return request(this.baseURL, 'GET', `/workflow/instances/${id}`);
    }
    async listUserTasks() {
        return request(this.baseURL, 'GET', '/workflow/tasks');
    }
    async completeTask(id, outcome, comment) {
        return request(this.baseURL, 'POST', `/workflow/tasks/${id}/complete`, { outcome, comment });
    }
    // ---------- Approvals ----------
    async listApprovals(status) {
        return request(this.baseURL, 'GET', '/approvals', undefined, { status });
    }
    async approveTask(id, comment) {
        return request(this.baseURL, 'POST', `/approvals/${id}/approve`, { comment });
    }
    async rejectTask(id, comment) {
        return request(this.baseURL, 'POST', `/approvals/${id}/reject`, { comment });
    }
    // ---------- Notifications ----------
    async listNotifications(params) {
        return request(this.baseURL, 'GET', '/notifications', undefined, params);
    }
    async markNotificationRead(id) {
        return request(this.baseURL, 'POST', `/notifications/${id}/read`);
    }
    // ---------- Connectors / IM / Plugin market ----------
    async listConnectors() {
        return request(this.baseURL, 'GET', '/connectors');
    }
    async listConnectorConfigs() {
        return request(this.baseURL, 'GET', '/connectors/configs');
    }
    async provisionConnector(cfg) {
        return request(this.baseURL, 'POST', '/connectors/configs', cfg);
    }
    async testConnector(name) {
        return request(this.baseURL, 'POST', `/connectors/${name}/test`);
    }
    async sendViaConnector(name, payload) {
        return request(this.baseURL, 'POST', `/connectors/${name}/send`, payload);
    }
    async connectorHealth() {
        return request(this.baseURL, 'GET', '/connectors/health');
    }
    // ---------- Dashboard ----------
    async getDashboardOverview() {
        return request(this.baseURL, 'GET', '/dashboard/overview');
    }
}
export const apiClient = new ApiClient();
