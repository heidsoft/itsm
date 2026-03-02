import React, { useState, useEffect } from 'react';
import { deployService } from '../services/deploy';
import './Deployments.css';

function Deployments() {
  const [deployments, setDeployments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [metrics, setMetrics] = useState({});

  useEffect(() => {
    loadDeployments();
  }, []);

  const loadDeployments = async () => {
    try {
      setLoading(true);
      const data = await deployService.getDeployments();
      setDeployments(data.data || []);
      
      // 加载监控数据
      const metricsData = {};
      for (const deployment of data.data || []) {
        try {
          const metrics = await deployService.getMetrics(deployment.id);
          metricsData[deployment.id] = metrics.data;
        } catch (error) {
          console.error('Failed to load metrics:', error);
        }
      }
      setMetrics(metricsData);
    } catch (error) {
      console.error('Failed to load deployments:', error);
      alert('加载部署列表失败：' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleStart = async (id) => {
    try {
      await deployService.startDeployment(id);
      loadDeployments();
      alert('启动成功');
    } catch (error) {
      alert('启动失败：' + error.message);
    }
  };

  const handleStop = async (id) => {
    try {
      await deployService.stopDeployment(id);
      loadDeployments();
      alert('停止成功');
    } catch (error) {
      alert('停止失败：' + error.message);
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('确定要删除这个部署吗？此操作不可恢复！')) {
      return;
    }
    try {
      await deployService.deleteDeployment(id);
      loadDeployments();
      alert('删除成功');
    } catch (error) {
      alert('删除失败：' + error.message);
    }
  };

  const getStatusBadge = (status) => {
    const statusMap = {
      running: { class: 'status-running', text: '运行中', icon: 'fa-circle' },
      deploying: { class: 'status-deploying', text: '部署中', icon: 'fa-spinner fa-spin' },
      stopped: { class: 'status-stopped', text: '已停止', icon: 'fa-circle' },
      error: { class: 'status-error', text: '错误', icon: 'fa-exclamation-circle' }
    };
    const statusInfo = statusMap[status] || statusMap.stopped;
    return (
      <span className={`status-badge ${statusInfo.class}`}>
        <i className={`fas ${statusInfo.icon}`}></i> {statusInfo.text}
      </span>
    );
  };

  if (loading) {
    return <div className="loading-container"><div className="loading">加载中...</div></div>;
  }

  return (
    <div className="deployments-page">
      <div className="page-header">
        <h1><i className="fas fa-server"></i> 部署实例</h1>
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
                {deployment.domain && (
                  <small className="domain">
                    <i className="fas fa-globe"></i> {deployment.domain}
                  </small>
                )}
              </div>
              {getStatusBadge(deployment.status)}
            </div>

            <div className="deployment-metrics">
              {metrics[deployment.id] && (
                <>
                  <div className="metric">
                    <div className="metric-value">{metrics[deployment.id].cpu_usage}%</div>
                    <div className="metric-label">CPU</div>
                  </div>
                  <div className="metric">
                    <div className="metric-value">{metrics[deployment.id].memory_usage}GB</div>
                    <div className="metric-label">内存</div>
                  </div>
                  <div className="metric">
                    <div className="metric-value">{metrics[deployment.id].qps}</div>
                    <div className="metric-label">QPS</div>
                  </div>
                  <div className="metric">
                    <div className="metric-value">{metrics[deployment.id].response_time}ms</div>
                    <div className="metric-label">响应</div>
                  </div>
                </>
              )}
            </div>

            <div className="deployment-actions">
              <button className="btn btn-sm btn-outline-primary" title="监控">
                <i className="fas fa-chart-line"></i>
              </button>
              <button className="btn btn-sm btn-outline-secondary" title="配置">
                <i className="fas fa-cog"></i>
              </button>
              <button className="btn btn-sm btn-outline-info" title="日志">
                <i className="fas fa-file-alt"></i>
              </button>
              {deployment.status === 'running' ? (
                <button 
                  className="btn btn-sm btn-outline-danger"
                  onClick={() => handleStop(deployment.id)}
                  title="停止"
                >
                  <i className="fas fa-stop"></i>
                </button>
              ) : (
                <button 
                  className="btn btn-sm btn-outline-success"
                  onClick={() => handleStart(deployment.id)}
                  title="启动"
                >
                  <i className="fas fa-play"></i>
                </button>
              )}
              <button 
                className="btn btn-sm btn-outline-danger"
                onClick={() => handleDelete(deployment.id)}
                title="删除"
              >
                <i className="fas fa-trash"></i>
              </button>
            </div>
          </div>
        ))}
      </div>

      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>新建部署</h2>
            <form onSubmit={(e) => {
              e.preventDefault();
              // TODO: 实现创建逻辑
              setShowCreateModal(false);
              loadDeployments();
            }}>
              <div className="form-group">
                <label>套餐选择</label>
                <select required>
                  <option value="">请选择套餐</option>
                  <option value="community">社区版（免费）</option>
                  <option value="pro">专业版（¥9,800/年）</option>
                  <option value="enterprise">企业版（定制）</option>
                </select>
              </div>
              <div className="form-group">
                <label>实例名称</label>
                <input type="text" placeholder="请输入实例名称" required />
              </div>
              <div className="form-group">
                <label>域名</label>
                <input type="text" placeholder="xxx.openclaw.cn" />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn" onClick={() => setShowCreateModal(false)}>取消</button>
                <button type="submit" className="btn btn-primary">创建</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

export default Deployments;
