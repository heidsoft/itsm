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
    async logout() {
        await request(this.baseURL, 'POST', '/auth/logout');
    }
    async getUserInfo() {
        return request(this.baseURL, 'GET', '/auth/userinfo');
    }
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
    async getSLAStats() {
        return request(this.baseURL, 'GET', '/sla/stats');
    }
    async getOverdueTickets() {
        return request(this.baseURL, 'GET', '/tickets/overdue');
    }
}
export const apiClient = new ApiClient();
