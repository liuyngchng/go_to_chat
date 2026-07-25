// 知识库管理页 JS

const uid = document.getElementById('uid').value;
let currentKbId = null;
let selectedFiles = [];
let refreshInterval = null;

// ============================================================
// 知识库列表
// ============================================================

function getFormData() {
    const fd = new FormData();
    fd.append('uid', uid);
    return fd;
}

async function refreshKbList() {
    const sel = document.getElementById('kb_selector');
    const refreshBtn = document.getElementById('kbRefreshBtn');
    refreshBtn.disabled = true;

    try {
        const fd = getFormData();
        const resp = await fetch('/vdb/my/list', { method: 'POST', body: fd });
        const data = await resp.json();

        sel.innerHTML = '<option value="">请选择知识库</option>';
        if (data.data) {
            data.data.forEach(kb => {
                const opt = document.createElement('option');
                opt.value = kb.id;
                opt.textContent = kb.name + (kb.is_default ? ' ★' : '');
                sel.appendChild(opt);
            });
        }
    } catch (e) {
        console.error('获取知识库列表失败:', e);
    }

    refreshBtn.disabled = false;
}

// KB 选择事件
document.getElementById('kb_selector').addEventListener('change', function() {
    const id = this.value;
    currentKbId = id ? parseInt(id) : null;

    // 清除旧的轮询定时器
    clearInterval(refreshInterval);
    refreshInterval = null;

    document.getElementById('deleteBtn').style.display = id ? 'inline-block' : 'none';
    document.getElementById('setDefaultBtn').style.display = id ? 'inline-block' : 'none';

    if (id) {
        loadFileList(parseInt(id));
        const selected = this.options[this.selectedIndex];
        document.getElementById('vdb_status_desc').textContent = selected.textContent;
        // 启动 5 秒轮询自动刷新文件处理进度
        refreshInterval = setInterval(() => loadFileList(parseInt(id)), 5000);
    } else {
        document.getElementById('fileListContainer').style.display = 'none';
        document.getElementById('vdb_status_desc').textContent = '未选择';
        document.getElementById('public_badge').style.display = 'none';
        document.getElementById('default_badge').style.display = 'none';
    }
});

// 刷新按钮
document.getElementById('kbRefreshBtn').addEventListener('click', refreshKbList);

// 创建知识库
document.getElementById('createKB').addEventListener('click', async function() {
    const name = document.getElementById('kb_name').value.trim();
    if (!name) {
        alert('请输入知识库名称');
        return;
    }

    const isPublic = document.getElementById('public_checkbox').checked;
    const fd = getFormData();
    fd.append('name', name);
    fd.append('is_public', isPublic ? 'true' : 'false');

    try {
        const resp = await fetch('/vdb/create', { method: 'POST', body: fd });
        const data = await resp.json();
        if (data.status === 'ok') {
            document.getElementById('kb_name').value = '';
            document.getElementById('public_checkbox').checked = false;
            refreshKbList();
            showStatus('知识库创建成功！');
        } else {
            alert(data.error || '创建失败');
        }
    } catch (e) {
        console.error('创建知识库失败:', e);
    }
});

// 删除知识库
document.getElementById('deleteBtn').addEventListener('click', async function() {
    if (!currentKbId || !confirm('确定要删除该知识库吗？此操作不可恢复。')) return;

    const fd = getFormData();
    fd.append('id', currentKbId);

    try {
        const resp = await fetch('/vdb/delete', { method: 'POST', body: fd });
        const data = await resp.json();
        if (data.status === 'ok') {
            refreshKbList();
            document.getElementById('fileListContainer').style.display = 'none';
            document.getElementById('vdb_status_desc').textContent = '未选择';
            showStatus('知识库已删除');
        } else {
            alert(data.error || '删除失败');
        }
    } catch (e) {
        console.error('删除知识库失败:', e);
    }
});

// 设为默认
document.getElementById('setDefaultBtn').addEventListener('click', async function() {
    if (!currentKbId) return;

    const fd = getFormData();
    fd.append('id', currentKbId);

    try {
        const resp = await fetch('/vdb/set/default', { method: 'POST', body: fd });
        const data = await resp.json();
        if (data.status === 'ok') {
            refreshKbList();
            showStatus('已设为默认知识库');
        } else {
            alert(data.error || '设置失败');
        }
    } catch (e) {
        console.error('设置默认失败:', e);
    }
});

// ============================================================
// 文件列表
// ============================================================

async function loadFileList(vdbId) {
    const fd = getFormData();
    fd.append('vdb_id', vdbId);

    try {
        const resp = await fetch('/vdb/file/list', { method: 'POST', body: fd });
        const data = await resp.json();
        renderFileList(data.data || []);
        document.getElementById('fileListContainer').style.display = 'block';
    } catch (e) {
        console.error('获取文件列表失败:', e);
    }
}

function renderFileList(files) {
    const tbody = document.querySelector('#fileListTable tbody');
    tbody.innerHTML = '';

    files.forEach((f, i) => {
        const tr = document.createElement('tr');
        const progressPct = Math.round(f.percent || 0);
        tr.innerHTML =
            '<td>' + (i + 1) + '</td>' +
            '<td>' + escapeHtml(f.name) + '</td>' +
            '<td>' + formatTime(f.create_time) + '</td>' +
            '<td>' + progressPct + '%</td>' +
            '<td>' + escapeHtml(f.process_info || '') + '</td>' +
            '<td><button class="btn btn-primary btn-sm" onclick="deleteFile(' + f.id + ')">' +
            '<i class="fas fa-trash-alt"></i> 删除</button></td>';
        tbody.appendChild(tr);
    });
}

async function deleteFile(fileId) {
    if (!confirm('确定要删除该文件吗？')) return;

    const fd = getFormData();
    fd.append('file_id', fileId);

    try {
        const resp = await fetch('/vdb/file/delete', { method: 'POST', body: fd });
        const data = await resp.json();
        if (data.status === 'ok' && currentKbId) {
            loadFileList(currentKbId);
        }
    } catch (e) {
        console.error('删除文件失败:', e);
    }
}

// ============================================================
// 文件上传
// ============================================================

document.getElementById('selectBtn').addEventListener('click', function() {
    document.getElementById('fileInput').click();
});

document.getElementById('fileInput').addEventListener('change', function() {
    selectedFiles = Array.from(this.files);
    document.getElementById('fileCount').textContent = selectedFiles.length;

    const itemsDiv = document.getElementById('fileItems');
    itemsDiv.innerHTML = selectedFiles.map(f => '<div>' + escapeHtml(f.name) + '</div>').join('');
});

document.getElementById('clearFilesBtn').addEventListener('click', function() {
    selectedFiles = [];
    document.getElementById('fileInput').value = '';
    document.getElementById('fileCount').textContent = '0';
    document.getElementById('fileItems').innerHTML = '';
});

document.getElementById('startBtn').addEventListener('click', async function() {
    if (!currentKbId) {
        alert('请先选择知识库');
        return;
    }
    if (selectedFiles.length === 0) {
        alert('请先选择文件');
        return;
    }

    const progressDiv = document.getElementById('uploadProgress');
    const progressFill = document.getElementById('overallProgressFill');
    const progressText = document.getElementById('progressText');
    const progressPercent = document.getElementById('progressPercent');

    progressDiv.style.display = 'block';

    for (let i = 0; i < selectedFiles.length; i++) {
        const file = selectedFiles[i];
        const fd = getFormData();
        fd.append('vdb_id', currentKbId);
        fd.append('file', file);

        try {
            const resp = await fetch('/vdb/upload', { method: 'POST', body: fd });
            const data = await resp.json();
            if (data.status !== 'ok') {
                document.getElementById('fileUploadResult').textContent = '上传 ' + file.name + ' 失败: ' + (data.error || '未知错误');
            } else {
                document.getElementById('fileUploadResult').textContent = '上传 ' + file.name + ' 成功，后台处理中...';
            }
        } catch (e) {
            console.error('上传失败:', e);
            document.getElementById('fileUploadResult').textContent = '上传失败: ' + e.message;
        }

        const pct = Math.round((i + 1) / selectedFiles.length * 100);
        progressFill.style.width = pct + '%';
        progressText.textContent = '已上传 ' + (i + 1) + '/' + selectedFiles.length;
        progressPercent.textContent = pct + '%';
    }

    // 刷新文件列表
    loadFileList(currentKbId);
    // 确保轮询已启动（如果还没启动）
    if (!refreshInterval && currentKbId) {
        refreshInterval = setInterval(() => loadFileList(currentKbId), 5000);
    }
    selectedFiles = [];
    document.getElementById('fileInput').value = '';
    document.getElementById('fileCount').textContent = '0';
    document.getElementById('fileItems').innerHTML = '';
});

// ============================================================
// 辅助函数
// ============================================================

function showStatus(msg) {
    const el = document.getElementById('kb_status');
    el.style.display = 'block';
    el.querySelector('span').textContent = msg;
    setTimeout(() => { el.style.display = 'none'; }, 3000);
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatTime(ts) {
    if (!ts) return '';
    const d = new Date(ts);
    return d.toLocaleString('zh-CN');
}

// 页面初始化
refreshKbList();

// 页面离开时清理定时器
window.addEventListener('beforeunload', () => clearInterval(refreshInterval));
