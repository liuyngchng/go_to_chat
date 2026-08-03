import React, { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import type { AgentNodeData, ToolNodeData, VariableNodeData, BranchNodeData, StartNodeData, NoteNodeData } from './types';

// ============================================================
// Font Awesome 图标
// ============================================================

const Fa = ({ icon, style }: { icon: string; style?: React.CSSProperties }) => (
  <i className={`fas ${icon}`} style={{ fontSize: 14, ...style }} />
);

// ============================================================
// 节点卡片统一样式（匹配 g/ 的 .wf-node-card）
// ============================================================

const card: React.CSSProperties = {
  background: '#fff',
  borderRadius: 10,
  border: '2px solid #e0e3e8',
  minWidth: 200,
  boxShadow: '0 2px 6px rgba(0,0,0,0.08)',
  overflow: 'hidden',
  cursor: 'pointer',
  transition: 'border-color 0.2s, box-shadow 0.2s',
};

const header: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '9px 14px',
  background: 'linear-gradient(to right, #4b6cb7, #182848)',
  color: '#fff',
  borderBottom: 'none',
};

const body: React.CSSProperties = {
  padding: '10px 14px',
  display: 'flex',
  flexDirection: 'column',
  gap: 4,
};

const row: React.CSSProperties = {
  fontSize: 11,
  color: '#666',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  lineHeight: 1.5,
};

const tag = (bg: string, fg: string): React.CSSProperties => ({
  display: 'inline-block',
  padding: '2px 7px',
  borderRadius: 4,
  background: bg,
  color: fg,
  fontSize: 10,
  fontWeight: 500,
  marginTop: 2,
  alignSelf: 'flex-start',
});

const handleStyle = (color: string): React.CSSProperties => ({
  width: 12,
  height: 12,
  border: '2px solid #fff',
  background: color,
});

// ============================================================
// StartNode
// ============================================================

export const StartNode = memo(function StartNode({ data }: { data: StartNodeData }) {
  const c = '#4b6cb7';
  return (
    <div style={card}>
      <div style={header}>
        <Fa icon="fa-play" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        <span style={row}>用户问题入口</span>
        {data.purpose && <span style={{ ...row, color: '#888', fontStyle: 'italic' }}>💡 {data.purpose}</span>}
      </div>
      <Handle type="source" position={Position.Right} style={handleStyle(c)} />
    </div>
  );
});

// ============================================================
// AgentNode
// ============================================================

export const AgentNode = memo(function AgentNode({ data }: { data: AgentNodeData }) {
  const c = '#4b6cb7';
  return (
    <div style={card}>
      <div style={header}>
        <Fa icon="fa-robot" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        {data.outputVar && <span style={row}>输出: {`{{${data.outputVar}}}`}</span>}
        {data.purpose && <span style={{ ...row, color: '#888', fontStyle: 'italic' }}>💡 {data.purpose}</span>}
        {data.parallelGroup && <span style={tag('#e8eaf6', '#4b6cb7')}>∥ {data.parallelGroup}</span>}
      </div>
      <Handle type="target" position={Position.Left} style={handleStyle(c)} />
      <Handle type="source" position={Position.Right} style={handleStyle(c)} />
    </div>
  );
});

// ============================================================
// ToolNode
// ============================================================

export const ToolNode = memo(function ToolNode({ data }: { data: ToolNodeData }) {
  const c = '#4b6cb7';
  return (
    <div style={card}>
      <div style={header}>
        <Fa icon="fa-wrench" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        {data.toolName && <span style={row}>工具: {data.toolName}</span>}
        {data.outputVar && <span style={row}>输出: {`{{${data.outputVar}}}`}</span>}
        {data.purpose && <span style={{ ...row, color: '#888', fontStyle: 'italic' }}>💡 {data.purpose}</span>}
      </div>
      <Handle type="target" position={Position.Left} style={handleStyle(c)} />
      <Handle type="source" position={Position.Right} style={handleStyle(c)} />
    </div>
  );
});

// ============================================================
// VariableNode
// ============================================================

export const VariableNode = memo(function VariableNode({ data }: { data: VariableNodeData }) {
  const c = '#4b6cb7';
  return (
    <div style={card}>
      <div style={header}>
        <Fa icon="fa-database" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        <code style={{ ...row, fontFamily: 'Consolas, Monaco, monospace', fontSize: 12, color: '#4b6cb7' }}>
          {`{{${data.varName}}}`}
        </code>
        {data.purpose && <span style={{ ...row, color: '#888', fontStyle: 'italic' }}>💡 {data.purpose}</span>}
      </div>
      <Handle type="source" position={Position.Right} style={handleStyle(c)} />
    </div>
  );
});

// ============================================================
// BranchNode（条件分支 — switch/case）
// ============================================================

export const BranchNode = memo(function BranchNode({ data }: { data: BranchNodeData }) {
  const c = '#4b6cb7';
  return (
    <div style={{
      ...card,
      borderLeft: `6px solid #e67e22`,
    }}>
      <div style={header}>
        <Fa icon="fa-code-branch" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
        <span style={{ marginLeft: 'auto', fontSize: 9, background: 'rgba(255,255,255,0.2)', padding: '2px 7px', borderRadius: 4 }}>
          SWITCH
        </span>
      </div>
      <div style={body}>
        {data.inputVar && <span style={row}>依据: {`{{${data.inputVar}}}`}</span>}
        {data.purpose && <span style={{ ...row, color: '#888', fontStyle: 'italic' }}>💡 {data.purpose}</span>}
      </div>
      <Handle type="target" position={Position.Left} style={handleStyle(c)} />
      <Handle type="source" position={Position.Right} style={handleStyle(c)} />
    </div>
  );
});

// ============================================================
// NoteNode（便签/注释节点）
// ============================================================

export const NoteNode = memo(function NoteNode({ data }: { data: NoteNodeData }) {
  const noteColors: Record<string, { bg: string; border: string; text: string }> = {
    yellow: { bg: '#fff9c4', border: '#f9a825', text: '#5d4037' },
    green: { bg: '#c8e6c9', border: '#43a047', text: '#1b5e20' },
    blue: { bg: '#bbdefb', border: '#1e88e5', text: '#0d47a1' },
    pink: { bg: '#f8bbd0', border: '#e91e63', text: '#880e4f' },
    purple: { bg: '#e1bee7', border: '#8e24aa', text: '#4a148c' },
  };
  const c = noteColors[data.color] || noteColors.yellow;

  return (
    <div style={{
      background: c.bg,
      border: `2px solid ${c.border}`,
      borderRadius: 10,
      minWidth: 180,
      maxWidth: 280,
      padding: '12px 16px',
      boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
      cursor: 'pointer',
      fontFamily: 'inherit',
    }}>
      <div style={{ fontWeight: 600, fontSize: 12, color: c.text, marginBottom: 4 }}>
        {data.label}
      </div>
      <div style={{ fontSize: 11, color: c.text, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>
        {data.content || '（空）'}
      </div>
    </div>
  );
});

// ============================================================
// 注册表
// ============================================================

export const nodeTypes = {
  start: StartNode,
  agent: AgentNode,
  tool: ToolNode,
  variable: VariableNode,
  branch: BranchNode,
  note: NoteNode,
};
