// 模拟CMDB关系数据
export interface CI {
  id: string;
  name: string;
  type: string;
  status: string;
  business: string;
  owner: string;
  location: string;
  ip: string;
  cpu: string;
  memory: string;
  disk: string;
}

export interface Relation {
  id: string;
  source: string;
  target: string;
  type: string;
  description: string;
}

export const mockCIs: CI[] = [
  {
    id: "CI-ECS-001",
    name: "Web服务器-生产环境",
    type: "云服务器",
    status: "运行中",
    business: "电商平台",
    owner: "运维部",
    location: "阿里云华东1",
    ip: "192.168.1.100",
    cpu: "4核",
    memory: "8GB",
    disk: "100GB SSD",
  },
  {
    id: "CI-ECS-002",
    name: "数据库服务器-生产环境",
    type: "云服务器",
    status: "运行中",
    business: "电商平台",
    owner: "运维部",
    location: "阿里云华东1",
    ip: "192.168.1.101",
    cpu: "8核",
    memory: "16GB",
    disk: "500GB SSD",
  },
  {
    id: "CI-RDS-001",
    name: "MySQL主库",
    type: "关系型数据库",
    status: "运行中",
    business: "电商平台",
    owner: "运维部",
    location: "阿里云华东1",
    ip: "192.168.2.100",
    cpu: "4核",
    memory: "16GB",
    disk: "1TB SSD",
  },
  {
    id: "CI-LB-001",
    name: "负载均衡器",
    type: "网络设备",
    status: "运行中",
    business: "电商平台",
    owner: "运维部",
    location: "阿里云华东1",
    ip: "192.168.1.50",
    cpu: "2核",
    memory: "4GB",
    disk: "50GB SSD",
  },
  {
    id: "CI-REDIS-001",
    name: "Redis缓存",
    type: "缓存数据库",
    status: "运行中",
    business: "电商平台",
    owner: "运维部",
    location: "阿里云华东1",
    ip: "192.168.3.100",
    cpu: "2核",
    memory: "8GB",
    disk: "50GB SSD",
  },
];

export const mockRelations: Relation[] = [
  {
    id: "REL-001",
    source: "CI-ECS-001",
    target: "CI-LB-001",
    type: "依赖",
    description: "Web服务器通过负载均衡器对外提供服务",
  },
  {
    id: "REL-002",
    source: "CI-ECS-001",
    target: "CI-RDS-001",
    type: "依赖",
    description: "Web服务器连接数据库服务器获取数据",
  },
  {
    id: "REL-003",
    source: "CI-ECS-001",
    target: "CI-REDIS-001",
    type: "依赖",
    description: "Web服务器使用Redis缓存提升性能",
  },
  {
    id: "REL-004",
    source: "CI-ECS-002",
    target: "CI-RDS-001",
    type: "宿主",
    description: "数据库运行在数据库服务器上",
  },
];