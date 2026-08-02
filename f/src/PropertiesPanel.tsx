import React, { useState } from 'react';
import type { Node } from '@xyflow/react';
import type { AppNodeData, AgentNodeData, ToolNodeData, VariableNodeData, ClassifierNodeData, IntentCategory } from './types';
import { SYSTEM_VARS, LEGACY_VARS } from './types';

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
  onUpdate: (id: string, data: Partial<AppNodeData>) => void;
  onDelete: (id: string) => void;
}

export function PropertiesPanel({ node, onUpdate, onDelete }: Props) {
  if (!node) {
    return (
      <div style={panel}>
        <div style={hdr}>属性面板</div>
        <div style={emptyMsg}>点击画布上的节点<br />查看和编辑属性</div>
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
        {data.nodeType === 'agent' && <AgentPanel node={node} data={data} onUpdate={onUpdate} />}
        {data.nodeType === 'tool' && <ToolPanel node={node} data={data} onUpdate={onUpdate} />}
        {data.nodeType === 'variable' && <VarPanel node={node} data={data} onUpdate={onUpdate} />}
        {data.nodeType === 'classifier' && <ClassifierPanel node={node} data={data} onUpdate={onUpdate} />}
        {data.nodeType === 'start' && (
          <div style={sect}>
            <div style={{ color: '#888', fontSize: 12, lineHeight: 1.6 }}>
              入口节点 — 用户问题从这里输入，流向后续处理节点。
            </div>
          </div>
        )}
      </div>

      {data.nodeType !== 'start' && (
        <div style={{ padding: '12px 18px', borderTop: '1px solid #f0f0f0' }}>
          <button style={dangerBtn} onClick={() => { if (confirm('确定删除节点？')) onDelete(node.id); }}>
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

function AgentPanel({ node, data, onUpdate }: { node: Node; data: AgentNodeData; onUpdate: Props['onUpdate'] }) {
  return (
    <>
      <FormField label="Agent ID">
        <input style={inp} type="number" value={data.agentId || ''} placeholder="数据库 ID"
          onChange={(e) => onUpdate(node.id, { agentId: parseInt(e.target.value) || 0 } as any)} />
      </FormField>
      <FormField label="名称">
        <input style={inp} value={data.agentName || ''} placeholder="如：客服助手"
          onChange={(e) => onUpdate(node.id, { agentName: e.target.value } as any)} />
      </FormField>
      <FormField label="输入模板">
        <textarea style={txt} value={data.inputTemplate || ''}
          placeholder={'使用 {{变量名}} 引用上游输出\n例：用户问题：{{sys.user_query}}'}
          onChange={(e) => onUpdate(node.id, { inputTemplate: e.target.value } as any)} />
        <VarSuggest onSelect={(v) => onUpdate(node.id, { inputTemplate: (data.inputTemplate || '') + `{{${v}}}` } as any)} />
      </FormField>
      <FormField label="输出变量名">
        <input style={inp} value={data.outputVar || ''} placeholder="如 my_output"
          onChange={(e) => onUpdate(node.id, { outputVar: e.target.value } as any)} />
        <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>下游可通过 {`{{${data.outputVar || 'xxx'}}}`} 引用</div>
      </FormField>
      <FormField label="触发条件">
        <input style={inp} value={data.condition || ''} placeholder="如 emergency，空=无条件执行"
          onChange={(e) => onUpdate(node.id, { condition: e.target.value } as any)} />
        <div style={{ fontSize: 10, color: '#999', marginTop: 4 }}>匹配意图分类结果时执行</div>
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

function ToolPanel({ node, data, onUpdate }: { node: Node; data: ToolNodeData; onUpdate: Props['onUpdate'] }) {
  return (
    <>
      <FormField label="工具名称">
        <input style={inp} value={data.toolName || ''} placeholder="如 kb_search"
          onChange={(e) => onUpdate(node.id, { toolName: e.target.value } as any)} />
      </FormField>
      <FormField label="参数（JSON / 模板）">
        <textarea style={txt} value={data.toolParams || ''}
          placeholder={'{"query": "{{sys.user_query}}", "top_k": 5}'}
          onChange={(e) => onUpdate(node.id, { toolParams: e.target.value } as any)} />
        <VarSuggest onSelect={(v) => onUpdate(node.id, { toolParams: (data.toolParams || '') + `{{${v}}}` } as any)} />
      </FormField>
      <FormField label="输出变量名">
        <input style={inp} value={data.outputVar || ''} placeholder="如 tool_result"
          onChange={(e) => onUpdate(node.id, { outputVar: e.target.value } as any)} />
      </FormField>
      <FormField label="触发条件">
        <input style={inp} value={data.condition || ''} placeholder="如 emergency"
          onChange={(e) => onUpdate(node.id, { condition: e.target.value } as any)} />
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

function VarPanel({ node, data, onUpdate }: { node: Node; data: VariableNodeData; onUpdate: Props['onUpdate'] }) {
  return (
    <FormField label="系统变量">
      <select style={sel} value={data.varName || ''}
        onChange={(e) => {
          const vn = e.target.value;
          const sv = SYSTEM_VARS.find((v) => v.name === vn);
          onUpdate(node.id, { varName: vn, varDesc: sv?.description || '', label: sv?.name || '变量' } as any);
        }}>
        <option value="">— 选择变量 —</option>
        <optgroup label="新版系统变量">
          {SYSTEM_VARS.map((v) => (
            <option key={v.name} value={v.name}>{v.name} — {v.description}</option>
          ))}
        </optgroup>
        <optgroup label="旧版兼容变量">
          {LEGACY_VARS.map((v) => (
            <option key={v.name} value={v.name}>{v.name} — {v.description}</option>
          ))}
        </optgroup>
      </select>
      {data.varName && (
        <div style={{ fontSize: 11, color: '#666', marginTop: 8, fontFamily: 'Consolas, Monaco, monospace' }}>
          {`下游可用 {{${data.varName}}} 引用`}
        </div>
      )}
    </FormField>
  );
}

// ============================================================
// Classifier 属性
// ============================================================

function ClassifierPanel({ node, data, onUpdate }: { node: Node; data: ClassifierNodeData; onUpdate: Props['onUpdate'] }) {
  const cats = data.categories || [];

  const addCat = () => {
    const name = prompt('意图标识（英文）:');
    if (!name) return;
    const desc = prompt('意图描述（中文）:') || '';
    const kws = prompt('关键词（逗号分隔）:') || '';
    const keywords = kws.split(',').map((k: string) => k.trim()).filter(Boolean);
    onUpdate(node.id, { categories: [...cats, { name, description: desc, keywords }] } as any);
  };

  return (
    <>
      <FormField label="分类 Prompt">
        <textarea style={txt} value={data.prompt || ''} placeholder="LLM 分类提示词..."
          onChange={(e) => onUpdate(node.id, { prompt: e.target.value } as any)} />
      </FormField>
      <FormField label="输出变量名">
        <input style={inp} value={data.outputVar || 'intent'} placeholder="如 intent"
          onChange={(e) => onUpdate(node.id, { outputVar: e.target.value } as any)} />
      </FormField>
      <div style={sect}>
        <div style={lab}>意图类别 ({cats.length})</div>
        {cats.map((cat, i) => (
          <div key={i} style={{
            display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start',
            padding: '8px 11px', marginBottom: 4, borderRadius: 6,
            background: '#f8f9fb', border: '1px solid #e8eaed',
          }}>
            <div>
              <div style={{ fontWeight: 600, color: '#4b6cb7', fontSize: 12 }}>{cat.name}</div>
              <div style={{ fontSize: 10, color: '#888', marginTop: 1 }}>{cat.description}</div>
              {cat.keywords.length > 0 && (
                <div style={{ fontSize: 9, color: '#aaa', marginTop: 3 }}>{cat.keywords.join(', ')}</div>
              )}
            </div>
            <span onClick={() => onUpdate(node.id, { categories: cats.filter((_, j) => j !== i) } as any)}
              style={{ cursor: 'pointer', color: '#c62828', fontSize: 16, lineHeight: 1 }}>×</span>
          </div>
        ))}
        <button style={btnBase} onClick={addCat}>
          <i className="fas fa-plus" style={{ marginRight: 4 }} /> 添加类别
        </button>
      </div>
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

function VarSuggest({ onSelect }: { onSelect: (varName: string) => void }) {
  const [show, setShow] = useState(false);
  const allVars = [...SYSTEM_VARS, ...LEGACY_VARS];

  return (
    <div style={{ marginTop: 6 }}>
      <button style={{ ...btnBase, fontSize: 10, padding: '4px 10px' }} onClick={() => setShow(!show)}>
        {show ? '收起' : '+ 插入变量'}
      </button>
      {show && (
        <div style={{ marginTop: 6, display: 'flex', flexWrap: 'wrap', gap: 3 }}>
          {allVars.map((v) => (
            <span key={v.name} style={chipStyle} onClick={() => onSelect(v.name)} title={v.description}>
              {v.name}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
