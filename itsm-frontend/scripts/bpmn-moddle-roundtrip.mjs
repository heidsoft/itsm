#!/usr/bin/env node
/**
 * 真实 bpmn-moddle 往返校验脚本。
 *
 * 为什么单独成脚本而不是写在 Jest 里：bpmn-moddle v10 与其依赖（min-dash /
 * moddle-xml / saxen）只发布 ESM，而 next/jest 会注入
 * `/node_modules/(?!.pnpm)(?!(geist)/)` 的 transformIgnorePatterns，
 * 任何追加规则都排在它之后，node_modules 里的 ESM 无法被转换。
 * 因此由 Jest 传入 descriptor、由本脚本用真实 moddle 跑出证据。
 *
 * stdin : { "descriptor": <itsm moddle descriptor> }
 * stdout: 校验报告 JSON
 */
import { readFileSync } from 'node:fs';
import { BpmnModdle } from 'bpmn-moddle';

const TARGET_ELEMENT = {
  'bpmn:UserTask': 'UT',
  'bpmn:ServiceTask': 'ST',
};

const DIAGRAM = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  id="D" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="P" isExecutable="true">
    <bpmn:userTask id="UT" name="审批"/>
    <bpmn:serviceTask id="ST" name="抄送" implementation="cc_handler"/>
    <bpmn:exclusiveGateway id="GW"/>
    <bpmn:endEvent id="E1"/>
    <bpmn:sequenceFlow id="F1" sourceRef="UT" targetRef="GW"/>
    <bpmn:sequenceFlow id="F2" sourceRef="GW" targetRef="E1"/>
  </bpmn:process>
</bpmn:definitions>`;

/** 带 operationRef 的文档，用于验证「重复声明标准属性名」会毁掉整个元素 */
const DIAGRAM_WITH_OPERATION_REF = DIAGRAM.replace(
  'implementation="cc_handler"',
  'implementation="cc_handler" operationRef="cc_handler"'
);

function sampleFor(fieldType, name) {
  if (name === 'approvalThreshold') return 7;
  if (fieldType === 'Integer') return 7;
  if (fieldType === 'Boolean') return false;
  return 'sample-value';
}

function flowElements(definitions) {
  const rootElements = definitions.rootElements ?? [];
  const process = rootElements.find(element => element.$type === 'bpmn:Process');
  return process?.flowElements ?? [];
}

function elementById(definitions, id) {
  return flowElements(definitions).find(element => element.id === id);
}

function moddleOf(descriptor) {
  // moddle Registry 会就地把 descriptor 的属性名改写成限定的 "itsm:<name>"，
  // 因此每个 moddle 实例都必须拿到自己的副本，否则后续场景会读到被污染的名字。
  return new BpmnModdle({ itsm: structuredClone(descriptor) });
}

async function writeReadBack(descriptor, elementId, name, value) {
  const moddle = moddleOf(descriptor);
  const { rootElement: definitions } = await moddle.fromXML(DIAGRAM);
  const target = elementById(definitions, elementId);
  if (!target) return { failure: `element ${elementId} missing` };

  target.set(name, value);
  const { xml } = await moddle.toXML(definitions, { format: true });
  const reloaded = await moddle.fromXML(xml);
  const restored = elementById(reloaded.rootElement, elementId) ?? {};
  return {
    emitted: xml.includes(`${name}=`),
    namespaced: xml.includes(`itsm:${name}=`),
    readBack: restored[name] === undefined ? null : restored[name],
    parkedInAttrs: Boolean(restored.$attrs && name in restored.$attrs),
    failure: null,
  };
}

async function serializeAttempt(descriptor, elementId, name, value) {
  const moddle = moddleOf(descriptor);
  const { rootElement: definitions } = await moddle.fromXML(DIAGRAM);
  const target = elementById(definitions, elementId);
  target.set(name, value);
  try {
    const { xml } = await moddle.toXML(definitions, { format: true });
    const reloaded = await moddle.fromXML(xml);
    const restored = elementById(reloaded.rootElement, elementId) ?? {};
    return {
      serialized: true,
      xmlFragment: xml.split('\n').map(l => l.trim()).find(l => l.includes(`id="${elementId}"`)) ?? null,
      readBack: restored[name] === undefined ? null : restored[name],
      warnings: (reloaded.warnings ?? []).map(w => String(w.message ?? w).slice(0, 120)),
      failure: null,
    };
  } catch (error) {
    return { serialized: false, failure: String(error?.message ?? error).split('\n')[0] };
  }
}

const input = JSON.parse(readFileSync(0, 'utf8'));
const descriptor = input.descriptor;

const declared = [];
for (const type of descriptor.types ?? []) {
  const elementId = TARGET_ELEMENT[type.extends[0]];
  for (const property of type.properties) {
    declared.push({
      target: type.extends[0],
      elementId,
      name: property.name,
      fieldType: property.type,
      result: await writeReadBack(descriptor, elementId, property.name, sampleFor(property.type, property.name)),
    });
  }
}

// 正确的写入形状：BPMNDesigner.normalizeNodeProperties 产出的就是这些
const correctDocumentation = await (async () => {
  const moddle = moddleOf(descriptor);
  const { rootElement: definitions } = await moddle.fromXML(DIAGRAM);
  const element = elementById(definitions, 'UT');
  element.set('documentation', [moddle.create('bpmn:Documentation', { text: '节点说明' })]);
  const { xml } = await moddle.toXML(definitions, { format: true });
  const reloaded = await moddle.fromXML(xml);
  const restored = elementById(reloaded.rootElement, 'UT');
  return {
    xmlHasElement: xml.includes('<bpmn:documentation>节点说明</bpmn:documentation>'),
    readBackText: restored.documentation?.[0]?.text ?? null,
  };
})();

const correctCondition = await (async () => {
  const moddle = moddleOf(descriptor);
  const { rootElement: definitions } = await moddle.fromXML(DIAGRAM);
  const element = elementById(definitions, 'F1');
  element.set(
    'conditionExpression',
    moddle.create('bpmn:FormalExpression', { body: '${variables["amount"] > 10000}' })
  );
  const { xml } = await moddle.toXML(definitions, { format: true });
  const reloaded = await moddle.fromXML(xml);
  return {
    readBackBody: elementById(reloaded.rootElement, 'F1').conditionExpression?.body ?? null,
    warnings: (reloaded.warnings ?? []).map(w => String(w.message ?? w).slice(0, 120)),
  };
})();

const correctDefault = await (async () => {
  const moddle = moddleOf(descriptor);
  const { rootElement: definitions } = await moddle.fromXML(DIAGRAM);
  const gateway = elementById(definitions, 'GW');
  gateway.set('default', elementById(definitions, 'F2'));
  const { xml } = await moddle.toXML(definitions, { format: true });
  const reloaded = await moddle.fromXML(xml);
  return {
    xmlHasRef: xml.includes('default="F2"'),
    readBackId: elementById(reloaded.rootElement, 'GW').default?.id ?? null,
  };
})();

// 在 operationRef 上重复声明标准属性名的后果（曾经的错误修法）
const collision = await (async () => {
  const poisoned = structuredClone(descriptor);
  poisoned.types.push({
    name: 'BadCollision',
    extends: ['bpmn:ServiceTask'],
    properties: [{ name: 'operationRef', isAttr: true, type: 'String' }],
  });
  const moddle = new BpmnModdle({ itsm: poisoned });
  const { rootElement: definitions, warnings } = await moddle.fromXML(DIAGRAM_WITH_OPERATION_REF);
  return {
    serviceTaskParsed: Boolean(elementById(definitions, 'ST')),
    warnings: (warnings ?? []).map(w => String(w.message ?? w).slice(0, 140)),
  };
})();

const report = {
  declared,
  checks: {
    followUpDate: await writeReadBack(descriptor, 'UT', 'followUpDate', 'PT1H'),
    documentationAsString: await serializeAttempt(descriptor, 'UT', 'documentation', '节点说明'),
    documentationAsObject: await serializeAttempt(descriptor, 'UT', 'documentation', { text: 'x' }),
    conditionAsPlainObject: await serializeAttempt(descriptor, 'F1', 'conditionExpression', {
      type: 'bpmn:FormalExpression',
      body: '${a > 1}',
    }),
    defaultAsString: await serializeAttempt(descriptor, 'GW', 'default', 'F2'),
    operationRefAsString: await serializeAttempt(descriptor, 'ST', 'operationRef', 'cc_handler'),
    correctDocumentation,
    correctCondition,
    correctDefault,
    standardNameCollision: collision,
  },
};

process.stdout.write(JSON.stringify(report, null, 2));
