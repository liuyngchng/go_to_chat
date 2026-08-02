import React, { useState } from 'react';
import type { Node } from '@xyflow/react';
import type { AppNodeData, AgentNodeData, ToolNodeData, VariableNodeData, ClassifierNodeData, IntentCategory } from './types';
import { SYSTEM_VARS, LEGACY_VARS } from './types';

// ============================================================
// 样式常量
// ============================================================

const panel: React.CSSProperties = {
  width: 300, background: '#14141f', borderLeft: '1px solid #1e1e30',
  display: 'flex', flexDirection: 'column', overflow: 'hidden',
};

const hdr: React.CSSProperties = {
  fontSize: 14, fontWeight: 700, color: '#e2e4ed',
  padding: '16px 18px', borderBottom: '1px solid #1e1e30',
};

const emptyMsg: React.CSSProperties = {
  padding: 32, color: '#5b5d78', fontSize: 13, textAlign: 'center', lineHeight: 1.7,
};

const sect: React.CSSProperties = { padding: '14px 18px', borderBottom: '1px solid #1e1e30' };

const lab: React.CSSProperties = {
  fontSize: 11, fontWeight: 600, color: '#8b8fa5', marginBottom: 6,
  textTransform: 'uppercase', letterSpacing: '0.05em',
};

const inp: React.CSSProperties = {
  width: '100%', boxSizing: 'border-box', padding: '8px 11px',
  borderRadius: 7, border: '1px solid #2a2a3d', background: '#1c1c2a',
  color: '#e2e4ed', fontSize: 12, fontFamily: 'inherit',
  transition: 'border-color 0.15s',
};

const txt: React.CSSProperties = { ...inp, resize: 'vertical', minHeight: 64, fontFamily: '"JetBrains Mono", "Fira Code", monospace' };

const sel: React.CSSProperties = { ...inp, appearance: 'none', cursor: 'pointer', backgroundImage: 'none' };

const chipStyle: React.CSSProperties = {
  display: 'inline-flex', alignItems: 'center', gap: 4,
  padding: '3px 9px', borderRadius: 5, background: '#252538',
  color: '#c5c6d4', fontSize: 10, fontFamily: '"JetBrains Mono", monospace',
  marginRight: 4, marginBottom: 4, cursor: 'pointer', border: '1px solid #2a2a3d',
};

const btnBase: React.CSSProperties = {
  padding: '7px 14px', borderRadius: 7, border: '1px solid #2a2a3d',
  background: '#1c1c2a', color: '#c5c6d4', cursor: 'pointer', fontSize: 11,
  fontWeight: 500, fontFamily: 'inherit', transition: 'all 0.12s',
  marginRight: 6, marginTop: 4,
};

const dangerBtn: React.CSSProperties = {
  ...btnBase, borderColor: '#f8717120', color: '#f87171', background: '#f871710c',
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
        <div style={emptyMsg}>
          点击画布上的节点<br />查看和编辑属性
        </div>
      </div>
    );
  }

  const data = node.data as unknown as AppNodeData;

  return (
    <div style={panel}>
      <div style={hdr}>
        节点属性
        <div style={{ fontSize: 10, color: '#5b5d78', fontWeight: 400, marginTop: 3 }}>
          {node.id} <span style={{ margin: '0 4px', color: '#2a2a3d' }}>·</span> {data.nodeType}
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto' }}>
        {data.nodeType === 'agent' && <AgentPanel node={node} data={data} onUpdate={onUpdate} />}
        {data.nodeType === 'tool' && <ToolPanel node={node} data={data} onUpdate={onUpdate} />}
        {data.nodeType === 'variable' && <VarPanel node={node} data={data} onUpdate={onUpdate} />}
        {data.nodeType === 'classifier' && <ClassifierPanel node={node} data={data} onUpdate={onUpdate} />}
        {data.nodeType === 'start' && (
          <div style={sect}>
            <div style={{ color: '#8b8fa5', fontSize: 12, lineHeight: 1.6 }}>
              入口节点 — 用户问题从这里输入，流向后续处理节点。
            </div>
          </div>
        )}
      </div>

      {data.nodeType !== 'start' && (
        <div style={{ padding: '12px 18px', borderTop: '1px solid #1e1e30' }}>
          <button style={dangerBtn} onClick={() => { if (confirm(`确定删除节点？`)) onDelete(node.id); }}>
            删除节点
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
      <div style={sect}>
        <div style={lab}>Agent ID</div>
        <input style={inp} type="number" value={data.agentId || ''} placeholder="数据库 ID"
          onChange={(e) => onUpdate(node.id, { agentId: parseInt(e.target.value) || 0 } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>名称</div>
        <input style={inp} value={data.agentName || ''} placeholder="如：客服助手"
          onChange={(e) => onUpdate(node.id, { agentName: e.target.value } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>输入模板</div>
        <textarea style={txt} value={data.inputTemplate || ''}
          placeholder={'使用 {{变量名}} 引用上游输出\n例：用户问题：{{sys.user_query}}'}
          onChange={(e) => onUpdate(node.id, { inputTemplate: e.target.value } as any)} />
        <VarSuggest onSelect={(v) => onUpdate(node.id, { inputTemplate: (data.inputTemplate || '') + `{{${v}}}` } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>输出变量名</div>
        <input style={inp} value={data.outputVar || ''} placeholder="如 my_output"
          onChange={(e) => onUpdate(node.id, { outputVar: e.target.value } as any)} />
        <div style={{ fontSize: 10, color: '#5b5d78', marginTop: 4 }}>
          下游可通过 {`{{${data.outputVar || 'xxx'}}}`} 引用
        </div>
      </div>
      <div style={sect}>
        <div style={lab}>触发条件</div>
        <input style={inp} value={data.condition || ''} placeholder="如 emergency，空=无条件执行"
          onChange={(e) => onUpdate(node.id, { condition: e.target.value } as any)} />
        <div style={{ fontSize: 10, color: '#5b5d78', marginTop: 4 }}>匹配意图分类结果时执行</div>
      </div>
      <div style={sect}>
        <div style={lab}>并行组</div>
        <input style={inp} value={data.parallelGroup || ''} placeholder="相同组名并行执行"
          onChange={(e) => onUpdate(node.id, { parallelGroup: e.target.value } as any)} />
      </div>
    </>
  );
}

// ============================================================
// Tool 属性
// ============================================================

function ToolPanel({ node, data, onUpdate }: { node: Node; data: ToolNodeData; onUpdate: Props['onUpdate'] }) {
  return (
    <>
      <div style={sect}>
        <div style={lab}>工具名称</div>
        <input style={inp} value={data.toolName || ''} placeholder="如 kb_search"
          onChange={(e) => onUpdate(node.id, { toolName: e.target.value } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>参数（JSON / 模板）</div>
        <textarea style={txt} value={data.toolParams || ''}
          placeholder={'{"query": "{{sys.user_query}}", "top_k": 5}'}
          onChange={(e) => onUpdate(node.id, { toolParams: e.target.value } as any)} />
        <VarSuggest onSelect={(v) => onUpdate(node.id, { toolParams: (data.toolParams || '') + `{{${v}}}` } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>输出变量名</div>
        <input style={inp} value={data.outputVar || ''} placeholder="如 tool_result"
          onChange={(e) => onUpdate(node.id, { outputVar: e.target.value } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>触发条件</div>
        <input style={inp} value={data.condition || ''} placeholder="如 emergency"
          onChange={(e) => onUpdate(node.id, { condition: e.target.value } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>并行组</div>
        <input style={inp} value={data.parallelGroup || ''} placeholder="相同组名并行执行"
          onChange={(e) => onUpdate(node.id, { parallelGroup: e.target.value } as any)} />
      </div>
    </>
  );
}

// ============================================================
// Variable 属性
// ============================================================

function VarPanel({ node, data, onUpdate }: { node: Node; data: VariableNodeData; onUpdate: Props['onUpdate'] }) {
  return (
    <div style={sect}>
      <div style={lab}>系统变量</div>
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
        <div style={{ fontSize: 11, color: '#8b8fa5', marginTop: 8, fontFamily: '"JetBrains Mono", monospace' }}>
          {`下游可用 {{${data.varName}}} 引用`}
        </div>
      )}
    </div>
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
    const newCat: IntentCategory = { name, description: desc, keywords };
    onUpdate(node.id, { categories: [...cats, newCat] } as any);
  };

  const removeCat = (idx: number) => {
    onUpdate(node.id, { categories: cats.filter((_, i) => i !== idx) } as any);
  };

  return (
    <>
      <div style={sect}>
        <div style={lab}>分类 Prompt</div>
        <textarea style={txt} value={data.prompt || ''} placeholder="LLM 分类提示词..."
          onChange={(e) => onUpdate(node.id, { prompt: e.target.value } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>输出变量名</div>
        <input style={inp} value={data.outputVar || 'intent'} placeholder="如 intent"
          onChange={(e) => onUpdate(node.id, { outputVar: e.target.value } as any)} />
      </div>
      <div style={sect}>
        <div style={lab}>意图类别 ({cats.length})</div>
        {cats.map((cat, i) => (
          <div key={i} style={{
            display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start',
            padding: '8px 11px', marginBottom: 4, borderRadius: 7,
            background: '#1c1c2a', border: '1px solid #2a2a3d',
          }}>
            <div>
              <div style={{ fontWeight: 600, color: '#c4b5fd', fontSize: 12 }}>{cat.name}</div>
              <div style={{ fontSize: 10, color: '#8b8fa5', marginTop: 1 }}>{cat.description}</div>
              {cat.keywords.length > 0 && (
                <div style={{ fontSize: 9, color: '#5b5d78', marginTop: 3 }}>
                  {cat.keywords.join(', ')}
                </div>
              )}
            </div>
            <span onClick={() => removeCat(i)} style={{ cursor: 'pointer', color: '#f87171', fontSize: 16, lineHeight: 1 }}>×</span>
          </div>
        ))}
        <button style={btnBase} onClick={addCat}>+ 添加类别</button>
      </div>
    </>
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
