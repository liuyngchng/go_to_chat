import React, { useState } from 'react';
import type { Node, Edge } from '@xyflow/react';
import type { AppNodeData, AgentNodeData, ToolNodeData, VariableNodeData, BranchNodeData, NoteNodeData } from './types';
import { findDuplicateOutputVars, resolveTemplateVars, getUpstreamVars } from './validation';

// ============================================================
// 样式（匹配 g/ 后台面板风格）
// ============================================================

const panel: React.CSSProperties = {
  width: 300, background: '#fff', borderLeft: '1px solid #e0e3e8',
  display: 'flex', flexDirection: 'column', overflow: 'hidden',
};

const hdr: React.CSSProperties = {
  fontSize: 14, fontWeight: 700, color: '#333',
  padding: '16px 18px', borderBottom: '1px solid #e8eaed',
};

const emptyMsg: React.CSSProperties = {
  padding: 32, color: '#999', fontSize: 13, textAlign: 'center', lineHeight: 1.7,
};

const sect: React.CSSProperties = { padding: '14px 18px', borderBottom: '1px solid #f0f0f0' };

const lab: React.CSSProperties = {
  fontSize: 11, fontWeight: 600, color: '#666', marginBottom: 6,
  textTransform: 'uppercase', letterSpacing: '0.05em',
};

const inp: React.CSSProperties = {
  width: '100%', boxSizing: 'border-box', padding: '8px 11px',
  borderRadius: 6, border: '2px solid #e0e3e8', background: '#fff',
  color: '#333', fontSize: 12, fontFamily: 'inherit',
  transition: 'border-color 0.2s',
};

const txt: React.CSSProperties = {
  ...inp, resize: 'vertical', minHeight: 64,
  fontFamily: 'Consolas, Monaco, monospace',
};

const sel: React.CSSProperties = { ...inp, appearance: 'none', cursor: 'pointer', backgroundImage: 'none' };

const chipStyle: React.CSSProperties = {
  display: 'inline-flex', alignItems: 'center', gap: 4,
  padding: '3px 9px', borderRadius: 4, background: '#e3f2fd',
  color: '#1976d2', fontSize: 10, fontFamily: 'Consolas, Monaco, monospace',
  marginRight: 4, marginBottom: 4, cursor: 'pointer',
};

const btnBase: React.CSSProperties = {
  padding: '7px 14px', borderRadius: 6, border: 'none',
  background: '#f0f0f0', color: '#555', cursor: 'pointer', fontSize: 11,
  fontWeight: 500, fontFamily: 'inherit', transition: 'all 0.2s',
  marginRight: 6, marginTop: 4,
};

const dangerBtn: React.CSSProperties = {
  ...btnBase, background: '#ffebee', color: '#c62828',
};

// ============================================================
// 主组件
// ============================================================

interface Props {
  node: Node | null;
  edge: Edge | null;
  nodes: Node[];
  edges: Edge[];
  onUpdateNode: (id: string, data: Partial<AppNodeData>) => void;
  onUpdateEdge: (edgeId: string, updates: Partial<Edge>) => void;
  onDeleteNode: (id: string) => void;
}

export function PropertiesPanel({ node, edge, nodes, edges, onUpdateNode, onUpdateEdge, onDeleteNode }: Props) {
  // 选中边时显示边面板
  if (edge && !node) {
    return (
      <div style={panel}>
        <div style={hdr}>
          连线属性
          <div style={{ fontSize: 10, color: '#999', fontWeight: 400, marginTop: 3 }}>
            {edge.source} → {edge.target}
          </div>
        </div>
        <div style={{ flex: 1, overflowY: 'auto' }}>
          <EdgePanel edge={edge} onUpdateEdge={onUpdateEdge} />
        </div>
      </div>
    );
  }

  if (!node) {
    return (
      <div style={panel}>
        <div style={hdr}>属性面板</div>
        <div style={emptyMsg}>点击画布上的节点或连线<br />查看和编辑属性</div>
      </div>
    );
  }

  const data = node.data as unknown as AppNodeData;

  return (
    <div style={panel}>
      <div style={hdr}>
        节点属性
        <div style={{ fontSize: 10, color: '#999', fontWeight: 400, marginTop: 3 }}>
          {node.id} <span style={{ margin: '0 4px', color: '#ccc' }}>·</span> {data.nodeType}
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto' }}>
        {data.nodeType === 'agent' && <AgentPanel node={node} data={data} onUpdate={onUpdateNode} nodes={nodes} edges={edges} />}
        {data.nodeType === 'tool' && <ToolPanel node={node} data={data} onUpdate={onUpdateNode} nodes={nodes} edges={edges} />}
        {data.nodeType === 'variable' && <VarPanel node={node} data={data} onUpdate={onUpdateNode} />}
        {data.nodeType === 'branch' && <BranchPanel node={node} data={data} onUpdate={onUpdateNode} />}
        {data.nodeType === 'note' && <NotePanel node={node} data={data} onUpdate={onUpdateNode} />}
        {data.nodeType === 'start' && (
          <>
            <div style={sect}>
              <div style={{ color: '#888', fontSize: 12, lineHeight: 1.6 }}>
                入口节点 — 用户问题从这里输入，流向后续处理节点。
              </div>
            </div>
            <PurposeField node={node} data={data} onUpdate={onUpdateNode} />
          </>
        )}
      </div>

      {data.nodeType !== 'start' && (
        <div style={{ padding: '12px 18px', borderTop: '1px solid #f0f0f0' }}>
          <button style={dangerBtn} onClick={() => { if (confirm('确定删除节点？')) onDeleteNode(node.id); }}>
            <i className="fas fa-trash" style={{ marginRight: 4 }} /> 删除节点
          </button>
        </div>
      )}
    </div>
  );
}

// ============================================================
// Agent 属性
// ============================================================

function AgentPanel({ node, data, onUpdate, nodes, edges }: {
  node: Node; data: AgentNodeData;
  onUpdate: (id: string, data: Partial<AppNodeData>) => void;
  nodes: Node[]; edges: Edge[];
}) {
  const duplicates = findDuplicateOutputVars(nodes);
  const conflict = data.outputVar ? duplicates.get(data.outputVar) : undefined;
  const isConflict = conflict && conflict.length > 1 && conflict.includes(node.id);

  const templateVars = data.inputTemplate
    ? resolveTemplateVars(data.inputTemplate, node.id, nodes, edges)
    : [];
  const undefinedVars = templateVars.filter((v) => !v.defined);

  return (
    <>
      <PurposeField node={node} data={data} onUpdate={onUpdate} />
      <FormField label="名称">
        <input style={inp} value={data.agentName || ''} placeholder="如：客服助手"
          onChange={(e) => onUpdate(node.id, { agentName: e.target.value, label: e.target.value } as any)} />
      </FormField>
      <FormField label="输入模板">
        <textarea style={txt} value={data.inputTemplate || ''}
          placeholder={'使用 {{变量名}} 引用上游输出\n例：用户问题：{{user_query}}'}
          onChange={(e) => onUpdate(node.id, { inputTemplate: e.target.value } as any)} />
        <VarSuggest onSelect={(v) => onUpdate(node.id, { inputTemplate: (data.inputTemplate || '') + `{{${v}}}` } as any)}
          nodeId={node.id} nodes={nodes} edges={edges} />
        {undefinedVars.length > 0 && (
          <div style={{ marginTop: 6 }}>
            {undefinedVars.map((v) => (
              <div key={v.name} style={{ fontSize: 11, color: '#e67e22', padding: '3px 0' }}>
                ⚠️ <code>{`{{${v.name}}}`}</code> 上游未定义
              </div>
            ))}
          </div>
        )}
        {templateVars.filter((v) => v.defined).length > 0 && (
          <div style={{ marginTop: 4 }}>
            {templateVars.filter((v) => v.defined).map((v) => (
              <div key={v.name} style={{ fontSize: 11, color: '#2e7d32', padding: '2px 0' }}>
                ✓ <code>{`{{${v.name}}}`}</code> 来自 {v.definedByLabel || v.definedBy}
              </div>
            ))}
          </div>
        )}
      </FormField>
      <FormField label="输出变量名">
        <input style={{ ...inp, borderColor: isConflict ? '#e53935' : '#e0e3e8' }} value={data.outputVar || ''} placeholder="如 my_output"
          onChange={(e) => onUpdate(node.id, { outputVar: e.target.value } as any)} />
        {isConflict && (
          <div style={{ fontSize: 11, color: '#e53935', marginTop: 4 }}>
            ✗ 变量名 "{data.outputVar}" 与 {conflict!.filter((id) => id !== node.id).join(', ')} 冲突
          </div>
        )}
        {!isConflict && data.outputVar && (
          <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>下游可通过 {`{{${data.outputVar}}}`} 引用</div>
        )}
      </FormField>
      <FormField label="并行组">
        <input style={inp} value={data.parallelGroup || ''} placeholder="相同组名并行执行"
          onChange={(e) => onUpdate(node.id, { parallelGroup: e.target.value } as any)} />
      </FormField>
    </>
  );
}

// ============================================================
// Tool 属性
// ============================================================

function ToolPanel({ node, data, onUpdate, nodes, edges }: {
  node: Node; data: ToolNodeData;
  onUpdate: (id: string, data: Partial<AppNodeData>) => void;
  nodes: Node[]; edges: Edge[];
}) {
  const duplicates = findDuplicateOutputVars(nodes);
  const conflict = data.outputVar ? duplicates.get(data.outputVar) : undefined;
  const isConflict = conflict && conflict.length > 1 && conflict.includes(node.id);

  const templateVars = data.toolParams
    ? resolveTemplateVars(data.toolParams, node.id, nodes, edges)
    : [];
  const undefinedVars = templateVars.filter((v) => !v.defined);

  return (
    <>
      <PurposeField node={node} data={data} onUpdate={onUpdate} />
      <FormField label="工具名称">
        <input style={inp} value={data.toolName || ''} placeholder="如 kb_search"
          onChange={(e) => onUpdate(node.id, { toolName: e.target.value, label: e.target.value } as any)} />
      </FormField>
      <FormField label="参数（JSON / 模板）">
        <textarea style={txt} value={data.toolParams || ''}
          placeholder={'{"query": "{{user_query}}", "top_k": 5}'}
          onChange={(e) => onUpdate(node.id, { toolParams: e.target.value } as any)} />
        <VarSuggest onSelect={(v) => onUpdate(node.id, { toolParams: (data.toolParams || '') + `{{${v}}}` } as any)}
          nodeId={node.id} nodes={nodes} edges={edges} />
        {undefinedVars.length > 0 && (
          <div style={{ marginTop: 6 }}>
            {undefinedVars.map((v) => (
              <div key={v.name} style={{ fontSize: 11, color: '#e67e22', padding: '3px 0' }}>
                ⚠️ <code>{`{{${v.name}}}`}</code> 上游未定义
              </div>
            ))}
          </div>
        )}
        {templateVars.filter((v) => v.defined).length > 0 && (
          <div style={{ marginTop: 4 }}>
            {templateVars.filter((v) => v.defined).map((v) => (
              <div key={v.name} style={{ fontSize: 11, color: '#2e7d32', padding: '2px 0' }}>
                ✓ <code>{`{{${v.name}}}`}</code> 来自 {v.definedByLabel || v.definedBy}
              </div>
            ))}
          </div>
        )}
      </FormField>
      <FormField label="输出变量名">
        <input style={{ ...inp, borderColor: isConflict ? '#e53935' : '#e0e3e8' }} value={data.outputVar || ''} placeholder="如 tool_result"
          onChange={(e) => onUpdate(node.id, { outputVar: e.target.value } as any)} />
        {isConflict && (
          <div style={{ fontSize: 11, color: '#e53935', marginTop: 4 }}>
            ✗ 变量名 "{data.outputVar}" 与 {conflict!.filter((id) => id !== node.id).join(', ')} 冲突
          </div>
        )}
        {!isConflict && data.outputVar && (
          <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>下游可通过 {`{{${data.outputVar}}}`} 引用</div>
        )}
      </FormField>
      <FormField label="并行组">
        <input style={inp} value={data.parallelGroup || ''} placeholder="相同组名并行执行"
          onChange={(e) => onUpdate(node.id, { parallelGroup: e.target.value } as any)} />
      </FormField>
    </>
  );
}

// ============================================================
// Variable 属性
// ============================================================

function VarPanel({ node, data, onUpdate }: { node: Node; data: VariableNodeData; onUpdate: (id: string, data: Partial<AppNodeData>) => void }) {
  return (
    <>
      <PurposeField node={node} data={data} onUpdate={onUpdate} />
      <FormField label="变量名">
        <input style={inp} value={data.varName || ''} placeholder="如 my_variable, user_name"
          onChange={(e) => {
            const vn = e.target.value;
            onUpdate(node.id, { varName: vn, label: vn || '变量' } as any);
          }} />
        <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>
          变量名自由定义，下游节点通过 <code>{`{{${data.varName || '变量名'}}}`}</code> 引用
        </div>
      </FormField>
      <FormField label="说明（可选）">
        <input style={inp} value={data.varDesc || ''} placeholder="这个变量存什么？"
          onChange={(e) => onUpdate(node.id, { varDesc: e.target.value } as any)} />
      </FormField>
    </>
  );
}

// ============================================================
// Classifier 属性
// ============================================================

function BranchPanel({ node, data, onUpdate }: { node: Node; data: BranchNodeData; onUpdate: (id: string, data: Partial<AppNodeData>) => void }) {
  return (
    <>
      <PurposeField node={node} data={data} onUpdate={onUpdate} />
      <FormField label="分支依据变量">
        <input style={inp} value={data.inputVar || ''} placeholder="如 intent, user_query"
          onChange={(e) => onUpdate(node.id, { inputVar: e.target.value } as any)} />
        <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>
          根据此变量的值走不同分支，在连线上标注条件（如 emergency / default）
        </div>
      </FormField>
    </>
  );
}

// ============================================================
// 表单字段包装
// ============================================================

function FormField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={sect}>
      <div style={lab}>{label}</div>
      {children}
    </div>
  );
}

// ============================================================
// 变量快速插入
// ============================================================

function VarSuggest({ onSelect, nodeId, nodes, edges }: {
  onSelect: (varName: string) => void;
  nodeId: string;
  nodes: Node[];
  edges: Edge[];
}) {
  const [show, setShow] = useState(false);
  const upstream = getUpstreamVars(nodeId, nodes, edges);

  return (
    <div style={{ marginTop: 6 }}>
      <button style={{ ...btnBase, fontSize: 10, padding: '4px 10px' }} onClick={() => setShow(!show)}>
        {show ? '收起' : '+ 插入 {{变量名}}'}
      </button>
      {show && (
        <div style={{ marginTop: 6 }}>
          {/* 手动输入 */}
          <input style={inp} placeholder="自定义变量名，回车插入"
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                const v = (e.target as HTMLInputElement).value.trim();
                if (v) {
                  onSelect(v);
                  (e.target as HTMLInputElement).value = '';
                  setShow(false);
                }
              }
            }} />
          {/* 上游节点变量列表 */}
          {upstream.length > 0 && (
            <div style={{ marginTop: 6 }}>
              <div style={{ fontSize: 10, color: '#999', marginBottom: 4 }}>上游可用变量：</div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 3 }}>
                {upstream.map((v) => (
                  <span key={v.name} style={chipStyle} onClick={() => { onSelect(v.name); setShow(false); }} title={`来自 ${v.label}`}>
                    {`{{${v.name}}}`}
                  </span>
                ))}
              </div>
            </div>
          )}
          {upstream.length === 0 && (
            <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>
              尚无上游变量，可手动输入变量名
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================
// 节点用途说明（自描述核心字段 — AI 读这个理解节点）
// ============================================================

function PurposeField({ node, data, onUpdate }: { node: Node; data: AppNodeData; onUpdate: (id: string, data: Partial<AppNodeData>) => void }) {
  const purpose = (data as any).purpose || '';
  return (
    <FormField label="💡 节点用途">
      <textarea style={txt} value={purpose}
        placeholder={'这个节点做什么？用一句话说明。\n例如：把用户问题翻译成英文再交给下游'}
        onChange={(e) => onUpdate(node.id, { purpose: e.target.value } as any)} />
      <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>导出 JSON 时作为该节点的自描述说明</div>
    </FormField>
  );
}

// ============================================================
// Note 便签属性
// ============================================================

const NOTE_COLORS = [
  { value: 'yellow', label: '黄', bg: '#fff9c4' },
  { value: 'green', label: '绿', bg: '#c8e6c9' },
  { value: 'blue', label: '蓝', bg: '#bbdefb' },
  { value: 'pink', label: '粉', bg: '#f8bbd0' },
  { value: 'purple', label: '紫', bg: '#e1bee7' },
];

function NotePanel({ node, data, onUpdate }: { node: Node; data: NoteNodeData; onUpdate: (id: string, data: Partial<AppNodeData>) => void }) {
  return (
    <>
      <FormField label="标题">
        <input style={inp} value={data.label || ''} placeholder="如：TODO、注意、想法"
          onChange={(e) => onUpdate(node.id, { label: e.target.value } as any)} />
      </FormField>
      <FormField label="内容">
        <textarea style={{ ...txt, minHeight: 120 }} value={data.content || ''}
          placeholder="写在这里...（不参与流程逻辑）"
          onChange={(e) => onUpdate(node.id, { content: e.target.value } as any)} />
      </FormField>
      <FormField label="颜色">
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {NOTE_COLORS.map((c) => (
            <span key={c.value}
              onClick={() => onUpdate(node.id, { color: c.value } as any)}
              title={c.label}
              style={{
                width: 26, height: 26, borderRadius: 6, cursor: 'pointer',
                background: c.bg,
                border: data.color === c.value ? '3px solid #4b6cb7' : '2px solid #e0e3e8',
                display: 'inline-block',
              }} />
          ))}
        </div>
      </FormField>
    </>
  );
}

// ============================================================
// 边条件编辑
// ============================================================

function EdgePanel({ edge, onUpdateEdge }: { edge: Edge; onUpdateEdge: (edgeId: string, updates: Partial<Edge>) => void }) {
  const label = typeof edge.label === 'string' ? edge.label : '';
  return (
    <>
      <FormField label="条件标签">
        <input style={inp} value={label || ''}
          placeholder="如 emergency / default，空=无条件"
          onChange={(e) => onUpdateEdge(edge.id, { label: e.target.value || undefined } as any)} />
        <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>
          分支节点根据此条件决定走哪条路径。标记为 <code>default</code> 表示兜底分支。
        </div>
      </FormField>
      <div style={{ ...sect, borderBottom: 'none' }}>
        <div style={{ fontSize: 10, color: '#999', lineHeight: 1.6 }}>
          来源: <code>{edge.source}</code> → 目标: <code>{edge.target}</code>
        </div>
      </div>
    </>
  );
}
