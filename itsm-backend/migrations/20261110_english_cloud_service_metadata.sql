-- English display names for unchanged system cloud service types.
-- Tenant/provider/service/resource codes, API versions and configuration are preserved.

UPDATE cloud_services SET service_name = 'ECS Cloud Server', resource_type_name = 'Instance'
WHERE provider = 'aliyun' AND service_code = 'ecs' AND resource_type_code = 'instance'
  AND api_version = '2014-05-26' AND is_system = TRUE
  AND service_name = '云服务器 ECS' AND resource_type_name = '实例';

UPDATE cloud_services SET service_name = 'CVM Cloud Server', resource_type_name = 'Instance'
WHERE provider = 'tencent' AND service_code = 'cvm' AND resource_type_code = 'instance'
  AND api_version = '2017-03-12' AND is_system = TRUE
  AND service_name = '云服务器 CVM' AND resource_type_name = '实例';

UPDATE cloud_services SET service_name = 'Amazon EC2 Instance', resource_type_name = 'Instance'
WHERE provider = 'aws' AND service_code = 'ec2' AND resource_type_code = 'instance'
  AND api_version = '2016-11-15' AND is_system = TRUE
  AND service_name = 'Amazon EC2 实例' AND resource_type_name = '实例';

UPDATE cloud_services SET service_name = 'Volcengine ECS Cloud Server', resource_type_name = 'Instance'
WHERE provider = 'volcengine' AND service_code = 'ecs' AND resource_type_code = 'instance'
  AND api_version = '2020-04-01' AND is_system = TRUE
  AND service_name = '云服务器 ECS（火山引擎）' AND resource_type_name = '实例';

UPDATE cloud_services SET service_name = 'Kubernetes Pod', resource_type_name = 'Pod'
WHERE provider = 'k8s' AND service_code = 'pod' AND resource_type_code = 'pod'
  AND api_version = 'v1' AND is_system = TRUE
  AND service_name = 'Kubernetes Pod' AND resource_type_name = '容器组';
