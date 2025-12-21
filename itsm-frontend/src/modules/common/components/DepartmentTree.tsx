'use client';

/**
 * 部门树组件
 */

import React, { useState, useEffect } from 'react';
import { Tree, Card, Spin, message } from 'antd';
import { ClusterOutlined } from '@ant-design/icons';
import { CommonApi } from '../api';
import type { Department } from '../types';

const DepartmentTree: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [treeData, setTreeData] = useState<any[]>([]);

    const formatTreeData = (data: Department[]): any[] => {
        return data.map(item => ({
            title: item.name,
            key: item.id,
            children: item.children ? formatTreeData(item.children) : [],
        }));
    };

    const fetchTree = async () => {
        setLoading(true);
        try {
            const data = await CommonApi.getDepartmentTree();
            setTreeData(formatTreeData(data));
        } catch (error) {
            message.error('获取部门树失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchTree();
    }, []);

    return (
        <Card title={<span><ClusterOutlined /> 组织架构</span>}>
            {loading ? (
                <div style={{ textAlign: 'center', padding: '20px' }}><Spin /></div>
            ) : (
                <Tree
                    showLine
                    defaultExpandAll
                    treeData={treeData}
                />
            )}
        </Card>
    );
};

export default DepartmentTree;
