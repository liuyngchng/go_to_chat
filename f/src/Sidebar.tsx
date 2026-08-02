import React, { type DragEvent } from 'react';
import { Play, Bot, Wrench, Database, GitBranch } from 'lucide-react';

// ============================================================
// 节点类型定义
// ============================================================

const nodeTypes = [
  { type: 'start', label: '开始', Icon: Play, color: '#22c55e', desc: '工作流入口' },
  { type: 'agent', label: 'AI Agent', Icon: Bot, color: '#3b82f6', desc: 'AI 智能体节点' },
  { type: 'tool', label: '工具', Icon: Wrench, color: '#14b8a6', desc: '自定义工具调用' },
  { type: 'variable', label: '系统变量', Icon: Database, color: '#f59e0b', desc: '输出系统变量值' },
  { type: 'classifier', label: '意图分类', Icon: GitBranch, color: '#8b5cf6', desc: 'LLM 意图分类路由' },
] as const;

// ============================================================
// 样式
// ============================================================

const sidebarStyle: React.CSSProperties = {
  width: 220,
  background: '#14141f',
  borderRight: '1px solid #1e1e30',
  display: 'flex',
  flexDirection: 'column',
  overflow: 'hidden',
};

const titleBlock: React.CSSProperties = {
  padding: '18px 16px 14px',
  borderBottom: '1px solid #1e1e30',
  display: 'flex',
  alignItems: 'center',
  gap: 8,
};

const titleText: React.CSSProperties = {
  fontSize: 14,
  fontWeight: 700,
  color: '#e2e4ed',
  letterSpacing: '-0.01em',
};

const scroller: React.CSSProperties = {
  flex: 1,
  overflowY: 'auto',
  padding: '8px 10px',
};

const groupLabel: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 700,
  color: '#5b5d78',
  textTransform: 'uppercase',
  letterSpacing: '0.08em',
  margin: '12px 6px 6px',
};

const item = (color: string): React.CSSProperties => ({
  display: 'flex',
  alignItems: 'center',
  gap: 10,
  padding: '10px 12px',
  marginBottom: 2,
  borderRadius: 8,
  cursor: 'grab',
  fontSize: 13,
  color: '#c5c6d4',
  transition: 'all 0.12s ease',
  userSelect: 'none',
  background: 'transparent',
});

const iconBox = (color: string): React.CSSProperties => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 30,
  height: 30,
  borderRadius: 8,
  background: `${color}18`,
  color: color,
  flexShrink: 0,
});

const hintStyle: React.CSSProperties = {
  fontSize: 10,
  color: '#4b4d64',
  padding: '12px 16px',
  borderTop: '1px solid #1e1e30',
  lineHeight: 1.6,
};

// Inline hover style (simulated via CSS variable approach)
const hoverInjection = `
.sidebar-item:hover {
  background: #ffffff08 !important;
}
`;

// ============================================================
// Sidebar 组件
// ============================================================

export function Sidebar() {
  const onDragStart = (event: DragEvent<HTMLDivElement>, nodeType: string) => {
    event.dataTransfer.setData('application/reactflow-type', nodeType);
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <div style={sidebarStyle}>
      <style>{hoverInjection}</style>
      <div style={titleBlock}>
        <span style={{ fontSize: 16 }}>🧩</span>
        <span style={titleText}>节点面板</span>
      </div>

      <div style={scroller}>
        {['基础', '处理节点', '数据 & 路由'].map((group) => {
          const items = group === '基础'
            ? nodeTypes.filter((n) => n.type === 'start')
            : group === '处理节点'
              ? nodeTypes.filter((n) => n.type === 'agent' || n.type === 'tool')
              : nodeTypes.filter((n) => n.type === 'variable' || n.type === 'classifier');

          return (
            <React.Fragment key={group}>
              <div style={groupLabel}>{group}</div>
              {items.map((n) => {
                const IconComp = n.Icon;
                return (
                  <div
                    key={n.type}
                    className="sidebar-item"
                    style={item(n.color)}
                    draggable
                    onDragStart={(e) => onDragStart(e, n.type)}
                    title={n.desc}
                  >
                    <div style={iconBox(n.color)}>
                      <IconComp size={16} strokeWidth={1.8} />
                    </div>
                    <span>{n.label}</span>
                  </div>
                );
              })}
            </React.Fragment>
          );
        })}
      </div>

      <div style={hintStyle}>
        拖拽节点到画布上，连接创建流程。
      </div>
    </div>
  );
}
