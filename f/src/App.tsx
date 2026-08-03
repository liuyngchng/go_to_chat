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
  useReactFlow,
  MarkerType,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { Sidebar } from './Sidebar';
import { PropertiesPanel } from './PropertiesPanel';
import { nodeTypes } from './nodes';
import {
  exportToDesignDoc,
  importFromDesignDoc,
  exportToMarkdown,
  exportToAIPrompt,
  downloadJSON,
  downloadText,
} from './utils';
import { useUndoRedo } from './hooks';
import { computeLayout } from './layout';
import type { AppNodeData, DesignDoc } from './types';

// ============================================================
// 初始画布
// ============================================================

const initialNodes: Node[] = [
  {
    id: '__start__',
    type: 'start',
    position: { x: 80, y: 280 },
    data: { nodeType: 'start', label: '开始', purpose: '' },
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
  const [selectedEdge, setSelectedEdge] = useState<Edge | null>(null);

  const [workflowName, setWorkflowName] = useState('');
  const [workflowDesc, setWorkflowDesc] = useState('');
  const [workflowPurpose, setWorkflowPurpose] = useState('');
  const [workflowTags, setWorkflowTags] = useState('');

  const { pushHistory, undo, redo } = useUndoRedo();

  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const reactFlowInstance = useReactFlow();
  // 复制缓冲（含节点和边）
  const copyBufferRef = useRef<{ nodes: Node[]; edges: Edge[] } | null>(null);

  // ---- 节点/边变更 ----
  const onNodesChange: OnNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChangeBase(changes);
      const sel = changes.find((c) => c.type === 'select');
      if (sel) {
        if (sel.selected) {
          const node = nodes.find((n) => n.id === sel.id);
          setSelectedNode(node || null);
          setSelectedEdge(null);
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
    (connection: Connection) => {
      setEdges((eds) => addEdge({ ...connection }, eds));
    },
    [setEdges]
  );

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => setSelectedNode(node), []);

  const onEdgeClick = useCallback((_: React.MouseEvent, edge: Edge) => {
    setSelectedEdge(edge);
    setSelectedNode(null);
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
    setSelectedEdge(null);
  }, []);

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

      // 用 screenToFlowPosition 把鼠标屏幕坐标转换为画布坐标，
      // 这样即使画布缩放/平移过，新节点也能落在鼠标所在位置
      const position = reactFlowInstance.screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });
      const id = `${nodeType}_${Date.now()}`;
      let newNode: Node;

      switch (nodeType) {
        case 'start':
          newNode = { id, type: 'start', position, data: { nodeType: 'start', label: '开始', purpose: '' } };
          break;
        case 'agent':
          newNode = { id, type: 'agent', position, data: { nodeType: 'agent', label: '新 Agent', purpose: '', agentName: '', inputTemplate: '', outputVar: '', parallelGroup: '' } };
          break;
        case 'tool':
          newNode = { id, type: 'tool', position, data: { nodeType: 'tool', label: '新工具', purpose: '', toolName: '', toolParams: '', outputVar: '', parallelGroup: '' } };
          break;
        case 'branch':
          newNode = { id, type: 'branch', position, data: { nodeType: 'branch', label: '条件分支', purpose: '', inputVar: '' } };
          break;
        case 'note':
          newNode = { id, type: 'note', position, data: { nodeType: 'note', label: '便签', purpose: '', content: '', color: 'yellow' } };
          break;
        default:
          return;
      }
      setNodes((nds) => [...nds, newNode]);
      setSelectedNode(newNode);
    },
    [setNodes, reactFlowInstance]
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

  // ---- 更新边条件 ----
  const onUpdateEdge = useCallback(
    (edgeId: string, updates: Partial<Edge>) => {
      setEdges((eds) => eds.map((e) => e.id !== edgeId ? e : { ...e, ...updates }));
      setSelectedEdge((prev) => prev?.id === edgeId ? { ...prev, ...updates } : prev);
    },
    [setEdges]
  );

  // ---- 撤销 / 重做 ----
  const handleUndo = useCallback(() => {
    undo(nodes, edges, setNodes, setEdges);
    setSelectedNode(null);
  }, [undo, nodes, edges, setNodes, setEdges]);

  const handleRedo = useCallback(() => {
    redo(nodes, edges, setNodes, setEdges);
    setSelectedNode(null);
  }, [redo, nodes, edges, setNodes, setEdges]);

  // ---- 导出（JSON） ----
  const buildDoc = useCallback((): DesignDoc => {
    const tags = workflowTags.split(/[,，\s]+/).filter(Boolean);
    return exportToDesignDoc(workflowName, workflowDesc, workflowPurpose, tags, nodes, edges);
  }, [workflowName, workflowDesc, workflowPurpose, workflowTags, nodes, edges]);

  const onExport = useCallback(() => {
    const doc = buildDoc();
    downloadJSON(doc, workflowName || 'workflow');
  }, [buildDoc, workflowName]);

  const onExportMarkdown = useCallback(() => {
    const doc = buildDoc();
    downloadText(exportToMarkdown(doc), `${workflowName || 'workflow'}.md`);
  }, [buildDoc, workflowName]);

  const onExportAI = useCallback(() => {
    const doc = buildDoc();
    downloadText(exportToAIPrompt(doc), `${workflowName || 'workflow'}-prompt.txt`);
  }, [buildDoc, workflowName]);

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
          const doc: DesignDoc = JSON.parse(re.target?.result as string);
          setWorkflowName(doc.name || '');
          setWorkflowDesc(doc.description || '');
          setWorkflowPurpose(doc.purpose || '');
          setWorkflowTags((doc.metadata?.tags || []).join(', '));
          const { nodes: newNodes, edges: newEdges } = importFromDesignDoc(doc);
          setNodes(newNodes);
          setEdges(newEdges);
          setSelectedNode(null);
          // 导入后清空历史
          pushHistory(newNodes, newEdges);
        } catch { alert('导入失败：JSON 格式不正确'); }
      };
      reader.readAsText(file);
    };
    input.click();
  }, [setNodes, setEdges, pushHistory]);

  // ---- 键盘 ----
  const onKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      // Ctrl/Cmd + Z 撤销
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'z' && !event.shiftKey) {
        event.preventDefault();
        handleUndo();
        return;
      }
      // Ctrl/Cmd + Shift + Z 重做
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'z' && event.shiftKey) {
        event.preventDefault();
        handleRedo();
        return;
      }
      // Ctrl/Cmd + Y 重做
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'y') {
        event.preventDefault();
        handleRedo();
        return;
      }
      // Ctrl/Cmd + S 导出
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        onExport();
        return;
      }
      // Delete / Backspace 删除节点
      if ((event.key === 'Delete' || event.key === 'Backspace') && selectedNode) {
        const data = selectedNode.data as unknown as AppNodeData;
        if (data.nodeType === 'start') return;
        onDeleteNode(selectedNode.id);
      }
    },
    [selectedNode, onDeleteNode, handleUndo, handleRedo, onExport]
  );

  // 拖拽结束时记录历史
  const onNodeDragStop = useCallback(() => {
    pushHistory(nodes, edges);
  }, [nodes, edges, pushHistory]);

  // ---- 自动布局 ----
  const onAutoLayout = useCallback(() => {
    const positions = computeLayout(nodes, edges);
    pushHistory(nodes, edges);
    setNodes((nds) => nds.map((n) => {
      const pos = positions.get(n.id);
      return pos ? { ...n, position: { ...pos } } : n;
    }));
    // 布局后适配视野
    setTimeout(() => {
      reactFlowInstance.fitView({ padding: 0.15 });
    }, 50);
  }, [nodes, edges, setNodes, pushHistory, reactFlowInstance]);

  return (
    <div style={{ width: '100vw', height: '100vh', display: 'flex', flexDirection: 'column', background: '#f5f7fa' }}>
      <Toolbar
        name={workflowName} description={workflowDesc} purpose={workflowPurpose} tags={workflowTags}
        nodeCount={nodes.length} edgeCount={edges.length}
        onNameChange={setWorkflowName} onDescChange={setWorkflowDesc}
        onPurposeChange={setWorkflowPurpose} onTagsChange={setWorkflowTags}
        onExport={onExport} onExportMarkdown={onExportMarkdown} onExportAI={onExportAI}
        onImport={onImport} onUndo={handleUndo} onRedo={handleRedo}
        onAutoLayout={onAutoLayout}
      />

      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        <Sidebar />

        <div ref={reactFlowWrapper} style={{ flex: 1, background: '#f8f9fb' }}
          onDragOver={onDragOver} onDrop={onDrop} onKeyDown={onKeyDown} tabIndex={0}>
          <ReactFlow
            nodes={nodes} edges={edges}
            onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
            onConnect={onConnect} onNodeClick={onNodeClick} onPaneClick={onPaneClick}
            onEdgeClick={onEdgeClick}
            onNodeDragStop={onNodeDragStop}
            nodeTypes={nodeTypes} fitView deleteKeyCode={['Delete', 'Backspace']}
            style={{ background: '#f8f9fb' }}
            defaultEdgeOptions={{
              type: 'default',
              markerEnd: { type: MarkerType.ArrowClosed, width: 18, height: 18, color: '#b0b8c8' },
            }}
          >
            <Controls style={{ background: '#fff', borderRadius: 10, border: '1px solid #e0e3e8', boxShadow: '0 4px 16px rgba(0,0,0,0.08)' }} />
            <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="#d0d5dd" />
            <MiniMap
              style={{ background: '#fff', borderRadius: 10, border: '1px solid #e0e3e8', boxShadow: '0 4px 16px rgba(0,0,0,0.08)' }}
              maskColor="#f5f7fa88"
              nodeColor={(n) => (n.data as unknown as AppNodeData).nodeType === 'note' ? '#f9a825' : '#4b6cb7'}
            />
          </ReactFlow>
        </div>

        <PropertiesPanel
          node={selectedNode}
          edge={selectedEdge}
          nodes={nodes}
          edges={edges}
          onUpdateNode={onUpdateNode}
          onUpdateEdge={onUpdateEdge}
          onDeleteNode={onDeleteNode}
        />
      </div>
    </div>
  );
}

// ============================================================
// 顶部工具栏
// ============================================================

function Toolbar({
  name, description, purpose, tags, nodeCount, edgeCount,
  onNameChange, onDescChange, onPurposeChange, onTagsChange,
  onExport, onExportMarkdown, onExportAI, onImport, onUndo, onRedo, onAutoLayout,
}: {
  name: string; description: string; purpose: string; tags: string;
  nodeCount: number; edgeCount: number;
  onNameChange: (v: string) => void; onDescChange: (v: string) => void;
  onPurposeChange: (v: string) => void; onTagsChange: (v: string) => void;
  onExport: () => void; onExportMarkdown: () => void; onExportAI: () => void;
  onImport: () => void; onUndo: () => void; onRedo: () => void; onAutoLayout: () => void;
}) {
  const bar: React.CSSProperties = {
    display: 'flex', alignItems: 'center', gap: 10,
    padding: '10px 16px', flexWrap: 'wrap',
    background: 'linear-gradient(to right, #4b6cb7, #182848)',
    color: '#fff',
  };
  const inputBase: React.CSSProperties = {
    padding: '6px 12px', borderRadius: 6, border: '1px solid rgba(255,255,255,0.3)',
    background: 'rgba(255,255,255,0.12)', color: '#fff', fontSize: 12,
    fontFamily: 'inherit', transition: 'border-color 0.2s',
  };
  const btnBase: React.CSSProperties = {
    display: 'inline-flex', alignItems: 'center', gap: 6,
    padding: '6px 12px', borderRadius: 6, border: '1px solid rgba(255,255,255,0.3)',
    background: 'rgba(255,255,255,0.12)', color: '#fff',
    cursor: 'pointer', fontSize: 11, fontWeight: 600, fontFamily: 'inherit',
    transition: 'all 0.2s', whiteSpace: 'nowrap',
  };
  const primaryBtn: React.CSSProperties = {
    ...btnBase, background: 'rgba(255,255,255,0.22)', borderColor: 'rgba(255,255,255,0.4)',
  };
  const stats: React.CSSProperties = { fontSize: 11, color: 'rgba(255,255,255,0.6)', marginLeft: 'auto' };

  return (
    <div style={bar}>
      <i className="fas fa-project-diagram" style={{ fontSize: 18 }} />
      <input className="toolbar-input" style={{ ...inputBase, fontWeight: 600, width: 150 }}
        value={name} placeholder="工作流名称" onChange={(e) => onNameChange(e.target.value)} />
      <input className="toolbar-input" style={{ ...inputBase, fontWeight: 400, width: 180, fontSize: 11 }}
        value={description} placeholder="描述（可选）" onChange={(e) => onDescChange(e.target.value)} />
      <input className="toolbar-input" style={{ ...inputBase, fontWeight: 400, width: 220, fontSize: 11 }}
        value={purpose} placeholder="💡 整体设计意图（可选，写入 JSON）" onChange={(e) => onPurposeChange(e.target.value)} />
      <input className="toolbar-input" style={{ ...inputBase, fontWeight: 400, width: 120, fontSize: 11 }}
        value={tags} placeholder="标签（逗号分隔）" onChange={(e) => onTagsChange(e.target.value)} />

      <span style={{ display: 'flex', gap: 6, marginLeft: 'auto', flexWrap: 'wrap' }}>
        <button style={btnBase} onClick={onUndo} title="撤销 (Ctrl+Z)">
          <i className="fas fa-undo" />
        </button>
        <button style={btnBase} onClick={onRedo} title="重做 (Ctrl+Shift+Z / Ctrl+Y)">
          <i className="fas fa-redo" />
        </button>
        <button style={btnBase} onClick={onAutoLayout} title="自动布局 — 按拓扑层级整理节点">
          <i className="fas fa-arrows-alt" /> 自动布局
        </button>
        <button style={btnBase} onClick={onImport} title="导入 JSON">
          <i className="fas fa-upload" /> 导入
        </button>
        <button style={btnBase} onClick={onExport} title="导出自描述 JSON (Ctrl+S)">
          <i className="fas fa-download" /> 导出 JSON
        </button>
        <button style={btnBase} onClick={onExportMarkdown} title="导出 Markdown / Mermaid 文档">
          <i className="fas fa-file-alt" /> 导出 MD
        </button>
        <button style={primaryBtn} onClick={onExportAI} title="导出为 AI Prompt，直接喂给 AI 编程工具">
          <i className="fas fa-robot" /> 导出 AI Prompt
        </button>
      </span>
      <span style={stats}>{nodeCount} 节点 · {edgeCount} 连线</span>
    </div>
  );
}