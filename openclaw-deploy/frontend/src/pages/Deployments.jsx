import React, { useState, useEffect } from 'react';
import { deployService } from '../services/deploy';
import './Deployments.css';

function Deployments() {
  const [deployments, setDeployments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  useEffect(() => {
    loadDeployments();
  }, []);

  const loadDeployments = async () => {
    try {
      const data = await deployService.getDeployments();
      setDeployments(data);
    } catch (error) {
      console.error('Failed to load deployments:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleStart = async (id) => {
    try {
      await deployService.startDeployment(id);
      loadDeployments();
    } catch (error) {
      alert('启动失败：' + error.message);
    }
  };

  const handleStop = async (id) => {
    try {
      await deployService.stopDeployment(id);
      loadDeployments();
    } catch (error) {
      alert('停止失败：' + error.message);
    }
  };

  const getStatusBadge = (status) => {
    const statusMap = {
      running: { class: 'status-running', text: '运行中' },
      deploying: { class: 'status-deploying', text: '部署中' },
      stopped: { class: 'status-stopped', text: '已停止' },
      error: { class: 'status-error', text: '错误' }
    };
    const statusInfo = statusMap[status] || statusMap.stopped;
    return (
      <span className={`status-badge ${statusInfo.class}`}>
        {statusInfo.text}
      </span>
    );
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="deployments-page">
      <div className="page-header">
        <h1>部署实例</h1>
        <button 
          className="btn btn-primary"
          onClick={() => setShowCreateModal(true)}
        >
          <i className="fas fa-plus"></i> 新建部署
        </button>
      </div>

      <div className="deployments-list">
        {deployments.map(deployment => (
          <div key={deployment.id} className="deployment-card">
            <div className="deployment-header">
              <div className="deployment-info">
                <h3>
                  <i className="fas fa-server"></i>
                  {deployment.instance_name}
                </h3>
                <small>ID: {deployment.id}</small>
              </div>
              {getStatusBadge(deployment.status)}
            </div>

            <div className="deployment-metrics">
              <div className="metric">
                <strong>CPU: {deployment.metrics?.cpu_usage || 0}%</strong>
                <small>内存：{deployment.metrics?.memory_usage || 0}GB</small>
              </div>
              <div className="metric">
                <strong>QPS: {deployment.metrics?.qps || 0}</strong>
                <small>响应：{deployment.metrics?.response_time || 0}ms</small>
              </div>
            </div>

            <div className="deployment-actions">
              <button className="btn btn-sm btn-outline-primary">
                <i className="fas fa-eye"></i> 监控
              </button>
              <button className="btn btn-sm btn-outline-secondary">
                <i className="fas fa-cog"></i> 配置
              </button>
              {deployment.status === 'running' ? (
                <button 
                  className="btn btn-sm btn-outline-danger"
                  onClick={() => handleStop(deployment.id)}
                >
                  <i className="fas fa-stop"></i> 停止
                </button>
              ) : (
                <button 
                  className="btn btn-sm btn-outline-success"
                  onClick={() => handleStart(deployment.id)}
                >
                  <i className="fas fa-play"></i> 启动
                </button>
              )}
            </div>
          </div>
        ))}
      </div>

      {showCreateModal && (
        <div className="modal-overlay">
          <div className="modal">
            <h2>新建部署</h2>
            {/* 表单内容 */}
            <div className="modal-actions">
              <button className="btn" onClick={() => setShowCreateModal(false)}>取消</button>
              <button className="btn btn-primary">创建</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Deployments;
