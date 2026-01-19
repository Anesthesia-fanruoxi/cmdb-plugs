// SQL分析工具 JavaScript

// Toast提示
function showToast(title, message, type = 'info') {
    const icons = { success: '✅', error: '❌', warning: '⚠️', info: 'ℹ️' };
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.innerHTML = `
        <div class="toast-icon">${icons[type]}</div>
        <div class="toast-content">
            <div class="toast-title">${title}</div>
            <div class="toast-message">${message}</div>
        </div>
    `;
    document.body.appendChild(toast);
    setTimeout(() => {
        toast.classList.add('hide');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// 分析SQL
async function analyzeSQL() {
    const apiUrl = document.getElementById('apiUrl').value.trim();
    const dbName = document.getElementById('dbName').value.trim();
    const query = document.getElementById('query').value.trim();
    const analyzeBtn = document.getElementById('analyzeBtn');
    const btnText = document.getElementById('btnText');

    if (!apiUrl) { showToast('输入错误', '请输入插件API地址', 'warning'); return; }
    if (!query) { showToast('输入错误', '请输入SQL查询语句', 'warning'); return; }

    analyzeBtn.disabled = true;
    btnText.textContent = '分析中...';

    try {
        const url = apiUrl.replace(/\/$/, '') + '/api/sql/analyze';
        const response = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ dbName, query })
        });

        const data = await response.json();
        if (response.ok) {
            showMultiSQLResult(data.data);
            showToast('分析成功', `成功分析 ${data.data.total_count} 条SQL`, 'success');
        } else {
            showToast('分析失败', data.message || '未知错误', 'error');
        }
    } catch (error) {
        showToast('请求失败', error.message, 'error');
    } finally {
        analyzeBtn.disabled = false;
        btnText.textContent = '分析SQL';
    }
}

// 显示多SQL结果
let currentSQLIndex = 0;
function showMultiSQLResult(data) {
    document.getElementById('emptyState').style.display = 'none';
    const validationError = document.getElementById('validationError');
    const tabsContainer = document.getElementById('tabsContainer');
    const sqlTabs = document.getElementById('sqlTabs');
    const resultDiv = document.getElementById('analysisResult');
    
    // 显示验证错误
    if (!data.valid) {
        validationError.innerHTML = `<div class="alert alert-error">❌ ${escapeHTML(data.message)}</div>`;
        validationError.style.display = 'block';
    } else {
        validationError.style.display = 'none';
    }
    
    // 如果只有一条SQL,不显示标签页
    if (data.total_count === 1) {
        tabsContainer.style.display = 'none';
        showResult(data.results[0]);
        return;
    }
    
    // 显示标签页
    tabsContainer.style.display = 'flex';
    sqlTabs.innerHTML = '';
    currentSQLIndex = 0;
    
    // 创建标签
    data.results.forEach((result, index) => {
        const tab = document.createElement('div');
        tab.className = `tab ${index === 0 ? 'active' : ''}`;
        tab.innerHTML = `
            <span class="tab-label">SQL ${index + 1}</span>
            <span class="tab-badge badge-${result.sql_category.toLowerCase()}">${result.sql_category}</span>
        `;
        tab.onclick = () => switchTab(index, data.results);
        sqlTabs.appendChild(tab);
    });
    
    // 显示第一个SQL的结果
    showResult(data.results[0]);
}

// 切换标签
function switchTab(index, results) {
    currentSQLIndex = index;
    
    // 更新标签状态
    const tabs = document.querySelectorAll('.tab');
    tabs.forEach((tab, i) => {
        tab.classList.toggle('active', i === index);
    });
    
    // 显示对应的结果
    showResult(results[index]);
}

// 显示结果
function showResult(data) {
    const resultDiv = document.getElementById('analysisResult');
    
    switch(data.sql_category) {
        case 'DQL': showDQLResult(resultDiv, data); break;
        case 'DML': showDMLResult(resultDiv, data); break;
        case 'DDL': showDDLResult(resultDiv, data); break;
        default: showDefaultResult(resultDiv, data);
    }
    resultDiv.classList.add('show');
}

// HTML转义
function escapeHTML(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

// SQL转义
function escapeSQL(sql) {
    if (!sql) return '';
    return sql.replace(/`/g, '\\`').replace(/\$/g, '\\$');
}

// 复制SQL
function copySQL(sql) {
    navigator.clipboard.writeText(sql).then(() => {
        showToast('复制成功', 'SQL已复制到剪贴板', 'success');
    }).catch(() => {
        showToast('复制失败', '请手动复制', 'error');
    });
}

// 复制到剪贴板
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('复制成功', 'SQL已复制到剪贴板', 'success');
    }).catch(() => {
        showToast('复制失败', '请手动复制', 'error');
    });
}

// SQL高亮
function highlightSQL(sql) {
    if (!sql) return '';
    if (sql.startsWith('--')) return `<span class="sql-comment">${escapeHTML(sql)}</span>`;
    const keywords = ['SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'ORDER BY', 'GROUP BY', 'HAVING', 'LIMIT', 'OFFSET', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER', 'ON', 'AS', 'COUNT', 'SUM', 'AVG', 'MAX', 'MIN', 'DISTINCT', 'WITH', 'SHOW', 'DESCRIBE', 'EXPLAIN', 'USE', 'SET', 'IN', 'NOT', 'NULL', 'IS', 'LIKE', 'BETWEEN', 'INSERT', 'INTO', 'VALUES', 'UPDATE', 'DELETE', 'CREATE', 'ALTER', 'DROP', 'TABLE', 'INDEX', 'ADD', 'MODIFY', 'CHANGE', 'COLUMN'];
    let highlighted = escapeHTML(sql);
    keywords.forEach(keyword => {
        const regex = new RegExp(`\\b${keyword}\\b`, 'gi');
        highlighted = highlighted.replace(regex, `<span class="sql-keyword">${keyword}</span>`);
    });
    return highlighted;
}

// 生成规范化SQL显示组件
function generateNormalizedSQLBox(data) {
    if (!data || !data.normalized_sql) return '';
    var sql = data.normalized_sql;
    var highlighted = highlightSQL(sql);
    return '<div style="margin-top: 12px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; overflow: hidden;">' +
        '<div style="padding: 8px 12px; background: rgba(51, 65, 85, 0.5); border-bottom: 1px solid #334155; display: flex; justify-content: space-between; align-items: center;">' +
            '<span style="font-size: 12px; font-weight: 600; color: #94a3b8;">📝 规范化SQL</span>' +
            '<button onclick="copyToClipboard(this.parentElement.nextElementSibling.textContent)" style="padding: 4px 10px; background: #334155; border: none; border-radius: 4px; color: #94a3b8; font-size: 11px; cursor: pointer;">复制</button>' +
        '</div>' +
        '<div style="padding: 10px 12px; max-height: 120px; overflow-y: auto;">' +
            '<code style="font-family: Consolas, monospace; font-size: 12px; color: #4ade80; word-break: break-all; white-space: pre-wrap;">' + highlighted + '</code>' +
        '</div>' +
    '</div>';
}

// 快捷键
document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('query').addEventListener('keydown', function(e) {
        if (e.ctrlKey && e.key === 'Enter') analyzeSQL();
    });
});
