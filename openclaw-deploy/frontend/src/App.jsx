import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, Link, useLocation } from 'react-router-dom';
import { deployService, userService } from './services/deploy';
import Login from './pages/Login';
import Deployments from './pages/Deployments';
import Dashboard from './pages/Dashboard';
import Monitoring from './pages/Monitoring';
import Settings from './pages/Settings';
import './App.css';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [currentUser, setCurrentUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    checkAuth();
  }, []);

  const checkAuth = async () => {
    const token = localStorage.getItem('token');
    if (token) {
      try {
        const user = await userService.getCurrentUser();
        setCurrentUser(user);
        setIsAuthenticated(true);
      } catch (error) {
        localStorage.removeItem('token');
        setIsAuthenticated(false);
      }
    }
    setLoading(false);
  };

  const handleLogin = async (username, password) => {
    const user = await userService.login(username, password);
    setCurrentUser(user);
    setIsAuthenticated(true);
  };

  const handleLogout = () => {
    userService.logout();
    setIsAuthenticated(false);
    setCurrentUser(null);
  };

  if (loading) {
    return <div className="loading-container"><div className="loading">加载中...</div></div>;
  }

  if (!isAuthenticated) {
    return (
      <Router>
        <Routes>
          <Route path="/login" element={<Login onLogin={handleLogin} />} />
          <Route path="*" element={<Navigate to="/login" />} />
        </Routes>
      </Router>
    );
  }

  return (
    <Router>
      <div className="app">
        <Sidebar />
        <div className="main-content">
          <Header user={currentUser} onLogout={handleLogout} />
          <div className="content">
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/deployments" element={<Deployments />} />
              <Route path="/monitoring" element={<Monitoring />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="*" element={<Navigate to="/" />} />
            </Routes>
          </div>
        </div>
      </div>
    </Router>
  );
}

function Sidebar() {
  const location = useLocation();
  
  const menuItems = [
    { path: '/', icon: 'fa-home', text: '概览' },
    { path: '/deployments', icon: 'fa-server', text: '部署实例' },
    { path: '/monitoring', icon: 'fa-chart-line', text: '监控告警' },
    { path: '/settings', icon: 'fa-cog', text: '系统设置' },
  ];

  return (
    <div className="sidebar">
      <div className="sidebar-brand">
        <i className="fas fa-rocket"></i> OpenClaw
      </div>
      <ul className="sidebar-menu">
        {menuItems.map(item => (
          <li key={item.path}>
            <Link to={item.path} className={location.pathname === item.path ? 'active' : ''}>
              <i className={`fas ${item.icon}`}></i>
              <span>{item.text}</span>
            </Link>
          </li>
        ))}
        <li>
          <a href="/" target="_blank" rel="noopener noreferrer">
            <i className="fas fa-external-link-alt"></i>
            <span>返回官网</span>
          </a>
        </li>
      </ul>
    </div>
  );
}

function Header({ user, onLogout }) {
  return (
    <div className="top-bar">
      <div>
        <h4>部署管理后台</h4>
        <p className="text-muted mb-0">欢迎回来，{user?.username || '管理员'}</p>
      </div>
      <div>
        <button className="btn btn-outline-secondary" onClick={onLogout}>
          <i className="fas fa-sign-out-alt"></i> 退出
        </button>
      </div>
    </div>
  );
}

export default App;
