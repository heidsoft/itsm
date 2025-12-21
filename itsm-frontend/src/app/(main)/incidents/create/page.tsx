'use client';

import React from 'react';
import { Button, Card, Result } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';

export default function CreateIncidentPage() {
    const router = useRouter();

    return (
        <div style={{ padding: 24 }}>
            <div style={{ marginBottom: 16 }}>
                <Button
                    type="link"
                    icon={<ArrowLeftOutlined />}
                    onClick={() => router.back()}
                    style={{ paddingLeft: 0, color: '#666' }}
                >
                    返回列表
                </Button>
            </div>
            <Card bordered={false}>
                <Result
                    status="info"
                    title="创建事件功能开发中"
                    subTitle="请稍后重试，或联系管理员。"
                />
            </Card>
        </div>
    );
}
