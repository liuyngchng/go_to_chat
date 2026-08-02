// ============================================================
// 与 Go 后端 model.WorkflowDef / WorkflowNode 对应的 TypeScript 类型
// 参考: g/internal/model/types.go
// ============================================================

/** 节点类型 */
export type NodeType = 'start' | 'agent' | 'tool' | 'variable' | 'classifier';

/** 意图分类类别 */
export interface IntentCategory {
  name: string;        // 类别标识，如 "emergency"
  description: string; // 类别描述
  keywords: string[];  // 关键词列表
}

/** 意图分类器定义 (对应 Go ClassifierDef) */
export interface ClassifierDef {
  prompt: string;
  output_var: string;       // 默认 "intent"
  categories: IntentCategory[];
}

/** 工作流节点 (对应 Go WorkflowNode) */
export interface WorkflowNode {
  id: string;
  agent_id: number;
  agent_name: string;
  input_template: string;
  output_var: string;
  order_index: number;
  is_final: boolean;
  condition: string;
  next_nodes: string[];
  parallel_group: string;
}

/** 工作流定义 (对应 Go WorkflowDef) */
export interface WorkflowDef {
  name: string;
  description: string;
  classifier: ClassifierDef | null;
  nodes: WorkflowNode[];
}

/** 自定义节点的 data 结构 */
export interface AgentNodeData {
  nodeType: 'agent';
  label: string;
  agentId: number;
  agentName: string;
  inputTemplate: string;
  outputVar: string;
  condition: string;
  parallelGroup: string;
}

export interface ToolNodeData {
  nodeType: 'tool';
  label: string;
  toolName: string;
  toolParams: string;
  outputVar: string;
  condition: string;
  parallelGroup: string;
}

export interface VariableNodeData {
  nodeType: 'variable';
  label: string;
  varName: string;
  varDesc: string;
}

export interface ClassifierNodeData {
  nodeType: 'classifier';
  label: string;
  outputVar: string;
  prompt: string;
  categories: IntentCategory[];
}

export interface StartNodeData {
  nodeType: 'start';
  label: string;
}

/** 所有节点 data 的联合类型 */
export type AppNodeData =
  | AgentNodeData
  | ToolNodeData
  | VariableNodeData
  | ClassifierNodeData
  | StartNodeData;

/** 系统变量列表 */
export const SYSTEM_VARS = [
  { name: 'sys.user_query', description: '用户当前问题' },
  { name: 'sys.history', description: '历史对话记录' },
  { name: 'sys.cur_date', description: '当前日期 (YYYY-MM-DD)' },
  { name: 'sys.cur_week', description: '当前星期几（中文）' },
  { name: 'sys.kb_context', description: '知识库检索结果' },
  { name: 'sys.intent', description: '意图分类结果' },
];

/** 兼容旧版变量名 */
export const LEGACY_VARS = [
  { name: 'user_query', description: '用户当前问题（旧版）' },
  { name: 'history', description: '历史对话记录（旧版）' },
  { name: 'cur_date', description: '当前日期（旧版）' },
  { name: 'cur_week', description: '当前星期（旧版）' },
  { name: 'intent', description: '意图分类结果（旧版）' },
];
