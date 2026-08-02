import React, { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import type { AgentNodeData, ToolNodeData, VariableNodeData, ClassifierNodeData, StartNodeData } from './types';

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

const header = (accent: string): React.CSSProperties => ({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '9px 14px',
  background: accent === '#4b6cb7'
    ? 'linear-gradient(to right, #4b6cb7, #182848)'
    : accent === '#2e7d32'
      ? 'linear-gradient(to right, #43a047, #2e7d32)'
      : accent === '#e65100'
        ? 'linear-gradient(to right, #ef6c00, #e65100)'
        : `linear-gradient(to right, ${accent}, ${accent}dd)`,
  color: '#fff',
  borderBottom: 'none',
});

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
      <div style={header(c)}>
        <Fa icon="fa-play" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        <span style={row}>用户问题入口</span>
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
      <div style={header(c)}>
        <Fa icon="fa-robot" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        {data.agentName && <span style={row}>名称: {data.agentName}</span>}
        {data.outputVar && <span style={row}>输出: {`{{${data.outputVar}}}`}</span>}
        {data.condition && <span style={tag('#fff3e0', '#e65100')}>条件: {data.condition}</span>}
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
      <div style={header(c)}>
        <Fa icon="fa-wrench" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        {data.toolName && <span style={row}>工具: {data.toolName}</span>}
        {data.outputVar && <span style={row}>输出: {`{{${data.outputVar}}}`}</span>}
        {data.condition && <span style={tag('#fff3e0', '#e65100')}>条件: {data.condition}</span>}
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
  const c = '#e65100';
  return (
    <div style={card}>
      <div style={header(c)}>
        <Fa icon="fa-database" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        <code style={{ ...row, fontFamily: 'Consolas, Monaco, monospace', fontSize: 12, color: '#e65100' }}>
          {`{{${data.varName}}}`}
        </code>
      </div>
      <Handle type="source" position={Position.Right} style={handleStyle(c)} />
    </div>
  );
});

// ============================================================
// ClassifierNode
// ============================================================

export const ClassifierNode = memo(function ClassifierNode({ data }: { data: ClassifierNodeData }) {
  const c = '#4b6cb7';
  const cats = data.categories || [];
  return (
    <div style={card}>
      <div style={header(c)}>
        <Fa icon="fa-code-branch" /> <span style={{ fontWeight: 600, fontSize: 13 }}>{data.label}</span>
      </div>
      <div style={body}>
        <span style={row}>输出: {`{{${data.outputVar || 'intent'}}}`}</span>
        {cats.length > 0 && (
          <div style={{ marginTop: 4, display: 'flex', flexWrap: 'wrap', gap: 3 }}>
            {cats.map((cat, i) => (
              <span key={i} style={tag('#e8eaf6', '#4b6cb7')}>{cat.name}</span>
            ))}
          </div>
        )}
      </div>
      <Handle type="target" position={Position.Left} style={handleStyle(c)} />
      <Handle type="source" position={Position.Right} style={handleStyle(c)} />
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
  classifier: ClassifierNode,
};
