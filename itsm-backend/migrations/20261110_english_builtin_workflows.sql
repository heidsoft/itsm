-- Translate English display metadata only for completely unchanged built-in workflows.
-- Entire XML SHA-256 (CRLF normalized) + original name + description must match.
-- Customized workflows, including metadata-only edits, are wholly untouched.
-- IDs, versions, conditions, assignments, tenant ownership and lifecycle fields are unchanged.
-- Idempotent; run inside the standard migration transaction. No schema changes.
DO $migration$
DECLARE
    template JSONB;
    replacement JSONB;
    source RECORD;
    source_xml TEXT;
    translated_xml TEXT;
    affected INTEGER := 0;
BEGIN
    FOR template IN SELECT value FROM jsonb_array_elements($templates$
[
  {
    "kind": "definition",
    "key": "change_emergency_flow",
    "oldHash": "503d12a0425015878097d413eeacf927f5cb33770b5067f12ae0b28503cbb586",
    "newHash": "57df231e22fcd75f21f78d31b742980e12fdc6282797234d45eebb76852c3ce1",
    "oldName": "紧急变更流程",
    "newName": "Emergency Change Workflow",
    "oldDescription": "紧急变更快速处理流程",
    "newDescription": "Expedited emergency change workflow",
    "replacements": [
      [
        "<bpmn:process id=\"change_emergency_flow\" name=\"紧急变更流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"change_emergency_flow\" name=\"Emergency Change Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">紧急变更快速处理流程（单级紧急审批，跳过 CAB 与排期）</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Expedited emergency change workflow with one approval level, bypassing CAB and scheduling</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"紧急变更申请\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Emergency Change Request\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Assessment\" name=\"紧急评估\">",
        "<bpmn:userTask id=\"Activity_Assessment\" name=\"Emergency Assessment\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_EmergencyApproval\" name=\"紧急审批\">",
        "<bpmn:userTask id=\"Activity_EmergencyApproval\" name=\"Emergency Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"审批结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"Approval Result?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Reject\" name=\"紧急变更驳回\">",
        "<bpmn:userTask id=\"Activity_Reject\" name=\"Reject Emergency Change\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Implement\" name=\"紧急实施\">",
        "<bpmn:userTask id=\"Activity_Implement\" name=\"Emergency Implementation\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Verify\" name=\"紧急变更验证\">",
        "<bpmn:userTask id=\"Activity_Verify\" name=\"Verify Emergency Change\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_VerifyResult\" name=\"验证结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_VerifyResult\" name=\"Verification Result?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Close\" name=\"关闭紧急变更\">",
        "<bpmn:userTask id=\"Activity_Close\" name=\"Close Emergency Change\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"紧急变更完成\">",
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"Emergency Change Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "change_normal_flow",
    "oldHash": "830b3dfd66aa75f5942fbcf7e7ca2d9e77831318c90c55b0ed7ac48757522b11",
    "newHash": "e4a224055b5732389479ac8f905ad07807f128c564cbe53b4d42dbe87a0a194b",
    "oldName": "普通变更流程",
    "newName": "Normal Change Workflow",
    "oldDescription": "普通变更管理流程",
    "newDescription": "Normal change management workflow",
    "replacements": [
      [
        "<bpmn:process id=\"change_normal_flow\" name=\"普通变更流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"change_normal_flow\" name=\"Normal Change Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">普通变更管理流程</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Normal change management workflow</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"变更申请\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Change Request\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Assessment\" name=\"变更评估\">",
        "<bpmn:userTask id=\"Activity_Assessment\" name=\"Change Assessment\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"是否需要审批?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"Approval Required?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_CABApproval\" name=\"CAB审批\">",
        "<bpmn:userTask id=\"Activity_CABApproval\" name=\"CAB Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"审批结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"Approval Result?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Reject\" name=\"变更驳回\">",
        "<bpmn:userTask id=\"Activity_Reject\" name=\"Reject Change\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Schedule\" name=\"变更排期\">",
        "<bpmn:userTask id=\"Activity_Schedule\" name=\"Schedule Change\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Implement\" name=\"变更实施\">",
        "<bpmn:userTask id=\"Activity_Implement\" name=\"Implement Change\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Verify\" name=\"变更验证\">",
        "<bpmn:userTask id=\"Activity_Verify\" name=\"Verify Change\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_VerifyResult\" name=\"验证结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_VerifyResult\" name=\"Verification Result?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Close\" name=\"关闭变更\">",
        "<bpmn:userTask id=\"Activity_Close\" name=\"Close Change\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"变更完成\">",
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"Change Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "change_normal_flow_cn",
    "oldHash": "407d27058c45a3a092b7bdc1cea3062151819aa1e153117c3f903b051f04e5d1",
    "newHash": "08b4e74cd17347ddba90c3844c61102c2c7799916bb37e5f1f811b083bbc89b2",
    "oldName": "change_normal_flow_cn",
    "newName": "Standard Change Workflow (Enterprise Edition)",
    "oldDescription": "",
    "newDescription": "Enterprise standard change workflow with multi-level approval and risk assessment",
    "replacements": [
      [
        "<bpmn:process id=\"change_normal_flow_cn\" name=\"标准变更流程(国内企业版)\" isExecutable=\"true\">",
        "<bpmn:process id=\"change_normal_flow_cn\" name=\"Standard Change Workflow (Enterprise Edition)\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">国内企业标准变更流程，包含多级审批和风险评估</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Enterprise standard change workflow with multi-level approval and risk assessment</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_RFC\" name=\"变更申请提交\">",
        "<bpmn:startEvent id=\"StartEvent_RFC\" name=\"Submit Change Request\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_CMReview\" name=\"变更经理初审\">",
        "<bpmn:userTask id=\"Activity_CMReview\" name=\"Initial Change Manager Review\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ChangeType\" name=\"变更类型判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ChangeType\" name=\"Determine Change Type\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_LowRiskApprove\" name=\"低风险审批\">",
        "<bpmn:userTask id=\"Activity_LowRiskApprove\" name=\"Low-Risk Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_LowRisk\" name=\"低风险审批结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_LowRisk\" name=\"Low-Risk Approval Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_MediumRiskTechReview\" name=\"技术评审\">",
        "<bpmn:userTask id=\"Activity_MediumRiskTechReview\" name=\"Technical Review\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_MediumRiskManagerApprove\" name=\"运维经理审批\">",
        "<bpmn:userTask id=\"Activity_MediumRiskManagerApprove\" name=\"Operations Manager Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_MediumRisk\" name=\"中风险审批结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_MediumRisk\" name=\"Medium-Risk Approval Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_HighRiskTechReview\" name=\"专家技术评审\">",
        "<bpmn:userTask id=\"Activity_HighRiskTechReview\" name=\"Expert Technical Review\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_HighRiskSecurityCheck\" name=\"安全合规检查\">",
        "<bpmn:userTask id=\"Activity_HighRiskSecurityCheck\" name=\"Security Compliance Check\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_HighRiskOpsDirectorApprove\" name=\"运维总监审批\">",
        "<bpmn:userTask id=\"Activity_HighRiskOpsDirectorApprove\" name=\"Operations Director Approval\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_HighRiskITDirectorApprove\" name=\"IT总监审批\">",
        "<bpmn:userTask id=\"Activity_HighRiskITDirectorApprove\" name=\"IT Director Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_HighRisk\" name=\"高风险审批结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_HighRisk\" name=\"High-Risk Approval Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Implementation_Low\" name=\"低风险变更实施\">",
        "<bpmn:userTask id=\"Activity_Implementation_Low\" name=\"Implement Low-Risk Change\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Implementation_Medium\" name=\"中风险变更实施\">",
        "<bpmn:userTask id=\"Activity_Implementation_Medium\" name=\"Implement Medium-Risk Change\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Implementation_High\" name=\"高风险变更实施\">",
        "<bpmn:userTask id=\"Activity_Implementation_High\" name=\"Implement High-Risk Change\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Verify_Low\" name=\"低风险验证\">",
        "<bpmn:userTask id=\"Activity_Verify_Low\" name=\"Low-Risk Verification\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Verify_Medium\" name=\"中风险验证\">",
        "<bpmn:userTask id=\"Activity_Verify_Medium\" name=\"Medium-Risk Verification\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Verify_High\" name=\"高风险验证\">",
        "<bpmn:userTask id=\"Activity_Verify_High\" name=\"High-Risk Verification\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ChangeClose\" name=\"变更关闭\">",
        "<bpmn:userTask id=\"Activity_ChangeClose\" name=\"Change Closed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_ChangeComplete\" name=\"变更完成\">",
        "<bpmn:endEvent id=\"EndEvent_ChangeComplete\" name=\"Change Completed\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_NotifyRejection\" name=\"通知申请被拒绝\">",
        "<bpmn:serviceTask id=\"Activity_NotifyRejection\" name=\"Notify Request Rejection\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_ChangeRejected\" name=\"变更被拒绝\">",
        "<bpmn:endEvent id=\"EndEvent_ChangeRejected\" name=\"Change Rejected\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "cloud_private_ops_flow",
    "oldHash": "3bd7db86f2d02454ecd00f2d76b8d1a7125326a34d9d6a3832d8846f181e7189",
    "newHash": "370106323335bf899b7449f9f3946be6546e3bf166e9a6bfd89b057aab1a99e7",
    "oldName": "cloud_private_ops_flow",
    "newName": "Private and Hybrid Cloud Operations Workflow",
    "oldDescription": "",
    "newDescription": "Private cloud resource request, change, and operations workflow for VMware, OpenStack, and Kubernetes",
    "replacements": [
      [
        "<process id=\"cloud_private_ops\" name=\"私有云/混合云运维流程\" isExecutable=\"true\" camunda:versionTag=\"1.0\">",
        "<process id=\"cloud_private_ops\" name=\"Private and Hybrid Cloud Operations Workflow\" isExecutable=\"true\" camunda:versionTag=\"1.0\">"
      ],
      [
        "<documentation>私有云（VMware/OpenStack/Kubernetes）资源申请、变更、运维流程</documentation>",
        "<documentation>Private cloud resource request, change, and operations workflow for VMware, OpenStack, and Kubernetes</documentation>"
      ],
      [
        "<startEvent id=\"StartEvent_1\" name=\"开始\">",
        "<startEvent id=\"StartEvent_1\" name=\"Start\">"
      ],
      [
        "<userTask id=\"Task_Submit\" name=\"提交私有云资源申请\" camunda:assignee=\"${initiator}\">",
        "<userTask id=\"Task_Submit\" name=\"Submit Private Cloud Resource Request\" camunda:assignee=\"${initiator}\">"
      ],
      [
        "<userTask id=\"Task_ResourceApproval\" name=\"资源池管理员审批\">",
        "<userTask id=\"Task_ResourceApproval\" name=\"Resource Pool Administrator Approval\">"
      ],
      [
        "<userTask id=\"Task_CapacityCheck\" name=\"容量评估与规划\" camunda:assignee=\"technician\">",
        "<userTask id=\"Task_CapacityCheck\" name=\"Capacity Assessment and Planning\" camunda:assignee=\"technician\">"
      ],
      [
        "<exclusiveGateway id=\"Gateway_Tech\" name=\"技术审批\" default=\"Flow_6\">",
        "<exclusiveGateway id=\"Gateway_Tech\" name=\"Technical Approval\" default=\"Flow_6\">"
      ],
      [
        "<userTask id=\"Task_Reject\" name=\"告知申请人\">",
        "<userTask id=\"Task_Reject\" name=\"Notify Requester\">"
      ],
      [
        "<userTask id=\"Task_InfraPrepare\" name=\"基础设施准备\">",
        "<userTask id=\"Task_InfraPrepare\" name=\"Prepare Infrastructure\">"
      ],
      [
        "<userTask id=\"Task_ResourceOps\" name=\"执行资源操作\" camunda:assignable=\"true\">",
        "<userTask id=\"Task_ResourceOps\" name=\"Execute Resource Operation\" camunda:assignable=\"true\">"
      ],
      [
        "<userTask id=\"Task_Monitoring\" name=\"配置监控告警\">",
        "<userTask id=\"Task_Monitoring\" name=\"Configure Monitoring and Alerts\">"
      ],
      [
        "<userTask id=\"Task_Acceptance\" name=\"验收确认\" camunda:assignee=\"${initiator}\">",
        "<userTask id=\"Task_Acceptance\" name=\"Acceptance Confirmation\" camunda:assignee=\"${initiator}\">"
      ],
      [
        "<endEvent id=\"EndEvent_1\" name=\"结束\">",
        "<endEvent id=\"EndEvent_1\" name=\"End\">"
      ],
      [
        "<sequenceFlow id=\"Flow_5\" sourceRef=\"Gateway_Tech\" name=\"拒绝\" targetRef=\"Task_Reject\">",
        "<sequenceFlow id=\"Flow_5\" sourceRef=\"Gateway_Tech\" name=\"Rejected\" targetRef=\"Task_Reject\">"
      ],
      [
        "<sequenceFlow id=\"Flow_6\" sourceRef=\"Gateway_Tech\" name=\"通过\" targetRef=\"Task_InfraPrepare\"/>",
        "<sequenceFlow id=\"Flow_6\" sourceRef=\"Gateway_Tech\" name=\"Approved\" targetRef=\"Task_InfraPrepare\"/>"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "cloud_public_ops_flow",
    "oldHash": "ba8a690652b881bf4b6620f9eadb38f97116695bf967ad48369fe1d7d62d82b8",
    "newHash": "7f3111c00592af40cb7d30196d45e1be73ffb4f84c4b3ce54c60c39780c38a93",
    "oldName": "cloud_public_ops_flow",
    "newName": "Public Cloud Resource Operations Workflow",
    "oldDescription": "",
    "newDescription": "Public cloud resource request, change, and release workflow for Alibaba Cloud, Tencent Cloud, and AWS",
    "replacements": [
      [
        "<process id=\"cloud_public_ops\" name=\"公有云资源运维流程\" isExecutable=\"true\" camunda:versionTag=\"1.0\">",
        "<process id=\"cloud_public_ops\" name=\"Public Cloud Resource Operations Workflow\" isExecutable=\"true\" camunda:versionTag=\"1.0\">"
      ],
      [
        "<documentation>公有云（阿里云/腾讯云/AWS）资源申请、变更、释放等运维流程</documentation>",
        "<documentation>Public cloud resource request, change, and release workflow for Alibaba Cloud, Tencent Cloud, and AWS</documentation>"
      ],
      [
        "<startEvent id=\"StartEvent_1\" name=\"开始\">",
        "<startEvent id=\"StartEvent_1\" name=\"Start\">"
      ],
      [
        "<userTask id=\"Task_Submit\" name=\"提交云资源申请\" camunda:assignee=\"${initiator}\">",
        "<userTask id=\"Task_Submit\" name=\"Submit Cloud Resource Request\" camunda:assignee=\"${initiator}\">"
      ],
      [
        "<userTask id=\"Task_OpsApproval\" name=\"运维工程师审批\" camunda:assignee=\"technician\">",
        "<userTask id=\"Task_OpsApproval\" name=\"Operations Engineer Approval\" camunda:assignee=\"technician\">"
      ],
      [
        "<exclusiveGateway id=\"Gateway_Manager\" name=\"经理审批\" default=\"Flow_5\">",
        "<exclusiveGateway id=\"Gateway_Manager\" name=\"Manager Approval\" default=\"Flow_5\">"
      ],
      [
        "<userTask id=\"Task_Reject\" name=\"告知申请人\">",
        "<userTask id=\"Task_Reject\" name=\"Notify Requester\">"
      ],
      [
        "<userTask id=\"Task_Execute\" name=\"执行云资源操作\" camunda:assignee=\"technician\">",
        "<userTask id=\"Task_Execute\" name=\"Execute Cloud Resource Operation\" camunda:assignee=\"technician\">"
      ],
      [
        "<userTask id=\"Task_SecurityReview\" name=\"安全审核\">",
        "<userTask id=\"Task_SecurityReview\" name=\"Security Review\">"
      ],
      [
        "<userTask id=\"Task_Verify\" name=\"验证与反馈\" camunda:assignee=\"${initiator}\">",
        "<userTask id=\"Task_Verify\" name=\"Verification and Feedback\" camunda:assignee=\"${initiator}\">"
      ],
      [
        "<endEvent id=\"EndEvent_1\" name=\"结束\">",
        "<endEvent id=\"EndEvent_1\" name=\"End\">"
      ],
      [
        "<sequenceFlow id=\"Flow_4\" sourceRef=\"Gateway_Manager\" name=\"拒绝\" targetRef=\"Task_Reject\">",
        "<sequenceFlow id=\"Flow_4\" sourceRef=\"Gateway_Manager\" name=\"Rejected\" targetRef=\"Task_Reject\">"
      ],
      [
        "<sequenceFlow id=\"Flow_5\" sourceRef=\"Gateway_Manager\" name=\"通过\" targetRef=\"Task_Execute\"/>",
        "<sequenceFlow id=\"Flow_5\" sourceRef=\"Gateway_Manager\" name=\"Approved\" targetRef=\"Task_Execute\"/>"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "cloud_security_scan_flow",
    "oldHash": "1e284e4880e827c9ffb6bdd31400a75ffe128a9fb7bb7a4872e711924ac47a10",
    "newHash": "f760d290b0118aaf44e4c7dc24e4ddb6ec44b7aa53debf0c9180b8224fdcad20",
    "oldName": "cloud_security_scan_flow",
    "newName": "Cloud Security Scan and Remediation Workflow",
    "oldDescription": "",
    "newDescription": "Cloud security scanning, vulnerability remediation, and compliance checks",
    "replacements": [
      [
        "<process id=\"cloud_security_scan\" name=\"云安全扫描与漏洞修复流程\" isExecutable=\"true\" camunda:versionTag=\"1.0\">",
        "<process id=\"cloud_security_scan\" name=\"Cloud Security Scan and Remediation Workflow\" isExecutable=\"true\" camunda:versionTag=\"1.0\">"
      ],
      [
        "<documentation>云环境安全扫描、漏洞修复、合规检查流程</documentation>",
        "<documentation>Cloud security scanning, vulnerability remediation, and compliance checks</documentation>"
      ],
      [
        "<startEvent id=\"StartEvent_1\" name=\"开始\">",
        "<startEvent id=\"StartEvent_1\" name=\"Start\">"
      ],
      [
        "<userTask id=\"Task_TriggerScan\" name=\"触发安全扫描\">",
        "<userTask id=\"Task_TriggerScan\" name=\"Trigger Security Scan\">"
      ],
      [
        "<serviceTask id=\"Task_AutoScan\" name=\"自动安全扫描\" camunda:type=\"external\" camunda:topic=\"security-scan\">",
        "<serviceTask id=\"Task_AutoScan\" name=\"Automated Security Scan\" camunda:type=\"external\" camunda:topic=\"security-scan\">"
      ],
      [
        "<userTask id=\"Task_AnalyzeResult\" name=\"分析扫描结果\">",
        "<userTask id=\"Task_AnalyzeResult\" name=\"Analyze Scan Results\">"
      ],
      [
        "<exclusiveGateway id=\"Gateway_Risk\" name=\"风险等级判定\" default=\"Flow_6\">",
        "<exclusiveGateway id=\"Gateway_Risk\" name=\"Determine Risk Level\" default=\"Flow_6\">"
      ],
      [
        "<userTask id=\"Task_HighRisk\" name=\"高风险立即处置\">",
        "<userTask id=\"Task_HighRisk\" name=\"Immediate High-Risk Remediation\">"
      ],
      [
        "<userTask id=\"Task_MediumRisk\" name=\"计划修复\">",
        "<userTask id=\"Task_MediumRisk\" name=\"Schedule Remediation\">"
      ],
      [
        "<userTask id=\"Task_LowRisk\" name=\"记录并持续监控\">",
        "<userTask id=\"Task_LowRisk\" name=\"Record and Monitor\">"
      ],
      [
        "<userTask id=\"Task_VerifyFix\" name=\"修复验证\">",
        "<userTask id=\"Task_VerifyFix\" name=\"Verify Remediation\">"
      ],
      [
        "<userTask id=\"Task_Report\" name=\"生成安全报告\">",
        "<userTask id=\"Task_Report\" name=\"Generate Security Report\">"
      ],
      [
        "<endEvent id=\"EndEvent_1\" name=\"结束\">",
        "<endEvent id=\"EndEvent_1\" name=\"End\">"
      ],
      [
        "<sequenceFlow id=\"Flow_5\" sourceRef=\"Gateway_Risk\" name=\"高风险\" targetRef=\"Task_HighRisk\">",
        "<sequenceFlow id=\"Flow_5\" sourceRef=\"Gateway_Risk\" name=\"High Risk\" targetRef=\"Task_HighRisk\">"
      ],
      [
        "<sequenceFlow id=\"Flow_6\" sourceRef=\"Gateway_Risk\" name=\"中风险\" targetRef=\"Task_MediumRisk\">",
        "<sequenceFlow id=\"Flow_6\" sourceRef=\"Gateway_Risk\" name=\"Medium Risk\" targetRef=\"Task_MediumRisk\">"
      ],
      [
        "<sequenceFlow id=\"Flow_7\" sourceRef=\"Gateway_Risk\" name=\"低风险\" targetRef=\"Task_LowRisk\">",
        "<sequenceFlow id=\"Flow_7\" sourceRef=\"Gateway_Risk\" name=\"Low Risk\" targetRef=\"Task_LowRisk\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "dev_approval_ops_flow",
    "oldHash": "cc2ef9306c6ed7940bfeeeaf8c4c46180d6c63929a6884d15faf993ab1f486c0",
    "newHash": "6c385a48bc7d60d4653fd7d9d4241672a98d6f45447b0d3af7629df458dc90a4",
    "oldName": "开发审批运维流程",
    "newName": "Development Approval and Operations Workflow",
    "oldDescription": "开发提交→主管审批→运维操作三级流程",
    "newDescription": "Three-stage workflow: development submission, supervisor approval, and operations execution",
    "replacements": [
      [
        "<bpmn:process id=\"dev_approval_ops_flow\" name=\"开发审批运维流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"dev_approval_ops_flow\" name=\"Development Approval and Operations Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">开发提交→主管审批→运维操作三级流程</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Three-stage workflow: development submission, supervisor approval, and operations execution</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"流程开始\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Workflow Started\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_DevSubmit\" name=\"开发提交\">",
        "<bpmn:userTask id=\"Activity_DevSubmit\" name=\"Development Submission\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ManagerApproval\" name=\"主管审批\">",
        "<bpmn:userTask id=\"Activity_ManagerApproval\" name=\"Supervisor Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"审批结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"Approval Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_OpsExecute\" name=\"运维操作\">",
        "<bpmn:userTask id=\"Activity_OpsExecute\" name=\"Operations Execution\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_DevRework\" name=\"开发修改\">",
        "<bpmn:userTask id=\"Activity_DevRework\" name=\"Development Revision\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"流程结束\">",
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"Workflow Ended\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "incident_emergency_flow",
    "oldHash": "28863a604349e0114523dfd821b96c8903bcc7bc498e77c4adb9d33e8ff7a0e7",
    "newHash": "8df17d616c350236e6f2811d535d889bd11c34b57f1b547cc46e16bac0be2942",
    "oldName": "紧急事件流程",
    "newName": "Emergency Incident Workflow",
    "oldDescription": "紧急事件快速响应流程",
    "newDescription": "Rapid emergency incident response workflow",
    "replacements": [
      [
        "<bpmn:process id=\"incident_emergency_flow\" name=\"紧急事件流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"incident_emergency_flow\" name=\"Emergency Incident Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">紧急事件快速响应流程（带主管审批）</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Rapid emergency incident response workflow with supervisor approval</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"事件报告\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Incident Report\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoAssign\" name=\"自动分配\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_AutoAssign\" name=\"Automatic Assignment\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ManagerApproval\" name=\"主管审批\">",
        "<bpmn:userTask id=\"Activity_ManagerApproval\" name=\"Supervisor Approval\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Diagnosis\" name=\"初步诊断\">",
        "<bpmn:userTask id=\"Activity_Diagnosis\" name=\"Initial Diagnosis\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Resolved\" name=\"问题已定位?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Resolved\" name=\"Issue Identified?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Resolve\" name=\"立即解决\">",
        "<bpmn:userTask id=\"Activity_Resolve\" name=\"Resolve Immediately\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoEscalate\" name=\"自动升级\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_AutoEscalate\" name=\"Automatic Escalation\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ExpertHandle\" name=\"专家团队处理\">",
        "<bpmn:userTask id=\"Activity_ExpertHandle\" name=\"Expert Team Response\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Solved\" name=\"问题解决?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Solved\" name=\"Issue Resolved?\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_Notify\" name=\"通知相关方\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_Notify\" name=\"Notify Stakeholders\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Close\" name=\"事件关闭\">",
        "<bpmn:userTask id=\"Activity_Close\" name=\"Incident Closed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"事件完成\">",
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"Incident Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "incident_emergency_flow_cn",
    "oldHash": "26f810e63896742796332b6b951d9b1ef75b298506b8289b3464243754c45c71",
    "newHash": "823cac7b9f84879477bc9c2b13ec243a57db2f7aeebcab4fa066df56677bfca8",
    "oldName": "incident_emergency_flow_cn",
    "newName": "Emergency Incident Response Workflow (Enterprise Edition)",
    "oldDescription": "",
    "newDescription": "Enterprise ITSM emergency incident workflow with rapid response and multi-level escalation",
    "replacements": [
      [
        "<bpmn:process id=\"incident_emergency_flow_cn\" name=\"紧急故障响应流程(国内企业版)\" isExecutable=\"true\">",
        "<bpmn:process id=\"incident_emergency_flow_cn\" name=\"Emergency Incident Response Workflow (Enterprise Edition)\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">国内企业ITSM紧急故障响应流程，包含快速响应、多级升级机制</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Enterprise ITSM emergency incident workflow with rapid response and multi-level escalation</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_P0Report\" name=\"P0故障上报\">",
        "<bpmn:startEvent id=\"StartEvent_P0Report\" name=\"Report P0 Incident\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_InstantNotify\" name=\"即时通知\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_InstantNotify\" name=\"Immediate Notification\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoAssign_P0\" name=\"自动分派\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_AutoAssign_P0\" name=\"Automatic Dispatch\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_L1Response\" name=\"一线响应\">",
        "<bpmn:userTask id=\"Activity_L1Response\" name=\"First-Line Response\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ProblemStatus\" name=\"问题状态判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ProblemStatus\" name=\"Determine Issue Status\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_L1Fix\" name=\"一线修复\">",
        "<bpmn:userTask id=\"Activity_L1Fix\" name=\"First-Line Remediation\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoEscalate_L2\" name=\"自动升级L2\">",
        "<bpmn:serviceTask id=\"Activity_AutoEscalate_L2\" name=\"Automatically Escalate to L2\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_L2Diagnosis\" name=\"L2诊断\">",
        "<bpmn:userTask id=\"Activity_L2Diagnosis\" name=\"L2 Diagnosis\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_L2Status\" name=\"L2处理结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_L2Status\" name=\"L2 Response Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_L2Fix\" name=\"L2修复\">",
        "<bpmn:userTask id=\"Activity_L2Fix\" name=\"L2 Remediation\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_EscalateL3\" name=\"升级L3专家\">",
        "<bpmn:serviceTask id=\"Activity_EscalateL3\" name=\"Escalate to L3 Experts\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_L3ExpertHandle\" name=\"L3专家处理\">",
        "<bpmn:userTask id=\"Activity_L3ExpertHandle\" name=\"L3 Expert Response\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_L3Status\" name=\"L3处理结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_L3Status\" name=\"L3 Response Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_L3Fix\" name=\"L3修复完成\">",
        "<bpmn:userTask id=\"Activity_L3Fix\" name=\"L3 Remediation Completed\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_MajorIncidentEscalate\" name=\"重大事件升级\">",
        "<bpmn:serviceTask id=\"Activity_MajorIncidentEscalate\" name=\"Major Incident Escalation\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_WarRoomHandle\" name=\"事件指挥室处理\">",
        "<bpmn:userTask id=\"Activity_WarRoomHandle\" name=\"Incident Command Response\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_NotifyStakeholders\" name=\"通知相关方\">",
        "<bpmn:serviceTask id=\"Activity_NotifyStakeholders\" name=\"Notify Stakeholders\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_IncidentClose\" name=\"事件关闭\">",
        "<bpmn:userTask id=\"Activity_IncidentClose\" name=\"Incident Closed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Complete\" name=\"事件完成\">",
        "<bpmn:endEvent id=\"EndEvent_Complete\" name=\"Incident Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "incident_emergency_flow_v1.1",
    "oldHash": "837eb074d41c16b69ea42a3da1f5c9fdca7e9e9dad4a6ac1c152c450d30524e6",
    "newHash": "e5eee460ae1273a2820e0b28d76a3920ed97c8b5fb1c049cbd42ec3b3e7b611f",
    "oldName": "incident_emergency_flow_v1.1",
    "newName": "Emergency Incident Workflow",
    "oldDescription": "",
    "newDescription": "Rapid emergency incident response workflow with supervisor approval",
    "replacements": [
      [
        "<bpmn:process id=\"incident_emergency_flow\" name=\"紧急事件流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"incident_emergency_flow\" name=\"Emergency Incident Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">紧急事件快速响应流程（带主管审批）</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Rapid emergency incident response workflow with supervisor approval</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"事件报告\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Incident Report\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoAssign\" name=\"自动分配\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_AutoAssign\" name=\"Automatic Assignment\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ManagerApproval\" name=\"主管审批\">",
        "<bpmn:userTask id=\"Activity_ManagerApproval\" name=\"Supervisor Approval\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Diagnosis\" name=\"初步诊断\">",
        "<bpmn:userTask id=\"Activity_Diagnosis\" name=\"Initial Diagnosis\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Resolved\" name=\"问题已定位?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Resolved\" name=\"Issue Identified?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Resolve\" name=\"立即解决\">",
        "<bpmn:userTask id=\"Activity_Resolve\" name=\"Resolve Immediately\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoEscalate\" name=\"自动升级\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_AutoEscalate\" name=\"Automatic Escalation\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ExpertHandle\" name=\"专家团队处理\">",
        "<bpmn:userTask id=\"Activity_ExpertHandle\" name=\"Expert Team Response\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Solved\" name=\"问题解决?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Solved\" name=\"Issue Resolved?\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_Notify\" name=\"通知相关方\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_Notify\" name=\"Notify Stakeholders\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Close\" name=\"事件关闭\">",
        "<bpmn:userTask id=\"Activity_Close\" name=\"Incident Closed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"事件完成\">",
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"Incident Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "problem_management_flow",
    "oldHash": "daf213cad23b0b1248229788ad29f7582ed7d0274eb7d2d7a6505fca721ccf90",
    "newHash": "c6be94126155f6f597f140cdfc7a989074b17d28b784993b6485af17714ed259",
    "oldName": "问题管理流程",
    "newName": "Problem Management Workflow",
    "oldDescription": "问题管理全流程",
    "newDescription": "End-to-end problem management workflow",
    "replacements": [
      [
        "<bpmn:process id=\"problem_management_flow\" name=\"问题管理流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"problem_management_flow\" name=\"Problem Management Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">问题管理全流程</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">End-to-end problem management workflow</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"问题创建\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Problem Created\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Record\" name=\"问题记录\">",
        "<bpmn:userTask id=\"Activity_Record\" name=\"Record Problem\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Classify\" name=\"问题分类\">",
        "<bpmn:userTask id=\"Activity_Classify\" name=\"Classify Problem\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Investigate\" name=\"调查分析\">",
        "<bpmn:userTask id=\"Activity_Investigate\" name=\"Investigation and Analysis\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_RootCause\" name=\"根因分析\">",
        "<bpmn:userTask id=\"Activity_RootCause\" name=\"Root Cause Analysis\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Solution\" name=\"找到解决方案?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Solution\" name=\"Solution Found?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Solution\" name=\"制定解决方案\">",
        "<bpmn:userTask id=\"Activity_Solution\" name=\"Develop Solution\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Evaluate\" name=\"评估方案\">",
        "<bpmn:userTask id=\"Activity_Evaluate\" name=\"Assess Solution\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_SolutionApproval\" name=\"方案审批\">",
        "<bpmn:userTask id=\"Activity_SolutionApproval\" name=\"Solution Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_SolutionResult\" name=\"审批通过?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_SolutionResult\" name=\"Approval Granted?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Implement\" name=\"实施解决\">",
        "<bpmn:userTask id=\"Activity_Implement\" name=\"Implement Solution\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Verify\" name=\"验证解决\">",
        "<bpmn:userTask id=\"Activity_Verify\" name=\"Verify Resolution\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_VerifyResult\" name=\"验证成功?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_VerifyResult\" name=\"Verification Successful?\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_KnowledgeBase\" name=\"更新知识库\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_KnowledgeBase\" name=\"Update Knowledge Base\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Close\" name=\"问题关闭\">",
        "<bpmn:userTask id=\"Activity_Close\" name=\"Problem Closed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"问题关闭\">",
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"Problem Closed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "problem_management_flow_cn",
    "oldHash": "542e99d6ca8e44f12792e1940d260d8ff25774cd225840363b19794f71fea3ab",
    "newHash": "16fb58ee129fbfa19969acf1cc6d5f50d5f718b594a5176dd9c771d6be84ca0d",
    "oldName": "problem_management_flow_cn",
    "newName": "Problem Management Workflow (Enterprise Edition)",
    "oldDescription": "",
    "newDescription": "Enterprise problem management workflow focused on root cause analysis and prevention",
    "replacements": [
      [
        "<bpmn:process id=\"problem_management_flow_cn\" name=\"问题管理流程(国内企业版)\" isExecutable=\"true\">",
        "<bpmn:process id=\"problem_management_flow_cn\" name=\"Problem Management Workflow (Enterprise Edition)\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">国内企业问题管理流程，专注根因分析和预防</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Enterprise problem management workflow focused on root cause analysis and prevention</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_ProblemCreate\" name=\"问题创建\">",
        "<bpmn:startEvent id=\"StartEvent_ProblemCreate\" name=\"Problem Created\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoClassify\" name=\"自动分类\">",
        "<bpmn:serviceTask id=\"Activity_AutoClassify\" name=\"Automatic Classification\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ProblemManagerAssign\" name=\"问题经理分配\">",
        "<bpmn:userTask id=\"Activity_ProblemManagerAssign\" name=\"Problem Manager Assignment\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_RootCauseAnalysis\" name=\"根因分析\">",
        "<bpmn:userTask id=\"Activity_RootCauseAnalysis\" name=\"Root Cause Analysis\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_RootCauseStatus\" name=\"根因判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_RootCauseStatus\" name=\"Determine Root Cause\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_RootCauseConfirmed\" name=\"根因确认\">",
        "<bpmn:userTask id=\"Activity_RootCauseConfirmed\" name=\"Confirm Root Cause\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_RootCausePendingNotify\" name=\"通知相关人\">",
        "<bpmn:serviceTask id=\"Activity_RootCausePendingNotify\" name=\"Notify Stakeholders\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_WaitRootCause\" name=\"等待根因分析\">",
        "<bpmn:userTask id=\"Activity_WaitRootCause\" name=\"Await Root Cause Analysis\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_SolutionDesign\" name=\"解决方案设计\">",
        "<bpmn:userTask id=\"Activity_SolutionDesign\" name=\"Solution Design\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_SolutionReview\" name=\"解决方案评审\">",
        "<bpmn:exclusiveGateway id=\"Gateway_SolutionReview\" name=\"Solution Review\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_SolutionApproved\" name=\"解决方案审批\">",
        "<bpmn:userTask id=\"Activity_SolutionApproved\" name=\"Solution Approval\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_SolutionRevise\" name=\"解决方案修订\">",
        "<bpmn:userTask id=\"Activity_SolutionRevise\" name=\"Revise Solution\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_CreateRFC\" name=\"创建变更请求\">",
        "<bpmn:serviceTask id=\"Activity_CreateRFC\" name=\"Create Change Request\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ImplementSolution\" name=\"解决方案实施\">",
        "<bpmn:userTask id=\"Activity_ImplementSolution\" name=\"Implement Solution\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_VerifySolution\" name=\"解决方案验证\">",
        "<bpmn:userTask id=\"Activity_VerifySolution\" name=\"Verify Solution\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_VerifyResult\" name=\"验证结果判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_VerifyResult\" name=\"Determine Verification Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ReviseSolution\" name=\"修订解决方案\">",
        "<bpmn:userTask id=\"Activity_ReviseSolution\" name=\"Revise Solution\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_CreateKBArticle\" name=\"创建知识库\">",
        "<bpmn:serviceTask id=\"Activity_CreateKBArticle\" name=\"Create Knowledge Article\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_TriggerPrevention\" name=\"触发预防措施\">",
        "<bpmn:serviceTask id=\"Activity_TriggerPrevention\" name=\"Trigger Preventive Measures\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ProblemClose\" name=\"问题关闭\">",
        "<bpmn:userTask id=\"Activity_ProblemClose\" name=\"Problem Closed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_ProblemComplete\" name=\"问题关闭\">",
        "<bpmn:endEvent id=\"EndEvent_ProblemComplete\" name=\"Problem Closed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "release_approval_flow",
    "oldHash": "490fac63e6c16f039ec172949b806148116fb3d38d5c8dfaf5d2ce496ad9e0ed",
    "newHash": "35061bf3db80017f7798dc263efad1826eaa83bb88b2c2a88a751c8dd0c08449",
    "oldName": "发布审批流程",
    "newName": "Release Approval Workflow",
    "oldDescription": "软件发布审批管理流程",
    "newDescription": "Software release approval workflow",
    "replacements": [
      [
        "<bpmn:process id=\"release_approval_flow\" name=\"发布审批流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"release_approval_flow\" name=\"Release Approval Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">软件发布审批管理流程</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Software release approval workflow</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"发布申请\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Release Request\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_TechReview\" name=\"技术评审\">",
        "<bpmn:userTask id=\"Activity_TechReview\" name=\"Technical Review\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_TechResult\" name=\"评审通过?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_TechResult\" name=\"Review Passed?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Approval\" name=\"发布审批\">",
        "<bpmn:userTask id=\"Activity_Approval\" name=\"Release Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"审批结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"Approval Result?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Schedule\" name=\"计划发布\">",
        "<bpmn:userTask id=\"Activity_Schedule\" name=\"Schedule Release\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Execute\" name=\"执行发布\">",
        "<bpmn:userTask id=\"Activity_Execute\" name=\"Execute Release\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Verify\" name=\"验证确认\">",
        "<bpmn:userTask id=\"Activity_Verify\" name=\"Verification Confirmation\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Complete\" name=\"发布完成\">",
        "<bpmn:endEvent id=\"EndEvent_Complete\" name=\"Release Completed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Reject\" name=\"发布拒绝\">",
        "<bpmn:endEvent id=\"EndEvent_Reject\" name=\"Release Rejected\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "release_approval_flow_cn",
    "oldHash": "fb6d28f4a45a4579e5a63b72e6e3e28b36186d274d2a22c332d621a0cecb13e7",
    "newHash": "a7483e321bba1bcfb02333b8f67bea938943c65150d82c388f1144fb080a5eec",
    "oldName": "release_approval_flow_cn",
    "newName": "Release Approval Workflow (Enterprise Edition)",
    "oldDescription": "",
    "newDescription": "Enterprise software release approval workflow with comprehensive testing and canary deployment",
    "replacements": [
      [
        "<bpmn:process id=\"release_approval_flow_cn\" name=\"发布审批流程(国内企业版)\" isExecutable=\"true\">",
        "<bpmn:process id=\"release_approval_flow_cn\" name=\"Release Approval Workflow (Enterprise Edition)\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">国内企业软件发布审批流程，含严格测试和灰度发布</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Enterprise software release approval workflow with comprehensive testing and canary deployment</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_ReleaseRequest\" name=\"发布申请\">",
        "<bpmn:startEvent id=\"StartEvent_ReleaseRequest\" name=\"Release Request\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ReleaseManagerReview\" name=\"发布经理初审\">",
        "<bpmn:userTask id=\"Activity_ReleaseManagerReview\" name=\"Initial Release Manager Review\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ReleaseType\" name=\"发布类型判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ReleaseType\" name=\"Determine Release Type\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_HotfixTest\" name=\"快速测试\">",
        "<bpmn:userTask id=\"Activity_HotfixTest\" name=\"Quick Tests\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_HotfixApproval\" name=\"热修复审批\">",
        "<bpmn:userTask id=\"Activity_HotfixApproval\" name=\"Hotfix Approval\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_HotfixDeploy\" name=\"热修复部署\">",
        "<bpmn:serviceTask id=\"Activity_HotfixDeploy\" name=\"Deploy Hotfix\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_HotfixVerify\" name=\"热修复验证\">",
        "<bpmn:userTask id=\"Activity_HotfixVerify\" name=\"Verify Hotfix\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_UnitTest\" name=\"单元测试\">",
        "<bpmn:userTask id=\"Activity_UnitTest\" name=\"Unit Tests\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_IntegrationTest\" name=\"集成测试\">",
        "<bpmn:userTask id=\"Activity_IntegrationTest\" name=\"Integration Tests\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_UAT\" name=\"用户验收测试\">",
        "<bpmn:userTask id=\"Activity_UAT\" name=\"User Acceptance Tests\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_NormalTest\" name=\"测试结果判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_NormalTest\" name=\"Determine Test Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_NormalApproval\" name=\"常规发布审批\">",
        "<bpmn:userTask id=\"Activity_NormalApproval\" name=\"Standard Release Approval\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_StagingDeploy\" name=\"预发布部署\">",
        "<bpmn:serviceTask id=\"Activity_StagingDeploy\" name=\"Deploy to Staging\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_FixIssues\" name=\"修复问题\">",
        "<bpmn:userTask id=\"Activity_FixIssues\" name=\"Fix Issues\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_SecurityTest\" name=\"安全测试\">",
        "<bpmn:userTask id=\"Activity_SecurityTest\" name=\"Security Tests\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_PerformanceTest\" name=\"性能测试\">",
        "<bpmn:userTask id=\"Activity_PerformanceTest\" name=\"Performance Tests\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_IntegrationTest_Major\" name=\"集成测试\">",
        "<bpmn:userTask id=\"Activity_IntegrationTest_Major\" name=\"Integration Tests\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_UAT_Major\" name=\"用户验收测试\">",
        "<bpmn:userTask id=\"Activity_UAT_Major\" name=\"User Acceptance Tests\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_MajorTest\" name=\"测试结果判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_MajorTest\" name=\"Determine Test Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_MajorTechApproval\" name=\"技术评审\">",
        "<bpmn:userTask id=\"Activity_MajorTechApproval\" name=\"Technical Review\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_MajorSecurityApproval\" name=\"安全审批\">",
        "<bpmn:userTask id=\"Activity_MajorSecurityApproval\" name=\"Security Approval\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_MajorITDirectorApproval\" name=\"IT总监审批\">",
        "<bpmn:userTask id=\"Activity_MajorITDirectorApproval\" name=\"IT Director Approval\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_StagingDeploy_Major\" name=\"预发布部署\">",
        "<bpmn:serviceTask id=\"Activity_StagingDeploy_Major\" name=\"Deploy to Staging\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_FixIssues_Major\" name=\"修复问题\">",
        "<bpmn:userTask id=\"Activity_FixIssues_Major\" name=\"Fix Issues\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_CanaryDeploy\" name=\"灰度发布(10%)\">",
        "<bpmn:serviceTask id=\"Activity_CanaryDeploy\" name=\"Canary Deployment (10%)\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_CanaryMonitor\" name=\"灰度监控\">",
        "<bpmn:userTask id=\"Activity_CanaryMonitor\" name=\"Canary Monitoring\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_CanaryResult\" name=\"灰度结果判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_CanaryResult\" name=\"Determine Canary Result\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoRollback\" name=\"自动回滚\">",
        "<bpmn:serviceTask id=\"Activity_AutoRollback\" name=\"Automatic Rollback\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_FullDeploy\" name=\"全量发布\">",
        "<bpmn:serviceTask id=\"Activity_FullDeploy\" name=\"Full Deployment\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_PostReleaseVerify\" name=\"发布后验证\">",
        "<bpmn:userTask id=\"Activity_PostReleaseVerify\" name=\"Post-Release Verification\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ReleaseClose\" name=\"发布关闭\">",
        "<bpmn:userTask id=\"Activity_ReleaseClose\" name=\"Release Closed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_ReleaseComplete\" name=\"发布完成\">",
        "<bpmn:endEvent id=\"EndEvent_ReleaseComplete\" name=\"Release Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "scheduled_inspection_flow",
    "oldHash": "f119059b98faecfaba8dc13b91a35a6e47d210c3ec311bb97ab11fc2edb3c0c0",
    "newHash": "d6d67f8b6a28e60e9ae37fb6d225f4528767d2f36e2fdd93c127c05470b57e7e",
    "oldName": "scheduled_inspection_flow",
    "newName": "Scheduled Inspection Workflow",
    "oldDescription": "",
    "newDescription": "Cron-based inspection workflow: create scheduled tasks, confirm results, and automatically escalate and notify on exceptions",
    "replacements": [
      [
        "<bpmn:process id=\"scheduled_inspection_flow\" name=\"定时巡检流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"scheduled_inspection_flow\" name=\"Scheduled Inspection Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">按 cron 定时启动的巡检流程：到点自动生成巡检任务，人工确认巡检结果，异常自动升级并通知</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Cron-based inspection workflow: create scheduled tasks, confirm results, and automatically escalate and notify on exceptions</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_Schedule\" name=\"每日 09:00 定时启动\">",
        "<bpmn:startEvent id=\"StartEvent_Schedule\" name=\"Start Daily at 09:00\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_Dispatch\" name=\"下发巡检任务\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_Dispatch\" name=\"Dispatch Inspection Task\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Inspect\" name=\"执行巡检并确认结果\" instantiate=\"false\">",
        "<bpmn:userTask id=\"Activity_Inspect\" name=\"Perform Inspection and Confirm Results\" instantiate=\"false\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_InspectionResult\" name=\"巡检是否异常?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_InspectionResult\" name=\"Inspection Exception?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Escalate\" name=\"异常升级处理\" instantiate=\"false\">",
        "<bpmn:userTask id=\"Activity_Escalate\" name=\"Escalate Exception\" instantiate=\"false\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_NotifyResult\" name=\"通知巡检结果\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_NotifyResult\" name=\"Notify Inspection Results\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Completed\" name=\"巡检结束\">",
        "<bpmn:endEvent id=\"EndEvent_Completed\" name=\"Inspection Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "service_request_flow",
    "oldHash": "713caf87a7703b48384fea5ef6bb7614ce24df9907808e3c4bcaa152baa1b4e6",
    "newHash": "5ca8242d2270bd28f675592d8cb7844a20b8b1ee926e6e899bede82942e9c2d7",
    "oldName": "服务请求流程",
    "newName": "Service Request Workflow",
    "oldDescription": "标准服务请求处理流程",
    "newDescription": "Standard service request workflow",
    "replacements": [
      [
        "<bpmn:process id=\"service_request_flow\" name=\"服务请求流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"service_request_flow\" name=\"Service Request Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">标准服务请求处理流程（审批层级与审批人由流程变量驱动）</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Standard service request workflow with approval levels and approvers determined by process variables</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"请求提交\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Submit Request\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"需要审批?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"Approval Required?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Approval_L1\" name=\"一级审批\"\r\n                   itsm:taskPurpose=\"approval\" itsm:approvalLevel=\"1\"\r\n                   itsm:approvalMode=\"single\" itsm:commentRequiredOnReject=\"true\">",
        "<bpmn:userTask id=\"Activity_Approval_L1\" name=\"Level 1 Approval\"\r\n                   itsm:taskPurpose=\"approval\" itsm:approvalLevel=\"1\"\r\n                   itsm:approvalMode=\"single\" itsm:commentRequiredOnReject=\"true\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_L1_Result\" name=\"一级审批结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_L1_Result\" name=\"Level 1 Approval Result?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Approval_L2\" name=\"二级审批\"\r\n                   itsm:taskPurpose=\"approval\" itsm:approvalLevel=\"2\"\r\n                   itsm:approvalMode=\"single\" itsm:commentRequiredOnReject=\"true\">",
        "<bpmn:userTask id=\"Activity_Approval_L2\" name=\"Level 2 Approval\"\r\n                   itsm:taskPurpose=\"approval\" itsm:approvalLevel=\"2\"\r\n                   itsm:approvalMode=\"single\" itsm:commentRequiredOnReject=\"true\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_L2_Result\" name=\"二级审批结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_L2_Result\" name=\"Level 2 Approval Result?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Approval_L3\" name=\"三级审批\"\r\n                   itsm:taskPurpose=\"approval\" itsm:approvalLevel=\"3\"\r\n                   itsm:approvalMode=\"single\" itsm:commentRequiredOnReject=\"true\">",
        "<bpmn:userTask id=\"Activity_Approval_L3\" name=\"Level 3 Approval\"\r\n                   itsm:taskPurpose=\"approval\" itsm:approvalLevel=\"3\"\r\n                   itsm:approvalMode=\"single\" itsm:commentRequiredOnReject=\"true\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_L3_Result\" name=\"三级审批结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_L3_Result\" name=\"Level 3 Approval Result?\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_RejectNotify\" name=\"驳回通知\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_RejectNotify\" name=\"Rejection Notification\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Execute\" name=\"执行服务\" itsm:taskPurpose=\"work\">",
        "<bpmn:userTask id=\"Activity_Execute\" name=\"Fulfill Service\" itsm:taskPurpose=\"work\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Confirm\" name=\"用户确认\" itsm:taskPurpose=\"work\">",
        "<bpmn:userTask id=\"Activity_Confirm\" name=\"User Confirmation\" itsm:taskPurpose=\"work\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Satisfaction\" name=\"满意?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Satisfaction\" name=\"Satisfied?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Feedback\" name=\"改进服务\" itsm:taskPurpose=\"work\">",
        "<bpmn:userTask id=\"Activity_Feedback\" name=\"Improve Service\" itsm:taskPurpose=\"work\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_Complete\" name=\"服务完成\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_Complete\" name=\"Service Completed\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"流程结束\">",
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"Workflow Ended\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Reject\" name=\"请求被拒\">",
        "<bpmn:endEvent id=\"EndEvent_Reject\" name=\"Request Rejected\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "service_request_flow_cn",
    "oldHash": "0c2e869a55dc54d5977fcbe035269f72f2e2bb74bd658eeb816d9757d490c38c",
    "newHash": "0693627566daa9eae32cf443752db4d451724fdb589f9f779609ab8d0c45e039",
    "oldName": "service_request_flow_cn",
    "newName": "Service Request Workflow (Enterprise Edition)",
    "oldDescription": "",
    "newDescription": "Enterprise service request workflow with automatic dispatch and service catalog integration",
    "replacements": [
      [
        "<bpmn:process id=\"service_request_flow_cn\" name=\"服务请求流程(国内企业版)\" isExecutable=\"true\">",
        "<bpmn:process id=\"service_request_flow_cn\" name=\"Service Request Workflow (Enterprise Edition)\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">国内企业服务请求流程，自动化派单和服务目录化</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Enterprise service request workflow with automatic dispatch and service catalog integration</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_ServiceRequest\" name=\"服务请求提交\">",
        "<bpmn:startEvent id=\"StartEvent_ServiceRequest\" name=\"Submit Service Request\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoCategorize\" name=\"智能分类\">",
        "<bpmn:serviceTask id=\"Activity_AutoCategorize\" name=\"Intelligent Classification\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ServiceType\" name=\"服务类型判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ServiceType\" name=\"Determine Service Type\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ITServiceL1Handle\" name=\"一线处理\">",
        "<bpmn:userTask id=\"Activity_ITServiceL1Handle\" name=\"First-Line Fulfillment\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ITServiceStatus\" name=\"处理结果判定\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ITServiceStatus\" name=\"Determine Fulfillment Result\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ITServiceL2Handle\" name=\"L2处理\">",
        "<bpmn:userTask id=\"Activity_ITServiceL2Handle\" name=\"L2 Fulfillment\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_AccessApproval\" name=\"权限审批\">",
        "<bpmn:userTask id=\"Activity_AccessApproval\" name=\"Access Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_AccessStatus\" name=\"审批结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_AccessStatus\" name=\"Approval Result\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_GrantAccess\" name=\"授权执行\">",
        "<bpmn:serviceTask id=\"Activity_GrantAccess\" name=\"Grant Access\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_SoftwareLicenseCheck\" name=\"License检查\">",
        "<bpmn:userTask id=\"Activity_SoftwareLicenseCheck\" name=\"License Check\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_SoftwareStatus\" name=\"License检查结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_SoftwareStatus\" name=\"License Check Result\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_SoftwareProvision\" name=\"软件部署\">",
        "<bpmn:serviceTask id=\"Activity_SoftwareProvision\" name=\"Software Deployment\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_SoftwarePurchase\" name=\"采购申请\">",
        "<bpmn:serviceTask id=\"Activity_SoftwarePurchase\" name=\"Purchase Request\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_WaitForPurchase\" name=\"等待采购完成\">",
        "<bpmn:userTask id=\"Activity_WaitForPurchase\" name=\"Await Procurement\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_HardwareApproval\" name=\"硬件审批\">",
        "<bpmn:userTask id=\"Activity_HardwareApproval\" name=\"Hardware Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_HardwareStatus\" name=\"审批结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_HardwareStatus\" name=\"Approval Result\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_HardwareProvision\" name=\"硬件准备\">",
        "<bpmn:serviceTask id=\"Activity_HardwareProvision\" name=\"Prepare Hardware\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_NotifyRequester\" name=\"通知申请人\">",
        "<bpmn:serviceTask id=\"Activity_NotifyRequester\" name=\"Notify Requester\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_SatisfactionSurvey\" name=\"满意度评价\">",
        "<bpmn:userTask id=\"Activity_SatisfactionSurvey\" name=\"Satisfaction Survey\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ServiceRequestClose\" name=\"请求关闭\">",
        "<bpmn:userTask id=\"Activity_ServiceRequestClose\" name=\"Request Closed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_ServiceRequestComplete\" name=\"请求完成\">",
        "<bpmn:endEvent id=\"EndEvent_ServiceRequestComplete\" name=\"Request Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "ticket_assignment_flow",
    "oldHash": "1c7a68cce6472d38028ebb0f67b514d63d6983de062ef9ed7f35574dd5979b98",
    "newHash": "789341d3df2163eb46e9d87ec688430e8ffa7afc4f6c34cbae795641bbaf3f3e",
    "oldName": "工单分配流程",
    "newName": "Ticket Assignment Workflow",
    "oldDescription": "工单自动分配处理流程",
    "newDescription": "Automatic ticket assignment workflow",
    "replacements": [
      [
        "<bpmn:process id=\"ticket_assignment_flow\" name=\"工单分配流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"ticket_assignment_flow\" name=\"Ticket Assignment Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">工单自动分配处理流程</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">Automatic ticket assignment workflow</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"工单创建\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Ticket Created\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_AutoAssign\" name=\"自动分配\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_AutoAssign\" name=\"Automatic Assignment\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_AssignResult\" name=\"分配成功?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_AssignResult\" name=\"Assignment Successful?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_ManualAssign\" name=\"手动分配\">",
        "<bpmn:userTask id=\"Activity_ManualAssign\" name=\"Manual Assignment\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_NotifyAssignee\" name=\"通知处理人\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_NotifyAssignee\" name=\"Notify Assignee\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_SLA\" name=\"是否SLA超时?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_SLA\" name=\"SLA Breached?\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_SLAWarn\" name=\"SLA超时告警\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_SLAWarn\" name=\"SLA Breach Alert\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Complete\" name=\"分配完成\">",
        "<bpmn:endEvent id=\"EndEvent_Complete\" name=\"Assignment Completed\">"
      ]
    ]
  },
  {
    "kind": "definition",
    "key": "ticket_general_flow",
    "oldHash": "e57b659522a08e8b69c9c88328ea7807862cc99d2764fb95b858ac876f3273bc",
    "newHash": "98948f4d4a2e5c67d94ec2befa0b019d2b8f3b98a0e87f9333d1329d4a1012a6",
    "oldName": "通用工单流程",
    "newName": "General Ticket Workflow",
    "oldDescription": "通用工单处理流程",
    "newDescription": "General ticket handling workflow",
    "replacements": [
      [
        "<bpmn:process id=\"ticket_general_flow\" name=\"通用工单流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"ticket_general_flow\" name=\"General Ticket Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:metaData name=\"description\">通用工单处理流程</bpmn:metaData>",
        "<bpmn:metaData name=\"description\">General ticket handling workflow</bpmn:metaData>"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"工单创建\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Ticket Created\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Assign\" name=\"任务分配\" instantiate=\"false\">",
        "<bpmn:userTask id=\"Activity_Assign\" name=\"Assign Task\" instantiate=\"false\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Handle\" name=\"工单处理\">",
        "<bpmn:userTask id=\"Activity_Handle\" name=\"Handle Ticket\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Escalate\" name=\"是否需要升级?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Escalate\" name=\"Escalation Required?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Escalate\" name=\"升级处理\">",
        "<bpmn:userTask id=\"Activity_Escalate\" name=\"Handle Escalation\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Resolve\" name=\"解决工单\">",
        "<bpmn:userTask id=\"Activity_Resolve\" name=\"Resolve Ticket\">"
      ],
      [
        "<bpmn:serviceTask id=\"Activity_NotifyRequester\" name=\"通知请求人\" implementation=\"##WebService\">",
        "<bpmn:serviceTask id=\"Activity_NotifyRequester\" name=\"Notify Requester\" implementation=\"##WebService\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"工单关闭\">",
        "<bpmn:endEvent id=\"EndEvent_1\" name=\"Ticket Closed\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"是否需要审批?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"Approval Required?\">"
      ],
      [
        "<bpmn:userTask id=\"Activity_Approval\" name=\"工单审批\">",
        "<bpmn:userTask id=\"Activity_Approval\" name=\"Ticket Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"审批结果?\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"Approval Result?\">"
      ]
    ]
  },
  {
    "kind": "catalog",
    "key": "change_request",
    "oldHash": "994ab06fc5584406b74966a69d67030f366774ae863049b40c96f81ccdb088b0",
    "newHash": "1968d17f1aab6b636c396e6f2b0e454f7b3d5d1ee896985c62027a41da75d2cf",
    "oldName": "变更申请流程",
    "newName": "Change Request Workflow",
    "oldDescription": "ITIL 标准变更管理流程，包含风险评估与评审组审批",
    "newDescription": "ITIL standard change management with risk assessment and review board approval",
    "replacements": [
      [
        "<bpmn:process id=\"change_request\" name=\"变更申请流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"change_request\" name=\"Change Request Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"开始\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Start\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Submit\" name=\"提交变更申请\">",
        "<bpmn:userTask id=\"Task_Submit\" name=\"Submit Change Request\">"
      ],
      [
        "<bpmn:userTask id=\"Task_ImpactAnalysis\" name=\"影响分析\">",
        "<bpmn:userTask id=\"Task_ImpactAnalysis\" name=\"Impact Analysis\">"
      ],
      [
        "<bpmn:userTask id=\"Task_ITApproval\" name=\"IT负责人审批\">",
        "<bpmn:userTask id=\"Task_ITApproval\" name=\"IT Lead Approval\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Implementation\" name=\"实施变更\">",
        "<bpmn:userTask id=\"Task_Implementation\" name=\"Implement Change\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Verification\" name=\"验证确认\">",
        "<bpmn:userTask id=\"Task_Verification\" name=\"Verification Confirmation\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Completed\" name=\"变更完成\">",
        "<bpmn:endEvent id=\"EndEvent_Completed\" name=\"Change Completed\">"
      ]
    ]
  },
  {
    "kind": "catalog",
    "key": "expense_approval",
    "oldHash": "a108f419132425011107f1913172041e110f59a0df7a3ae5fb18823a9b73dc06",
    "newHash": "56f92fc115a82bb9cc3fe6f44674be3762a9ba5a7cfb37d30b50dde18a8499b2",
    "oldName": "费用报销流程",
    "newName": "Expense Reimbursement Workflow",
    "oldDescription": "员工费用报销申请与财务审批流程",
    "newDescription": "Employee expense claims with finance approval",
    "replacements": [
      [
        "<bpmn:process id=\"expense_approval\" name=\"费用报销流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"expense_approval\" name=\"Expense Reimbursement Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"开始\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Start\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Submit\" name=\"提交报销单\">",
        "<bpmn:userTask id=\"Task_Submit\" name=\"Submit Expense Claim\">"
      ],
      [
        "<bpmn:userTask id=\"Task_ManagerApprove\" name=\"部门负责人审批\">",
        "<bpmn:userTask id=\"Task_ManagerApprove\" name=\"Department Head Approval\">"
      ],
      [
        "<bpmn:userTask id=\"Task_FinanceApprove\" name=\"财务审批\">",
        "<bpmn:userTask id=\"Task_FinanceApprove\" name=\"Finance Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"是否通过\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"Approved?\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Approved\" name=\"审批通过\">",
        "<bpmn:endEvent id=\"EndEvent_Approved\" name=\"Approval Granted\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Rejected\" name=\"审批拒绝\">",
        "<bpmn:endEvent id=\"EndEvent_Rejected\" name=\"Approval Rejected\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Approved\" name=\"通过\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Approved\">",
        "<bpmn:sequenceFlow id=\"Flow_Approved\" name=\"Approved\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Approved\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Rejected\" name=\"拒绝\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Rejected\">",
        "<bpmn:sequenceFlow id=\"Flow_Rejected\" name=\"Rejected\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Rejected\">"
      ]
    ]
  },
  {
    "kind": "catalog",
    "key": "generic_request",
    "oldHash": "b08a25140a4d42ff5572b20ac3f18c97d088059714c32173485dab2e7f0b73b3",
    "newHash": "d75b69e4e822e5abaae757fec5f2358c697bfedac875e7f196ba2077c1bb6bf1",
    "oldName": "通用申请流程",
    "newName": "General Request Workflow",
    "oldDescription": "适用于各类行政、IT、设施等通用申请场景",
    "newDescription": "General requests for administration, IT, facilities, and other services",
    "replacements": [
      [
        "<bpmn:process id=\"generic_request\" name=\"通用申请流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"generic_request\" name=\"General Request Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"开始\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Start\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Submit\" name=\"提交申请\">",
        "<bpmn:userTask id=\"Task_Submit\" name=\"Submit Request\">"
      ],
      [
        "<bpmn:userTask id=\"Task_ManagerApprove\" name=\"主管审批\">",
        "<bpmn:userTask id=\"Task_ManagerApprove\" name=\"Supervisor Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"是否通过\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"Approved?\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Approved\" name=\"审批通过\">",
        "<bpmn:endEvent id=\"EndEvent_Approved\" name=\"Approval Granted\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Rejected\" name=\"审批拒绝\">",
        "<bpmn:endEvent id=\"EndEvent_Rejected\" name=\"Approval Rejected\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Approved\" name=\"通过\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Approved\">",
        "<bpmn:sequenceFlow id=\"Flow_Approved\" name=\"Approved\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Approved\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Rejected\" name=\"拒绝\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Rejected\">",
        "<bpmn:sequenceFlow id=\"Flow_Rejected\" name=\"Rejected\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Rejected\">"
      ]
    ]
  },
  {
    "kind": "catalog",
    "key": "incident_response",
    "oldHash": "6ff3b367ca9d353d198f05b0326c6bc2b7ee8c2669e57dd33dd736ffeccc5650",
    "newHash": "df8c479bb433aca50324af97ddd2e2451fb5d6e0e3c7eb4da64295009cc0f3e6",
    "oldName": "事件响应流程",
    "newName": "Incident Response Workflow",
    "oldDescription": "ITIL 事件管理流程，包含分级、分派、解决与回顾",
    "newDescription": "ITIL incident management with triage, assignment, resolution, and review",
    "replacements": [
      [
        "<bpmn:process id=\"incident_response\" name=\"事件响应流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"incident_response\" name=\"Incident Response Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"事件发现\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Incident Detected\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Triage\" name=\"事件分级\">",
        "<bpmn:userTask id=\"Task_Triage\" name=\"Incident Triage\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Severity\" name=\"严重程度\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Severity\" name=\"Severity\">"
      ],
      [
        "<bpmn:userTask id=\"Task_MajorResponse\" name=\"重大事件响应\">",
        "<bpmn:userTask id=\"Task_MajorResponse\" name=\"Major Incident Response\">"
      ],
      [
        "<bpmn:userTask id=\"Task_AssignResolve\" name=\"分派与解决\">",
        "<bpmn:userTask id=\"Task_AssignResolve\" name=\"Assign and Resolve\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Review\" name=\"事件回顾\">",
        "<bpmn:userTask id=\"Task_Review\" name=\"Incident Review\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Closed\" name=\"事件关闭\">",
        "<bpmn:endEvent id=\"EndEvent_Closed\" name=\"Incident Closed\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Major\" name=\"重大\" sourceRef=\"Gateway_Severity\" targetRef=\"Task_MajorResponse\">",
        "<bpmn:sequenceFlow id=\"Flow_Major\" name=\"Major\" sourceRef=\"Gateway_Severity\" targetRef=\"Task_MajorResponse\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Normal\" name=\"一般\" sourceRef=\"Gateway_Severity\" targetRef=\"Task_AssignResolve\">",
        "<bpmn:sequenceFlow id=\"Flow_Normal\" name=\"Normal\" sourceRef=\"Gateway_Severity\" targetRef=\"Task_AssignResolve\">"
      ]
    ]
  },
  {
    "kind": "catalog",
    "key": "leave_request",
    "oldHash": "1b8996e78ff5afc39d0475b91f3944e93d99c08eb0369fd7a956c309cbded294",
    "newHash": "609376d450f6ad19000be3c2245d4ed2a2a4ba07e2fb264e5a505e15528fc863",
    "oldName": "请假审批流程",
    "newName": "Leave Approval Workflow",
    "oldDescription": "员工请假申请与多级审批流程",
    "newDescription": "Employee leave requests with multi-level approval",
    "replacements": [
      [
        "<bpmn:process id=\"leave_request\" name=\"请假审批流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"leave_request\" name=\"Leave Approval Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"开始\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Start\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Submit\" name=\"提交请假申请\">",
        "<bpmn:userTask id=\"Task_Submit\" name=\"Submit Leave Request\">"
      ],
      [
        "<bpmn:userTask id=\"Task_ManagerApprove\" name=\"部门经理审批\">",
        "<bpmn:userTask id=\"Task_ManagerApprove\" name=\"Department Manager Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"是否通过\">",
        "<bpmn:exclusiveGateway id=\"Gateway_Approval\" name=\"Approved?\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Approved\" name=\"审批通过\">",
        "<bpmn:endEvent id=\"EndEvent_Approved\" name=\"Approval Granted\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Rejected\" name=\"审批拒绝\">",
        "<bpmn:endEvent id=\"EndEvent_Rejected\" name=\"Approval Rejected\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Approved\" name=\"通过\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Approved\">",
        "<bpmn:sequenceFlow id=\"Flow_Approved\" name=\"Approved\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Approved\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Rejected\" name=\"拒绝\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Rejected\">",
        "<bpmn:sequenceFlow id=\"Flow_Rejected\" name=\"Rejected\" sourceRef=\"Gateway_Approval\" targetRef=\"EndEvent_Rejected\">"
      ]
    ]
  },
  {
    "kind": "catalog",
    "key": "service_request",
    "oldHash": "20a8583955e5137fd99c8adc340e550a8da0b47bb79d6e47b5628894c861e729",
    "newHash": "30f18579505e30b668f1991ce6a1fbeb6edd00ce632cc7f7e2a42aac23fb9294",
    "oldName": "服务请求流程",
    "newName": "Service Request Workflow",
    "oldDescription": "标准服务请求履行流程，支持审批与自动履行",
    "newDescription": "Standard service request fulfillment with approval and automatic fulfillment",
    "replacements": [
      [
        "<bpmn:process id=\"service_request\" name=\"服务请求流程\" isExecutable=\"true\">",
        "<bpmn:process id=\"service_request\" name=\"Service Request Workflow\" isExecutable=\"true\">"
      ],
      [
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"提交请求\">",
        "<bpmn:startEvent id=\"StartEvent_1\" name=\"Submit Request\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Review\" name=\"请求审核\">",
        "<bpmn:userTask id=\"Task_Review\" name=\"Request Review\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_NeedApproval\" name=\"是否需要审批\">",
        "<bpmn:exclusiveGateway id=\"Gateway_NeedApproval\" name=\"Approval Required?\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Approval\" name=\"审批\">",
        "<bpmn:userTask id=\"Task_Approval\" name=\"Approval\">"
      ],
      [
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"审批结果\">",
        "<bpmn:exclusiveGateway id=\"Gateway_ApprovalResult\" name=\"Approval Result\">"
      ],
      [
        "<bpmn:userTask id=\"Task_Fulfill\" name=\"履行服务\">",
        "<bpmn:userTask id=\"Task_Fulfill\" name=\"Fulfill Service\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Completed\" name=\"请求完成\">",
        "<bpmn:endEvent id=\"EndEvent_Completed\" name=\"Request Completed\">"
      ],
      [
        "<bpmn:endEvent id=\"EndEvent_Rejected\" name=\"请求拒绝\">",
        "<bpmn:endEvent id=\"EndEvent_Rejected\" name=\"Request Rejected\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_NeedApproval\" name=\"需要审批\" sourceRef=\"Gateway_NeedApproval\" targetRef=\"Task_Approval\">",
        "<bpmn:sequenceFlow id=\"Flow_NeedApproval\" name=\"Approval Required\" sourceRef=\"Gateway_NeedApproval\" targetRef=\"Task_Approval\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_AutoFulfill\" name=\"自动履行\" sourceRef=\"Gateway_NeedApproval\" targetRef=\"Task_Fulfill\">",
        "<bpmn:sequenceFlow id=\"Flow_AutoFulfill\" name=\"Automatic Fulfillment\" sourceRef=\"Gateway_NeedApproval\" targetRef=\"Task_Fulfill\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Approved\" name=\"通过\" sourceRef=\"Gateway_ApprovalResult\" targetRef=\"Task_Fulfill\">",
        "<bpmn:sequenceFlow id=\"Flow_Approved\" name=\"Approved\" sourceRef=\"Gateway_ApprovalResult\" targetRef=\"Task_Fulfill\">"
      ],
      [
        "<bpmn:sequenceFlow id=\"Flow_Rejected\" name=\"拒绝\" sourceRef=\"Gateway_ApprovalResult\" targetRef=\"EndEvent_Rejected\">",
        "<bpmn:sequenceFlow id=\"Flow_Rejected\" name=\"Rejected\" sourceRef=\"Gateway_ApprovalResult\" targetRef=\"EndEvent_Rejected\">"
      ]
    ]
  }
]
$templates$::jsonb)
    LOOP
        IF template->>'kind' = 'definition' THEN
            FOR source IN SELECT p.id,p.tenant_id,p.key,p.bpmn_xml
                FROM process_definitions p
                WHERE p.key=template->>'key' AND p.tenant_id > 0
                  AND p.name=template->>'oldName'
                  AND p.description IS NOT DISTINCT FROM template->>'oldDescription'
                  AND jsonb_typeof(p.bpmn_xml)='string'
                FOR UPDATE
            LOOP
                BEGIN
                    source_xml := convert_from(decode(source.bpmn_xml #>> '{}', 'base64'), 'UTF8');
                EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '22021' THEN
                    CONTINUE; -- Malformed/imported content is outside this migration.
                END;
                IF encode(sha256(convert_to(replace(source_xml,E'\r\n',E'\n'),'UTF8')),'hex') <> template->>'oldHash' THEN
                    CONTINUE;
                END IF;
                translated_xml := source_xml;
                FOR replacement IN SELECT value FROM jsonb_array_elements(template->'replacements') LOOP
                    translated_xml := replace(translated_xml,replacement->>0,replacement->>1);
                END LOOP;
                IF encode(sha256(convert_to(replace(translated_xml,E'\r\n',E'\n'),'UTF8')),'hex') <> template->>'newHash' THEN
                    RAISE EXCEPTION 'Unexpected built-in workflow XML: %',template->>'key';
                END IF;
                UPDATE process_definitions SET name=template->>'newName',description=template->>'newDescription',
                    bpmn_xml=to_jsonb(CASE WHEN position(E'\n' in (source.bpmn_xml #>> '{}')) > 0
                        THEN encode(convert_to(translated_xml,'UTF8'),'base64')
                        ELSE replace(encode(convert_to(translated_xml,'UTF8'),'base64'),E'\n','') END),
                    updated_at=CURRENT_TIMESTAMP
                    WHERE id=source.id AND tenant_id=source.tenant_id AND key=source.key;
                affected := affected + 1;
            END LOOP;
        ELSE
            FOR source IN SELECT w.id,w.tenant_id,w.key,w.bpmn_xml
                FROM workflow_templates w
                WHERE w.key=template->>'key' AND w.tenant_id > 0
                  AND w.name=template->>'oldName'
                  AND w.description IS NOT DISTINCT FROM template->>'oldDescription'
                  AND jsonb_typeof(w.bpmn_xml)='string'
                FOR UPDATE
            LOOP
                source_xml := source.bpmn_xml #>> '{}';
                IF encode(sha256(convert_to(replace(source_xml,E'\r\n',E'\n'),'UTF8')),'hex') <> template->>'oldHash' THEN
                    CONTINUE;
                END IF;
                translated_xml := source_xml;
                FOR replacement IN SELECT value FROM jsonb_array_elements(template->'replacements') LOOP
                    translated_xml := replace(translated_xml,replacement->>0,replacement->>1);
                END LOOP;
                IF encode(sha256(convert_to(replace(translated_xml,E'\r\n',E'\n'),'UTF8')),'hex') <> template->>'newHash' THEN
                    RAISE EXCEPTION 'Unexpected built-in catalog XML: %',template->>'key';
                END IF;
                UPDATE workflow_templates SET name=template->>'newName',description=template->>'newDescription',
                    bpmn_xml=to_jsonb(translated_xml),updated_at=CURRENT_TIMESTAMP
                    WHERE id=source.id AND tenant_id=source.tenant_id AND key=source.key;
                affected := affected + 1;
            END LOOP;
        END IF;
    END LOOP;
    RAISE NOTICE 'English built-in workflow metadata updated: % rows',affected;
END
$migration$;
