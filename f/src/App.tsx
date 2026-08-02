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
import { Share2, Download, Upload } from 'lucide-react';

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

  // ---- 节点变化 ----
  const onNodesChange: OnNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChangeBase(changes);
      const selectionChange = changes.find((c) => c.type === 'select');
      if (selectionChange) {
        const nodeId = selectionChange.id;
        if (selectionChange.selected) {
          const node = nodes.find((n) => n.id === nodeId);
          setSelectedNode(node || null);
        } else {
          setSelectedNode(null);
        }
      }
    },
    [nodes, onNodesChangeBase]
  );

  // ---- 边变化 ----
  const onEdgesChange: OnEdgesChange = useCallback(
    (changes: EdgeChange[]) => onEdgesChangeBase(changes),
    [onEdgesChangeBase]
  );

  // ---- 连线 ----
  const onConnect: OnConnect = useCallback(
    (connection: Connection) => {
      setEdges((eds) => addEdge({ ...connection }, eds));
    },
    [setEdges]
  );

  // ---- 点击节点 ----
  const onNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      setSelectedNode(node);
    },
    []
  );

  // ---- 点击画布空白区域 ----
  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  // ---- 拖放新节点到画布 ----
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
      const position = {
        x: event.clientX - bounds.left - 100,
        y: event.clientY - bounds.top - 25,
      };

      const id = `${nodeType}_${Date.now()}`;
      let newNode: Node;

      switch (nodeType) {
        case 'start':
          newNode = { id, type: 'start', position, data: { nodeType: 'start', label: '开始' } };
          break;
        case 'agent':
          newNode = {
            id, type: 'agent', position,
            data: { nodeType: 'agent', label: '新 Agent', agentId: 0, agentName: '', inputTemplate: '', outputVar: '', condition: '', parallelGroup: '' },
          };
          break;
        case 'tool':
          newNode = {
            id, type: 'tool', position,
            data: { nodeType: 'tool', label: '新工具', toolName: '', toolParams: '', outputVar: '', condition: '', parallelGroup: '' },
          };
          break;
        case 'variable':
          newNode = {
            id, type: 'variable', position,
            data: { nodeType: 'variable', label: '变量', varName: '', varDesc: '' },
          };
          break;
        case 'classifier':
          newNode = {
            id, type: 'classifier', position,
            data: { nodeType: 'classifier', label: '意图分类', outputVar: 'intent', prompt: '', categories: [] },
          };
          break;
        default:
          return;
      }

      setNodes((nds) => [...nds, newNode]);
    },
    [setNodes]
  );

  // ---- 更新节点属性 ----
  const onUpdateNode = useCallback(
    (id: string, partialData: Partial<AppNodeData>) => {
      setNodes((nds) =>
        nds.map((n) => {
          if (n.id !== id) return n;
          return { ...n, data: { ...n.data, ...partialData } };
        })
      );
      setSelectedNode((prev) => {
        if (prev?.id === id) {
          return { ...prev, data: { ...prev.data, ...partialData } };
        }
        return prev;
      });
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
    const filename = workflowName || 'workflow';
    downloadJSON(def, filename);
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
        } catch (err) {
          alert('导入失败：JSON 格式不正确');
        }
      };
      reader.readAsText(file);
    };
    input.click();
  }, [setNodes, setEdges]);

  // ---- 键盘快捷键 ----
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
    <div style={{ width: '100vw', height: '100vh', display: 'flex', flexDirection: 'column', background: '#0d0d17' }}>
      <Toolbar
        name={workflowName}
        description={workflowDesc}
        nodeCount={nodes.length}
        edgeCount={edges.length}
        onNameChange={setWorkflowName}
        onDescChange={setWorkflowDesc}
        onExport={onExport}
        onImport={onImport}
      />

      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        <Sidebar />

        <div
          ref={reactFlowWrapper}
          style={{ flex: 1, background: '#12121f' }}
          onDragOver={onDragOver}
          onDrop={onDrop}
          onKeyDown={onKeyDown}
          tabIndex={0}
        >
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            nodeTypes={nodeTypes}
            fitView
            deleteKeyCode={['Delete', 'Backspace']}
            style={{ background: '#12121f' }}
          >
            <Controls
              style={{ background: '#1c1c2a', borderRadius: 10, border: '1px solid #1e1e30', boxShadow: '0 4px 16px rgba(0,0,0,0.4)' }}
            />
            <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="#1e1e30" />
            <MiniMap
              style={{ background: '#1c1c2a', borderRadius: 10, border: '1px solid #1e1e30', boxShadow: '0 4px 16px rgba(0,0,0,0.4)' }}
              maskColor="#0d0d1788"
              nodeColor={(n) => {
                const d = n.data as unknown as AppNodeData | undefined;
                switch (d?.nodeType) {
                  case 'start': return '#22c55e';
                  case 'agent': return '#3b82f6';
                  case 'tool': return '#14b8a6';
                  case 'variable': return '#f59e0b';
                  case 'classifier': return '#8b5cf6';
                  default: return '#5b5d78';
                }
              }}
            />
          </ReactFlow>
        </div>

        <PropertiesPanel
          node={selectedNode}
          onUpdate={onUpdateNode}
          onDelete={onDeleteNode}
        />
      </div>
    </div>
  );
}

// ============================================================
// 顶部工具栏
// ============================================================

function Toolbar({
  name, description, nodeCount, edgeCount,
  onNameChange, onDescChange, onExport, onImport,
}: {
  name: string;
  description: string;
  nodeCount: number;
  edgeCount: number;
  onNameChange: (v: string) => void;
  onDescChange: (v: string) => void;
  onExport: () => void;
  onImport: () => void;
}) {
  const bar: React.CSSProperties = {
    display: 'flex', alignItems: 'center', gap: 10,
    padding: '8px 20px', background: '#14141f',
    borderBottom: '1px solid #1e1e30',
  };
  const inputBase: React.CSSProperties = {
    padding: '7px 12px', borderRadius: 8, border: '1px solid #2a2a3d',
    background: '#1c1c2a', color: '#e2e4ed', fontSize: 13,
    fontFamily: 'inherit', transition: 'border-color 0.15s',
  };
  const btnBase: React.CSSProperties = {
    display: 'inline-flex', alignItems: 'center', gap: 6,
    padding: '7px 14px', borderRadius: 8, border: '1px solid #2a2a3d',
    background: '#1c1c2a', color: '#c5c6d4', cursor: 'pointer',
    fontSize: 12, fontWeight: 500, fontFamily: 'inherit',
    transition: 'all 0.12s',
  };
  const primaryBtn: React.CSSProperties = {
    ...btnBase, background: '#3b82f6', color: '#fff', borderColor: '#3b82f6',
  };
  const stats: React.CSSProperties = {
    fontSize: 11, color: '#5b5d78', marginLeft: 'auto', fontWeight: 500,
  };

  return (
    <div style={bar}>
      <Share2 size={18} style={{ color: '#5b5d78', marginRight: 2 }} />
      <input
        style={{ ...inputBase, fontWeight: 600, width: 180 }}
        value={name} placeholder="工作流名称"
        onChange={(e) => onNameChange(e.target.value)}
      />
      <input
        style={{ ...inputBase, fontWeight: 400, width: 200, fontSize: 12 }}
        value={description} placeholder="描述（可选）"
        onChange={(e) => onDescChange(e.target.value)}
      />
      <button style={btnBase} onClick={onImport}>
        <Upload size={14} /> 导入
      </button>
      <button style={primaryBtn} onClick={onExport}>
        <Download size={14} /> 导出 JSON
      </button>
      <span style={stats}>{nodeCount} 节点 · {edgeCount} 连线</span>
    </div>
  );
}
