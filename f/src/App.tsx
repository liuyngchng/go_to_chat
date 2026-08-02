import React, { useState, useCallback, useRef, type DragEvent } from 'react';
import {
  ReactFlow,
  type Node,
  type Edge,
  type Connection,
  type NodeChange,
  type EdgeChange,
  type OnNodesChange,
  type OnEdgesChange,
  type OnConnect,
  addEdge,
  Controls,
  Background,
  BackgroundVariant,
  MiniMap,
  useNodesState,
  useEdgesState,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { Sidebar } from './Sidebar';
import { PropertiesPanel } from './PropertiesPanel';
import { nodeTypes } from './nodes';
import { exportToWorkflowDef, importFromWorkflowDef, downloadJSON } from './utils';
import type { AppNodeData, WorkflowDef } from './types';

// ============================================================
// 初始画布
// ============================================================

const initialNodes: Node[] = [
  {
    id: '__start__',
    type: 'start',
    position: { x: 80, y: 280 },
    data: { nodeType: 'start', label: '开始' },
  },
];

const initialEdges: Edge[] = [];

// ============================================================
// App 主组件
// ============================================================

export default function App() {
  const [nodes, setNodes, onNodesChangeBase] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChangeBase] = useEdgesState(initialEdges);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);

  const [workflowName, setWorkflowName] = useState('');
  const [workflowDesc, setWorkflowDesc] = useState('');

  const reactFlowWrapper = useRef<HTMLDivElement>(null);

  const onNodesChange: OnNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChangeBase(changes);
      const sel = changes.find((c) => c.type === 'select');
      if (sel) {
        if (sel.selected) {
          const node = nodes.find((n) => n.id === sel.id);
          setSelectedNode(node || null);
        } else {
          setSelectedNode(null);
        }
      }
    },
    [nodes, onNodesChangeBase]
  );

  const onEdgesChange: OnEdgesChange = useCallback(
    (changes: EdgeChange[]) => onEdgesChangeBase(changes),
    [onEdgesChangeBase]
  );

  const onConnect: OnConnect = useCallback(
    (connection: Connection) => setEdges((eds) => addEdge({ ...connection }, eds)),
    [setEdges]
  );

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => setSelectedNode(node), []);
  const onPaneClick = useCallback(() => setSelectedNode(null), []);

  // ---- 拖放 ----
  const onDragOver = useCallback((event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      const nodeType = event.dataTransfer.getData('application/reactflow-type');
      if (!nodeType || !reactFlowWrapper.current) return;

      const bounds = reactFlowWrapper.current.getBoundingClientRect();
      const position = { x: event.clientX - bounds.left - 100, y: event.clientY - bounds.top - 25 };
      const id = `${nodeType}_${Date.now()}`;
      let newNode: Node;

      switch (nodeType) {
        case 'start':
          newNode = { id, type: 'start', position, data: { nodeType: 'start', label: '开始' } };
          break;
        case 'agent':
          newNode = { id, type: 'agent', position, data: { nodeType: 'agent', label: '新 Agent', agentId: 0, agentName: '', inputTemplate: '', outputVar: '', condition: '', parallelGroup: '' } };
          break;
        case 'tool':
          newNode = { id, type: 'tool', position, data: { nodeType: 'tool', label: '新工具', toolName: '', toolParams: '', outputVar: '', condition: '', parallelGroup: '' } };
          break;
        case 'variable':
          newNode = { id, type: 'variable', position, data: { nodeType: 'variable', label: '变量', varName: '', varDesc: '' } };
          break;
        case 'classifier':
          newNode = { id, type: 'classifier', position, data: { nodeType: 'classifier', label: '意图分类', outputVar: 'intent', prompt: '', categories: [] } };
          break;
        default:
          return;
      }
      setNodes((nds) => [...nds, newNode]);
    },
    [setNodes]
  );

  // ---- 更新节点 ----
  const onUpdateNode = useCallback(
    (id: string, partialData: Partial<AppNodeData>) => {
      setNodes((nds) => nds.map((n) => n.id !== id ? n : { ...n, data: { ...n.data, ...partialData } }));
      setSelectedNode((prev) => prev?.id === id ? { ...prev, data: { ...prev.data, ...partialData } } : prev);
    },
    [setNodes]
  );

  // ---- 删除节点 ----
  const onDeleteNode = useCallback(
    (id: string) => {
      setNodes((nds) => nds.filter((n) => n.id !== id));
      setEdges((eds) => eds.filter((e) => e.source !== id && e.target !== id));
      setSelectedNode(null);
    },
    [setNodes, setEdges]
  );

  // ---- 导出 ----
  const onExport = useCallback(() => {
    const def = exportToWorkflowDef(workflowName, workflowDesc, nodes, edges);
    downloadJSON(def, workflowName || 'workflow');
  }, [workflowName, workflowDesc, nodes, edges]);

  // ---- 导入 ----
  const onImport = useCallback(() => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.onchange = (e: Event) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = (re) => {
        try {
          const def: WorkflowDef = JSON.parse(re.target?.result as string);
          setWorkflowName(def.name || '');
          setWorkflowDesc(def.description || '');
          const { nodes: newNodes, edges: newEdges } = importFromWorkflowDef(def);
          setNodes(newNodes);
          setEdges(newEdges);
        } catch { alert('导入失败：JSON 格式不正确'); }
      };
      reader.readAsText(file);
    };
    input.click();
  }, [setNodes, setEdges]);

  // ---- 键盘 ----
  const onKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if ((event.key === 'Delete' || event.key === 'Backspace') && selectedNode) {
        const data = selectedNode.data as unknown as AppNodeData;
        if (data.nodeType === 'start') return;
        onDeleteNode(selectedNode.id);
      }
    },
    [selectedNode, onDeleteNode]
  );

  return (
    <div style={{ width: '100vw', height: '100vh', display: 'flex', flexDirection: 'column', background: '#f5f7fa' }}>
      <Toolbar
        name={workflowName} description={workflowDesc}
        nodeCount={nodes.length} edgeCount={edges.length}
        onNameChange={setWorkflowName} onDescChange={setWorkflowDesc}
        onExport={onExport} onImport={onImport}
      />

      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        <Sidebar />

        <div ref={reactFlowWrapper} style={{ flex: 1, background: '#f8f9fb' }}
          onDragOver={onDragOver} onDrop={onDrop} onKeyDown={onKeyDown} tabIndex={0}>
          <ReactFlow
            nodes={nodes} edges={edges}
            onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
            onConnect={onConnect} onNodeClick={onNodeClick} onPaneClick={onPaneClick}
            nodeTypes={nodeTypes} fitView deleteKeyCode={['Delete', 'Backspace']}
            style={{ background: '#f8f9fb' }}
          >
            <Controls style={{ background: '#fff', borderRadius: 10, border: '1px solid #e0e3e8', boxShadow: '0 4px 16px rgba(0,0,0,0.08)' }} />
            <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="#d0d5dd" />
            <MiniMap
              style={{ background: '#fff', borderRadius: 10, border: '1px solid #e0e3e8', boxShadow: '0 4px 16px rgba(0,0,0,0.08)' }}
              maskColor="#f5f7fa88"
              nodeColor={() => '#4b6cb7'}
            />
          </ReactFlow>
        </div>

        <PropertiesPanel node={selectedNode} onUpdate={onUpdateNode} onDelete={onDeleteNode} />
      </div>
    </div>
  );
}

// ============================================================
// 顶部工具栏（匹配 g/ header 风格）
// ============================================================

function Toolbar({
  name, description, nodeCount, edgeCount,
  onNameChange, onDescChange, onExport, onImport,
}: {
  name: string; description: string; nodeCount: number; edgeCount: number;
  onNameChange: (v: string) => void; onDescChange: (v: string) => void;
  onExport: () => void; onImport: () => void;
}) {
  const bar: React.CSSProperties = {
    display: 'flex', alignItems: 'center', gap: 10,
    padding: '12px 20px',
    background: 'linear-gradient(to right, #4b6cb7, #182848)',
    color: '#fff',
  };
  const inputBase: React.CSSProperties = {
    padding: '7px 12px', borderRadius: 6, border: '1px solid rgba(255,255,255,0.3)',
    background: 'rgba(255,255,255,0.12)', color: '#fff', fontSize: 13,
    fontFamily: 'inherit', transition: 'border-color 0.2s',
  };
  const btnBase: React.CSSProperties = {
    display: 'inline-flex', alignItems: 'center', gap: 6,
    padding: '7px 14px', borderRadius: 6, border: '1px solid rgba(255,255,255,0.3)',
    background: 'rgba(255,255,255,0.12)', color: '#fff',
    cursor: 'pointer', fontSize: 12, fontWeight: 600, fontFamily: 'inherit',
    transition: 'all 0.2s',
  };
  const primaryBtn: React.CSSProperties = {
    ...btnBase, background: 'rgba(255,255,255,0.22)', borderColor: 'rgba(255,255,255,0.4)',
  };
  const stats: React.CSSProperties = { fontSize: 11, color: 'rgba(255,255,255,0.6)', marginLeft: 'auto' };

  return (
    <div style={bar}>
      <i className="fas fa-project-diagram" style={{ fontSize: 18 }} />
      <input style={{ ...inputBase, fontWeight: 600, width: 180 }}
        value={name} placeholder="工作流名称" onChange={(e) => onNameChange(e.target.value)} />
      <input style={{ ...inputBase, fontWeight: 400, width: 200, fontSize: 12 }}
        value={description} placeholder="描述（可选）" onChange={(e) => onDescChange(e.target.value)} />
      <button style={btnBase} onClick={onImport}>
        <i className="fas fa-upload" /> 导入
      </button>
      <button style={primaryBtn} onClick={onExport}>
        <i className="fas fa-download" /> 导出 JSON
      </button>
      <span style={stats}>{nodeCount} 节点 · {edgeCount} 连线</span>
    </div>
  );
}
