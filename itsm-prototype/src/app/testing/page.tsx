'use client';

import React from 'react';
import { withRouteGuard } from '../components/RouteGuard';
import TestRunner from '../components/TestRunner';
import { routes, type RouteConfig } from '../../lib/router/route-config';

const TestingPage: React.FC = () => {
  return <TestRunner />;
};

// 获取测试管理路由配置
const testingRoute = routes.find((route: RouteConfig) => route.path === '/testing');

export default withRouteGuard(TestingPage, testingRoute);