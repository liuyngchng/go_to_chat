// 聊天元素
const chatContainer = document.getElementById('chat-container');
const queryForm = document.getElementById('query-form');
const queryInput = document.getElementById('query-input');
const sendButton = document.getElementById('send-button');
const stopButton = document.getElementById('stop-button');

let isFetching = false;
let abortController = null;
let currentBotMessage = null;

const MAX_MESSAGES = 50;
let messages = [];

// 会话 ID
let sessionId = crypto.randomUUID ? crypto.randomUUID() : Date.now().toString(36) + Math.random().toString(36).slice(2);

// localStorage key
function getStorageKey() {
    const uidEl = document.getElementById('uid');
    const uid = uidEl ? uidEl.value : 'default';
    return 'chat2kb_messages_' + uid;
}

function saveMessages() {
    if (messages.length > MAX_MESSAGES) {
        messages = messages.slice(-MAX_MESSAGES);
    }
    localStorage.setItem(getStorageKey(), JSON.stringify(messages));
}

function loadMessages() {
    try {
        const raw = localStorage.getItem(getStorageKey());
        if (raw) {
            messages = JSON.parse(raw);
            if (messages.length > MAX_MESSAGES) {
                messages = messages.slice(-MAX_MESSAGES);
            }
        }
    } catch (e) {
        messages = [];
    }
}

function restoreMessages() {
    chatContainer.innerHTML = '';
    messages.forEach(m => addMessageToDOM(m.text, m.type));
}

// 页面加载
window.onload = function() {
    loadMessages();
    if (messages.length > 0) {
        restoreMessages();
    }
    queryInput.focus();
};

// 表单提交
queryForm.addEventListener('submit', async function(e) {
    e.preventDefault();
    if (isFetching) return;

    const query = queryInput.value.trim();
    if (!query) return;

    // 用户消息
    addMessage(query, 'user');
    queryInput.value = '';
    queryInput.focus();

    try {
        await fetchQueryData(query);
    } catch (error) {
        console.error('请求出错:', error);
        if (currentBotMessage) {
            updateBotMessage('抱歉，生成回复时出错，请重试。');
        }
        resetUI();
    }
});

// 停止按钮
stopButton.addEventListener('click', function() {
    if (abortController) {
        abortController.abort();
    }

    if (currentBotMessage) {
        const messageBubble = currentBotMessage.querySelector('.bot-message-bubble');
        if (messageBubble) {
            const typingIndicator = messageBubble.querySelector('.typing-indicator');
            if (typingIndicator) {
                typingIndicator.remove();
                const stopNotice = document.createElement('div');
                stopNotice.className = 'stop-notice';
                stopNotice.textContent = '已停止生成';
                messageBubble.appendChild(stopNotice);
            }
        }
    }
    resetUI();
});

// 流式请求
async function fetchQueryData(query) {
    isFetching = true;
    sendButton.disabled = true;
    stopButton.style.display = 'inline-block';

    abortController = new AbortController();

    // 加载中动画
    currentBotMessage = addMessage('<div class="typing-indicator"><span></span><span></span><span></span> 思考中...</div>', 'bot');

    try {
        const uid = document.getElementById('uid').value;
        const appSource = document.getElementById('app_source').value;

        const formData = new URLSearchParams();
        formData.append('msg', query);
        formData.append('uid', uid);
        formData.append('app_source', appSource);
        formData.append('session_id', sessionId);

        const response = await fetch('/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
                'Accept': 'text/event-stream'
            },
            body: formData.toString(),
            signal: abortController.signal
        });

        if (!response.ok) {
            throw new Error('网络请求失败');
        }
        if (!response.body) {
            throw new Error('不支持流式响应');
        }

        // 读取 SSE 流
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let accumulatedText = '';

        while (true) {
            const { value, done } = await reader.read();
            if (done) break;

            const chunk = decoder.decode(value, { stream: true });
            const lines = chunk.split('\n');

            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const data = line.substring(6);
                    if (data === '[DONE]' || data === '') continue;
                    accumulatedText += data;
                    updateBotMessage(accumulatedText);
                }
            }
        }

        // 添加复制按钮
        addCopyButton(currentBotMessage, accumulatedText);

    } catch (error) {
        if (error.name === 'AbortError') {
            console.log('请求已中止');
        } else {
            console.error('请求出错:', error);
            if (currentBotMessage) {
                updateBotMessage('请求失败，请重试。');
            }
        }
    } finally {
        resetUI();
    }
}

// 更新机器人消息
function updateBotMessage(text) {
    if (!currentBotMessage) return;

    const messageBubble = currentBotMessage.querySelector('.bot-message-bubble');
    if (messageBubble) {
        const sanitizedContent = DOMPurify.sanitize(
            marked.parse(text),
            { ADD_TAGS: ['canvas'], ADD_ATTR: ['id'] }
        );
        messageBubble.innerHTML = sanitizedContent;
        messageBubble.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }

    // 同步更新持久化
    if (messages.length > 0 && messages[messages.length - 1].type === 'bot') {
        messages[messages.length - 1].text = text;
        saveMessages();
    }
}

// 添加消息到 DOM
function addMessageToDOM(text, type) {
    const messageContainer = document.createElement('div');
    messageContainer.classList.add('message-container');

    if (type === 'user') {
        messageContainer.classList.add('user-message-container');
        const sanitized = DOMPurify.sanitize(text);
        messageContainer.innerHTML = '<div class="message-bubble user-message-bubble">' + sanitized + '</div>';
    } else {
        messageContainer.classList.add('bot-message-container');
        const sanitized = DOMPurify.sanitize(
            marked.parse(text),
            { ADD_TAGS: ['canvas'], ADD_ATTR: ['id'] }
        );
        messageContainer.innerHTML =
            '<div class="bot-message-header"><span><i class="fas fa-robot"></i> AI助手</span></div>' +
            '<div class="message-bubble bot-message-bubble">' + sanitized + '</div>';
    }

    chatContainer.appendChild(messageContainer);
    messageContainer.scrollIntoView({ behavior: 'smooth' });
    return messageContainer;
}

// 添加消息（持久化 + DOM）
function addMessage(text, type) {
    messages.push({ text, type });
    saveMessages();
    return addMessageToDOM(text, type);
}

// 复制按钮
function addCopyButton(messageContainer, text) {
    const actionsContainer = document.createElement('div');
    actionsContainer.classList.add('message-actions');

    const copyButton = document.createElement('button');
    copyButton.classList.add('copy-button');
    copyButton.innerHTML = '<i class="fas fa-copy"></i> 复制';
    copyButton.onclick = function() {
        navigator.clipboard.writeText(text).then(() => {
            copyButton.innerHTML = '<i class="fas fa-check"></i> 已复制';
            setTimeout(() => {
                copyButton.innerHTML = '<i class="fas fa-copy"></i> 复制';
            }, 2000);
        });
    };

    actionsContainer.appendChild(copyButton);
    messageContainer.appendChild(actionsContainer);
}

// 新对话
async function newChat() {
    if (isFetching && abortController) {
        abortController.abort();
    }
    const uid = document.getElementById('uid').value;
    try {
        const formData = new URLSearchParams();
        formData.append('session_id', sessionId);
        formData.append('uid', uid);
        await fetch('/chat/clear', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: formData.toString()
        });
    } catch (e) {
        console.warn('清空上下文失败:', e);
    }
    sessionId = crypto.randomUUID ? crypto.randomUUID() : Date.now().toString(36) + Math.random().toString(36).slice(2);
    messages = [];
    saveMessages();
    chatContainer.innerHTML = '';
    resetUI();
}

// 重置 UI
function resetUI() {
    isFetching = false;
    sendButton.disabled = false;
    stopButton.style.display = 'none';
    abortController = null;
}

// 新对话按钮
const newChatBtn = document.getElementById('newChat');
if (newChatBtn) {
    newChatBtn.addEventListener('click', newChat);
}

// 键盘快捷键
queryInput.addEventListener('keydown', function(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        queryForm.dispatchEvent(new Event('submit'));
    }
});
