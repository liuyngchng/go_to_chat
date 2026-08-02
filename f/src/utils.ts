import type { Node, Edge } from '@xyflow/react';
import type { AppNodeData, WorkflowDef, WorkflowNode, ClassifierDef } from './types';

/**
 * 将 React Flow 的 nodes + edges 转换为 Go 后端 WorkflowDef 格式
 */
export function exportToWorkflowDef(
  name: string,
  description: string,
  nodes: Node[],
  edges: Edge[]
): WorkflowDef {
  // 提取 classifier（从 classifier 类型节点）
  const classifierNode = nodes.find((n) => {
    const d = n.data as unknown as AppNodeData;
    return d.nodeType === 'classifier';
  });
  let classifier: ClassifierDef | null = null;
  if (classifierNode) {
    const cd = classifierNode.data as unknown as AppNodeData & { nodeType: 'classifier' };
    classifier = {
      prompt: cd.prompt || '',
      output_var: cd.outputVar || 'intent',
      categories: cd.categories || [],
    };
  }

  // 构建邻接表
  const adjacency = new Map<string, string[]>();
  for (const e of edges) {
    const targets = adjacency.get(e.source) || [];
    targets.push(e.target);
    adjacency.set(e.source, targets);
  }

  // 被引用的节点 ID 集合
  const referenced = new Set<string>();
  for (const e of edges) {
    referenced.add(e.target);
  }

  // 转换节点（跳过 start 和 classifier）
  const workflowNodes: WorkflowNode[] = [];
  let orderIdx = 0;

  for (const node of nodes) {
    const data = node.data as unknown as AppNodeData;
    if (data.nodeType === 'start') continue;
    if (data.nodeType === 'classifier') continue;

    const nextNodes = adjacency.get(node.id) || [];
    const isFinal = !referenced.has(node.id) || nextNodes.length === 0;

    let agentId = 0;
    let agentName = '';
    let inputTemplate = '';
    let outputVar = '';
    let condition = '';
    let parallelGroup = '';

    switch (data.nodeType) {
      case 'agent':
        agentId = data.agentId || 0;
        agentName = data.agentName || '';
        inputTemplate = data.inputTemplate || '';
        outputVar = data.outputVar || '';
        condition = data.condition || '';
        parallelGroup = data.parallelGroup || '';
        break;
      case 'tool':
        agentId = 0;
        agentName = `[工具] ${data.toolName || ''}`;
        inputTemplate = data.toolParams || '';
        outputVar = data.outputVar || '';
        condition = data.condition || '';
        parallelGroup = data.parallelGroup || '';
        break;
      case 'variable':
        agentId = 0;
        agentName = `[变量] ${data.varName || ''}`;
        inputTemplate = '';
        outputVar = data.varName || '';
        condition = '';
        parallelGroup = '';
        break;
    }

    workflowNodes.push({
      id: node.id,
      agent_id: agentId,
      agent_name: agentName,
      input_template: inputTemplate,
      output_var: outputVar,
      order_index: orderIdx++,
      is_final: isFinal,
      condition: condition,
      next_nodes: nextNodes,
      parallel_group: parallelGroup,
    });
  }

  return {
    name: name || '未命名工作流',
    description: description || '',
    classifier,
    nodes: workflowNodes,
  };
}

/**
 * 从 WorkflowDef JSON 还原为 React Flow nodes + edges
 */
export function importFromWorkflowDef(
  def: WorkflowDef
): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  // 添加 start 节点
  nodes.push({
    id: '__start__',
    type: 'start',
    position: { x: 50, y: 200 },
    data: { nodeType: 'start', label: '开始' },
  });

  // 添加 classifier 节点
  if (def.classifier) {
    nodes.push({
      id: '__classifier__',
      type: 'classifier',
      position: { x: 300, y: 200 },
      data: {
        nodeType: 'classifier',
        label: '意图分类',
        outputVar: def.classifier.output_var || 'intent',
        prompt: def.classifier.prompt || '',
        categories: def.classifier.categories || [],
      },
    });
    edges.push({
      id: 'e__start__classifier',
      source: '__start__',
      target: '__classifier__',
    });
  }

  // 添加工作流节点
  for (let i = 0; i < def.nodes.length; i++) {
    const wn = def.nodes[i];
    const x = def.classifier ? 600 : 350;
    const y = 100 + i * 120;

    if (wn.agent_name?.startsWith('[工具]')) {
      nodes.push({
        id: wn.id || `node_${i}`,
        type: 'tool',
        position: { x, y },
        data: {
          nodeType: 'tool',
          label: wn.agent_name.replace('[工具] ', ''),
          toolName: wn.agent_name.replace('[工具] ', ''),
          toolParams: wn.input_template,
          outputVar: wn.output_var,
          condition: wn.condition,
          parallelGroup: wn.parallel_group,
        },
      });
    } else if (wn.agent_name?.startsWith('[变量]')) {
      nodes.push({
        id: wn.id || `node_${i}`,
        type: 'variable',
        position: { x, y },
        data: {
          nodeType: 'variable',
          label: wn.agent_name.replace('[变量] ', ''),
          varName: wn.output_var,
          varDesc: '',
        },
      });
    } else {
      nodes.push({
        id: wn.id || `node_${i}`,
        type: 'agent',
        position: { x, y },
        data: {
          nodeType: 'agent',
          label: wn.agent_name || `节点 ${wn.id}`,
          agentId: wn.agent_id,
          agentName: wn.agent_name,
          inputTemplate: wn.input_template,
          outputVar: wn.output_var,
          condition: wn.condition,
          parallelGroup: wn.parallel_group,
        },
      });
    }
  }

  // 添加边
  for (const wn of def.nodes) {
    for (const target of wn.next_nodes || []) {
      edges.push({
        id: `e_${wn.id}_${target}`,
        source: wn.id,
        target,
      });
    }
  }

  return { nodes, edges };
}

/**
 * 下载 JSON 文件
 */
export function downloadJSON(data: WorkflowDef, filename: string) {
  const json = JSON.stringify(data, null, 2);
  const blob = new Blob([json], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename.endsWith('.json') ? filename : `${filename}.json`;
  a.click();
  URL.revokeObjectURL(url);
}
