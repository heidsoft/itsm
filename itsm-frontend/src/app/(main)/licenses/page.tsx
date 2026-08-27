'use client';

import { Typography } from 'antd';

import LicenseList from '@/components/license/LicenseList';

const { Title } = Typography;

export default function LicensesPage() {
  return (
    <div>
      <Title level={2} style={{ margin: 0, marginBottom: 16, padding: '0 24px' }}>
        许可证管理
      </Title>
      <LicenseList />
    </div>
  );
}
