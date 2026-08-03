// ============================================================
// 自动布局工具
// 用拓扑排序分层，同层节点竖排，层间横向排列
// ============================================================

import type { Node, Edge } from '@xyflow/react';
import type { AppNodeData } from './types';

// 布局参数
const X_GAP = 260;   // 层与层之间的水平间距
const Y_GAP = 40;    // 同层节点之间的垂直间距
const START_X = 60;  // 起始 X
const START_Y = 60;  // 起始 Y
const NODE_WIDTH = 200;
const NODE_HEIGHT = 70;

/** 计算每个节点的层级（Kahn 拓扑分层） */
function computeLevels(nodes: Node[], edges: Edge[]): Map<string, number> {
  const nodeIds = new Set(nodes.map((n) => n.id));
  const inDegree = new Map<string, number>();
  const adj = new Map<string, string[]>();

  // 初始化
  for (const n of nodes) {
    inDegree.set(n.id, 0);
    adj.set(n.id, []);
  }

  // 构建邻接表 + 入度
  for (const e of edges) {
    if (!nodeIds.has(e.source) || !nodeIds.has(e.target)) continue;
    adj.get(e.source)!.push(e.target);
    inDegree.set(e.target, (inDegree.get(e.target) || 0) + 1);
  }

  // Kahn 分层
  const levels = new Map<string, number>();
  let queue: string[] = [];
  for (const [id, deg] of inDegree) {
    if (deg === 0) queue.push(id);
  }

  let level = 0;
  while (queue.length > 0) {
    const nextQueue: string[] = [];
    for (const id of queue) {
      levels.set(id, level);
      for (const target of adj.get(id) || []) {
        const newDeg = (inDegree.get(target) || 0) - 1;
        inDegree.set(target, newDeg);
        if (newDeg === 0) nextQueue.push(target);
      }
    }
    queue = nextQueue;
    level++;
  }

  // 有环或孤立的节点，放到最后
  for (const n of nodes) {
    if (!levels.has(n.id)) {
      levels.set(n.id, level);
    }
  }

  return levels;
}

/** 计算节点位置 */
export function computeLayout(nodes: Node[], edges: Edge[]): Map<string, { x: number; y: number }> {
  const levels = computeLevels(nodes, edges);

  // 按层级分组
  const levelGroups = new Map<number, string[]>();
  for (const [id, lv] of levels) {
    const group = levelGroups.get(lv) || [];
    group.push(id);
    levelGroups.set(lv, group);
  }

  const positions = new Map<string, { x: number; y: number }>();

  // 计算每层的宽度（用于水平居中）
  const maxLevel = Math.max(...Array.from(levelGroups.keys()));

  for (const [lv, ids] of levelGroups) {
    // 同层垂直排列，居中到画布中心
    const totalHeight = ids.length * NODE_HEIGHT + (ids.length - 1) * Y_GAP;
    let y = START_Y;

    // 尝试居中：让整层在画布中垂直居中
    const levelStartY = START_Y + Math.max(0, (500 - totalHeight) / 2);

    for (const id of ids) {
      positions.set(id, {
        x: START_X + lv * X_GAP,
        y: levelStartY + ids.indexOf(id) * (NODE_HEIGHT + Y_GAP),
      });
    }
  }

  return positions;
}

/** 根据节点类型返回参考尺寸 */
export function getNodeSize(node: Node): { width: number; height: number } {
  const data = node.data as unknown as AppNodeData;
  // 便签可以更大
  if (data.nodeType === 'note') {
    return { width: 240, height: 100 };
  }
  return { width: NODE_WIDTH, height: NODE_HEIGHT };
}