import React, { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import { Play, Bot, Wrench, Database, GitBranch } from 'lucide-react';
import type { AgentNodeData, ToolNodeData, VariableNodeData, ClassifierNodeData, StartNodeData } from './types';

// ============================================================
// 节点卡片统一样式
// ============================================================

const card = (accent: string): React.CSSProperties => ({
  background: '#1c1c2a',
  borderRadius: 12,
  border: '1px solid #2a2a3d',
  borderLeft: `3px solid ${accent}`,
  minWidth: 200,
  boxShadow: '0 4px 24px rgba(0,0,0,0.25), 0 0 0 1px rgba(255,255,255,0.03)',
  overflow: 'hidden',
});

const header = (accent: string): React.CSSProperties => ({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '10px 14px',
  background: `${accent}0f`,
  borderBottom: '1px solid #2a2a3d',
});

const headerIcon = (accent: string): React.CSSProperties => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 28,
  height: 28,
  borderRadius: 7,
  background: `${accent}1e`,
  color: accent,
});

const headerLabel: React.CSSProperties = {
  fontWeight: 600,
  fontSize: 13,
  color: '#e2e4ed',
};

const body: React.CSSProperties = {
  padding: '10px 14px',
  display: 'flex',
  flexDirection: 'column',
  gap: 4,
};

const row: React.CSSProperties = {
  fontSize: 11,
  color: '#8b8fa5',
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

// ============================================================
// Handle 统一样式
// ============================================================

const handleStyle = (color: string): React.CSSProperties => ({
  width: 9,
  height: 9,
  border: `2px solid #2a2a3d`,
  background: color,
  transition: 'all 0.15s',
});

// ============================================================
// StartNode
// ============================================================

export const StartNode = memo(function StartNode({ data }: { data: StartNodeData }) {
  const c = '#22c55e';
  return (
    <div style={card(c)}>
      <div style={header(c)}>
        <div style={headerIcon(c)}><Play size={15} fill={c} /></div>
        <span style={headerLabel}>{data.label}</span>
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
  const c = '#3b82f6';
  return (
    <div style={card(c)}>
      <div style={header(c)}>
        <div style={headerIcon(c)}><Bot size={15} /></div>
        <span style={headerLabel}>{data.label}</span>
      </div>
      <div style={body}>
        {data.agentName && <span style={row}>名称: {data.agentName}</span>}
        {data.outputVar && <span style={row}>输出: {`{{${data.outputVar}}}`}</span>}
        {data.condition && <span style={tag('#fbbf2420', '#fbbf24')}>条件: {data.condition}</span>}
        {data.parallelGroup && <span style={tag('#a78bfa20', '#a78bfa')}>∥ {data.parallelGroup}</span>}
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
  const c = '#14b8a6';
  return (
    <div style={card(c)}>
      <div style={header(c)}>
        <div style={headerIcon(c)}><Wrench size={15} /></div>
        <span style={headerLabel}>{data.label}</span>
      </div>
      <div style={body}>
        {data.toolName && <span style={row}>工具: {data.toolName}</span>}
        {data.outputVar && <span style={row}>输出: {`{{${data.outputVar}}}`}</span>}
        {data.condition && <span style={tag('#fbbf2420', '#fbbf24')}>条件: {data.condition}</span>}
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
  const c = '#f59e0b';
  return (
    <div style={card(c)}>
      <div style={header(c)}>
        <div style={headerIcon(c)}><Database size={15} /></div>
        <span style={headerLabel}>{data.label}</span>
      </div>
      <div style={body}>
        <code style={{ ...row, fontFamily: 'JetBrains Mono, Fira Code, monospace', fontSize: 12, color: '#f59e0b' }}>
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
  const c = '#8b5cf6';
  const cats = data.categories || [];
  return (
    <div style={card(c)}>
      <div style={header(c)}>
        <div style={headerIcon(c)}><GitBranch size={15} /></div>
        <span style={headerLabel}>{data.label}</span>
      </div>
      <div style={body}>
        <span style={row}>输出: {`{{${data.outputVar || 'intent'}}}`}</span>
        {cats.length > 0 && (
          <div style={{ marginTop: 4, display: 'flex', flexWrap: 'wrap', gap: 3 }}>
            {cats.map((cat, i) => (
              <span key={i} style={tag('#8b5cf620', '#c4b5fd')}>{cat.name}</span>
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
