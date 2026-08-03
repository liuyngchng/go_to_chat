// ============================================================
// 工作流设计工具的 TypeScript 类型定义
// 专注头脑风暴 + 自描述导出，不依赖后端
// ============================================================

// ============================================================
// 节点类型
// ============================================================

export type NodeType = 'start' | 'agent' | 'tool' | 'variable' | 'branch' | 'note';

// ============================================================
// React Flow 内部节点 data 类型
// 每种节点都带 purpose 字段，便于 AI 理解
// ============================================================

export interface StartNodeData {
  nodeType: 'start';
  label: string;
  purpose: string;
}

export interface AgentNodeData {
  nodeType: 'agent';
  label: string;
  purpose: string;
  agentName: string;
  inputTemplate: string;
  outputVar: string;
  parallelGroup: string;
}

export interface ToolNodeData {
  nodeType: 'tool';
  label: string;
  purpose: string;
  toolName: string;
  toolParams: string;
  outputVar: string;
  parallelGroup: string;
}

export interface VariableNodeData {
  nodeType: 'variable';
  label: string;
  purpose: string;
  varName: string;
  varDesc: string;
}

export interface BranchNodeData {
  nodeType: 'branch';
  label: string;
  purpose: string;
  /** 分支依据的变量名，如 intent / user_query */
  inputVar: string;
}

export interface NoteNodeData {
  nodeType: 'note';
  label: string;
  purpose: string;
  content: string;
  color: string;
}

/** 所有节点 data 的联合类型 */
export type AppNodeData =
  | StartNodeData
  | AgentNodeData
  | ToolNodeData
  | VariableNodeData
  | BranchNodeData
  | NoteNodeData;

// ============================================================
// 自描述导出格式 — 用于导出 JSON，AI 可直接读懂
// ============================================================

/** 导出格式中的单个节点 */
export interface DesignDocNode {
  id: string;
  type: NodeType;
  label: string;
  purpose: string;

  // Agent 相关
  agentName?: string;
  inputTemplate?: string;
  outputVar?: string;
  parallelGroup?: string;

  // Tool 相关
  toolName?: string;
  toolParams?: string;

  // Variable 相关
  varName?: string;
  varDesc?: string;

  // Branch 相关（条件分支）
  branchInputVar?: string;

  // Note 相关
  content?: string;
  color?: string;
}

/** 导出格式中的边 */
export interface DesignDocEdge {
  from: string;
  to: string;
  /** 分支条件：匹配时走此边，default 表示兜底，无值表示无条件 */
  condition?: string;
  /** @deprecated 使用 condition */
  label?: string;
}

/** 自描述工作流设计文档 */
export interface DesignDoc {
  /** schema 版本，AI 可据此判断格式 */
  _schema: 'workflow-design-doc/v1';
  /** 工作流名称 */
  name: string;
  /** 简短描述 */
  description: string;
  /** 设计意图说明 — AI 读这个就知道整体要做什么 */
  purpose: string;
  /** 元数据 */
  metadata: {
    createdAt: string;
    tags: string[];
  };
  /** 顶层 I/O 摘要（AI 友好） */
  _summary?: {
    /** 工作流入参列表 */
    inputs: string[];
    /** 工作流出参列表 */
    outputs: string[];
  };
  /** 节点列表 */
  nodes: DesignDocNode[];
  /** 边列表 */
  edges: DesignDocEdge[];
}

// ============================================================
// 注意：变量节点可引用上游节点的 outputVar，
// 模板中使用 {{变量名}} 引用。
// 本工具不预设任何系统变量，变量名由用户自由定义。
// ============================================================