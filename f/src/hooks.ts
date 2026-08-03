import { useCallback, useRef } from 'react';
import type { Node, Edge } from '@xyflow/react';

// ============================================================
// Undo/Redo 历史栈
// 快照 nodes + edges，最多保留 50 步
// ============================================================

interface Snapshot {
  nodes: Node[];
  edges: Edge[];
}

const MAX_HISTORY = 50;

export function useUndoRedo() {
  // 用 ref 而非 state，避免触发重渲染
  const pastRef = useRef<Snapshot[]>([]);
  const futureRef = useRef<Snapshot[]>([]);

  const pushHistory = useCallback((nodes: Node[], edges: Edge[]) => {
    // 简单浅拷贝 + id 引用保持（React Flow 内部需要对象引用不变）
    const snap: Snapshot = {
      nodes: nodes.map((n) => ({ ...n, data: { ...n.data } })),
      edges: edges.map((e) => ({ ...e })),
    };
    pastRef.current.push(snap);
    if (pastRef.current.length > MAX_HISTORY) {
      pastRef.current.shift();
    }
    // 新操作清除 redo 栈
    futureRef.current = [];
  }, []);

  const undo = useCallback((
    currentNodes: Node[],
    currentEdges: Edge[],
    setNodes: (updater: Node[] | ((prev: Node[]) => Node[])) => void,
    setEdges: (updater: Edge[] | ((prev: Edge[]) => Edge[])) => void,
  ): boolean => {
    const prev = pastRef.current.pop();
    if (!prev) return false;

    // 先把当前状态推进 future
    futureRef.current.push({
      nodes: currentNodes.map((n) => ({ ...n, data: { ...n.data } })),
      edges: currentEdges.map((e) => ({ ...e })),
    });

    setNodes(prev.nodes);
    setEdges(prev.edges);
    return true;
  }, []);

  const redo = useCallback((
    currentNodes: Node[],
    currentEdges: Edge[],
    setNodes: (updater: Node[] | ((prev: Node[]) => Node[])) => void,
    setEdges: (updater: Edge[] | ((prev: Edge[]) => Edge[])) => void,
  ): boolean => {
    const next = futureRef.current.pop();
    if (!next) return false;

    // 先把当前状态推进 past
    pastRef.current.push({
      nodes: currentNodes.map((n) => ({ ...n, data: { ...n.data } })),
      edges: currentEdges.map((e) => ({ ...e })),
    });

    setNodes(next.nodes);
    setEdges(next.edges);
    return true;
  }, []);

  const clear = useCallback(() => {
    pastRef.current = [];
    futureRef.current = [];
  }, []);

  const canUndo = pastRef.current.length > 0;
  const canRedo = futureRef.current.length > 0;

  return { pushHistory, undo, redo, clear, canUndo, canRedo };
}