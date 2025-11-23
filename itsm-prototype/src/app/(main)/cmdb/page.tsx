"use client";

import React, { useState } from "react";
import {
  Card,
  Tabs,
  Button,
  Tree,
  Form,
  Skeleton,
} from "antd";
import {
  DatabaseOutlined,
  CloudServerOutlined,
  DesktopOutlined,
  PlusCircleOutlined,
  ClusterOutlined,
  BranchesOutlined,
} from "@ant-design/icons";
import { useCMDBData } from "./hooks/useCMDBData";
import { CMDBStats } from "./components/CMDBStats";
import { CMDBFilters } from "./components/CMDBFilters";
import { CIList } from "./components/CIList";
import { RelationGraph } from "./components/RelationGraph";
import { TopologyView } from "./components/TopologyView";
import { CreateCIModal } from "./components/CreateCIModal";
import { useI18n } from '@/lib/i18n';
import { message } from 'antd';

const CMDBSkeleton: React.FC = () => (
  <div style={{ padding: 24 }}>
    <Skeleton active paragraph={{ rows: 1 }} />
    <Skeleton active paragraph={{ rows: 2 }} style={{ marginTop: 24 }} />
    <Skeleton active paragraph={{ rows: 8 }} style={{ marginTop: 24 }} />
  </div>
);

const CMDBPage = () => {
  const { t } = useI18n();
  const {
    cis,
    relations,
    loading,
    setSearchText,
    setFilterType,
    refresh,
    setCis,
  } = useCMDBData();
  const [activeTab, setActiveTab] = useState("list");
  const [selectedCI, setSelectedCI] = useState<any | null>(null);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [relationModalVisible, setRelationModalVisible] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const [createForm] = Form.useForm();

  const treeData = [
    {
      title: t('cmdb.infrastructure'),
      key: "infrastructure",
      icon: <DesktopOutlined />,
      children: [
        {
          title: t('cmdb.cloudResources'),
          key: "cloud",
          icon: <CloudServerOutlined />,
          children: [
            { title: t('cmdb.aliyun'), key: "aliyun" },
            { title: t('cmdb.tencentCloud'), key: "tencent" },
          ],
        },
        {
          title: t('cmdb.physicalDevices'),
          key: "physical",
          icon: <DesktopOutlined />,
          children: [
            { title: t('cmdb.servers'), key: "servers" },
            { title: t('cmdb.networkDevices'), key: "network" },
          ],
        },
      ],
    },
    {
      title: t('cmdb.applicationSystems'),
      key: "applications",
      icon: <ClusterOutlined />,
      children: [
        { title: t('cmdb.ecommercePlatform'), key: "ecommerce" },
        { title: t('cmdb.crm'), key: "crm" },
      ],
    },
  ];

  const handleCreateCI = () => {
    setCreateModalVisible(true);
  };

  const handleCreateCIConfirm = async () => {
    try {
      const values = await createForm.validateFields();
      const newCI = {
        id: `CI-${Math.floor(Math.random() * 10000)}`,
        ...values,
        status: "Running",
        business: t('cmdb.newBusinessSystem'),
        owner: t('cmdb.operationsTeam'),
        location: t('cmdb.dataCenter'),
        ip: "192.168.1.102",
        cpu: `${values.cpu || 4} ${t('cmdb.cores')}`,
        memory: `${values.memory || 8}${t('cmdb.gb')}`,
        disk: `${values.disk || 100}${t('cmdb.gbSsd')}`,
      };
      setCis([...cis, newCI]);
      setCreateModalVisible(false);
      createForm.resetFields();
      message.success(t('cmdb.createCISuccess'));
    } catch (error) {
      console.error(t('cmdb.createCIFailed'), error);
      message.error(t('cmdb.createCIFailed'));
    }
  };

  const handleViewRelations = (ci: any) => {
    setSelectedCI(ci);
    setRelationModalVisible(true);
  };

  if (loading) {
    return <CMDBSkeleton />;
  }

  return (
    <div>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 24,
          padding: "20px 0",
        }}
      >
        <div>
          <h2
            style={{
              margin: 0,
              fontSize: 24,
              fontWeight: 700,
              color: "#1a1a1a",
              marginBottom: 8,
            }}
          >
            {t('cmdb.title')}
          </h2>
          <p
            style={{
              margin: 0,
              color: "#666",
              fontSize: 14,
            }}
          >
            {t('cmdb.description')}
          </p>
        </div>
        <Button
          icon={<PlusCircleOutlined />}
          type="primary"
          size="large"
          onClick={handleCreateCI}
        >
          {t('cmdb.newCI')}
        </Button>
      </div>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        style={{ marginBottom: 24 }}
        size="large"
        items={[
          {
            key: "list",
            label: (
              <span>
                <DatabaseOutlined style={{ marginRight: 8 }} />
                {t('cmdb.ciList')}
              </span>
            ),
          },
          {
            key: "relations",
            label: (
              <span>
                <ClusterOutlined style={{ marginRight: 8 }} />
                {t('cmdb.ciRelations')}
              </span>
            ),
          },
          {
            key: "topology",
            label: (
              <span>
                <BranchesOutlined style={{ marginRight: 8 }} />
                {t('cmdb.ciTopology')}
              </span>
            ),
          },
        ]}
      />

      {activeTab === "list" && (
        <>
          <CMDBStats cis={cis} />
          <CMDBFilters
            loading={loading}
            onSearch={setSearchText}
            onFilterTypeChange={setFilterType}
            onFilterStatusChange={() => {}}
            onRefresh={refresh}
            onCreateCI={handleCreateCI}
          />
          <CIList
            cis={cis}
            loading={loading}
            selectedRowKeys={selectedRowKeys}
            onSelectedRowKeysChange={setSelectedRowKeys}
            onViewRelations={handleViewRelations}
          />
        </>
      )}

      {activeTab === "relations" && (
        <Card>
          <div style={{ display: "flex", height: "100%" }}>
            <div
              style={{
                width: 300,
                borderRight: "1px solid #f0f0f0",
                padding: "16px 0",
              }}
            >
              <Tree
                showIcon
                treeData={treeData}
                defaultExpandedKeys={[
                  "infrastructure",
                  "cloud",
                  "physical",
                  "applications",
                ]}
              />
            </div>
            <div style={{ flex: 1 }}>
              <RelationGraph
                selectedCI={selectedCI}
                relations={relations}
                cis={cis}
                onClose={() => setSelectedCI(null)}
              />
            </div>
          </div>
        </Card>
      )}

      {activeTab === "topology" && (
        <Card>
          <TopologyView cis={cis} relations={relations} />
        </Card>
      )}

      <CreateCIModal
        visible={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          createForm.resetFields();
        }}
        onConfirm={handleCreateCIConfirm}
        form={createForm}
      />
    </div>
  );
};

export default CMDBPage;
