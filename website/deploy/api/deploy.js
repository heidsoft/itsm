/**
 * OpenClaw 部署管理 API
 */

const API_BASE = '/deploy/api';

// 部署服务类
class DeployService {
    // 获取所有部署实例
    async getDeployments() {
        const response = await fetch(`${API_BASE}/deployments`);
        return await response.json();
    }

    // 创建新部署
    async createDeployment(config) {
        const response = await fetch(`${API_BASE}/deployments`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        return await response.json();
    }

    // 启动部署
    async startDeployment(id) {
        const response = await fetch(`${API_BASE}/deployments/${id}/start`, {
            method: 'POST'
        });
        return await response.json();
    }

    // 停止部署
    async stopDeployment(id) {
        const response = await fetch(`${API_BASE}/deployments/${id}/stop`, {
            method: 'POST'
        });
        return await response.json();
    }

    // 删除部署
    async deleteDeployment(id) {
        const response = await fetch(`${API_BASE}/deployments/${id}`, {
            method: 'DELETE'
        });
        return await response.json();
    }

    // 获取监控数据
    async getMetrics(id) {
        const response = await fetch(`${API_BASE}/deployments/${id}/metrics`);
        return await response.json();
    }

    // 获取日志
    async getLogs(id, options = {}) {
        const params = new URLSearchParams(options);
        const response = await fetch(`${API_BASE}/deployments/${id}/logs?${params}`);
        return await response.json();
    }

    // 更新配置
    async updateConfig(id, config) {
        const response = await fetch(`${API_BASE}/deployments/${id}/config`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        return await response.json();
    }
}

// 监控服务类
class MonitorService {
    // 获取系统状态
    async getSystemStatus() {
        const response = await fetch(`${API_BASE}/monitor/system`);
        return await response.json();
    }

    // 获取告警列表
    async getAlerts() {
        const response = await fetch(`${API_BASE}/monitor/alerts`);
        return await response.json();
    }

    // 确认告警
    async acknowledgeAlert(id) {
        const response = await fetch(`${API_BASE}/monitor/alerts/${id}/ack`, {
            method: 'POST'
        });
        return await response.json();
    }
}

// 用户服务类
class UserService {
    // 获取当前用户
    async getCurrentUser() {
        const response = await fetch(`${API_BASE}/user/me`);
        return await response.json();
    }

    // 获取用户列表
    async getUsers() {
        const response = await fetch(`${API_BASE}/users`);
        return await response.json();
    }

    // 更新用户信息
    async updateUser(id, data) {
        const response = await fetch(`${API_BASE}/users/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return await response.json();
    }
}

// 导出服务实例
window.deployService = new DeployService();
window.monitorService = new MonitorService();
window.userService = new UserService();

console.log('OpenClaw Deploy API initialized');
