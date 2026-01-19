// DML结果显示
function showDMLResult(resultDiv, data) {
    const dml = data.dml_info || {};
    const riskClass = dml.risk_level === 'high' ? 'badge-danger' : dml.risk_level === 'medium' ? 'badge-warning' : 'badge-success';
    const isInsert = data.sql_type === 'INSERT';
    
    // 生成通俗易懂的DML解释
    const explanationParts = generateDMLExplanation(data.sql_type, dml);
    
    // INSERT不显示WHERE相关信息
    let statsHtml = '';
    if (isInsert) {
        statsHtml = `
        <div class="stats-grid" style="grid-template-columns: repeat(5, 1fr);">
            <div class="stat-card"><div class="stat-label">SQL类型</div><div class="stat-value"><span class="badge badge-warning">${data.sql_type}</span></div></div>
            <div class="stat-card"><div class="stat-label">SQL分类</div><div class="stat-value"><span class="badge badge-warning">${data.sql_category}</span></div></div>
            <div class="stat-card"><div class="stat-label">目标表</div><div class="stat-value" style="font-size: 12px; color: #60a5fa;">${dml.target_table || '未知'}</div></div>
            <div class="stat-card"><div class="stat-label">数据来源</div><div class="stat-value" style="font-size: 12px;">${dml.data_source || '-'}</div></div>
            <div class="stat-card"><div class="stat-label">风险等级</div><div class="stat-value"><span class="badge ${riskClass}">${dml.risk_level === 'high' ? '高危' : dml.risk_level === 'medium' ? '中等' : '低'}</span></div></div>
        </div>`;
    } else {
        statsHtml = `
        <div class="stats-grid" style="grid-template-columns: repeat(6, 1fr);">
            <div class="stat-card"><div class="stat-label">SQL类型</div><div class="stat-value"><span class="badge badge-warning">${data.sql_type}</span></div></div>
            <div class="stat-card"><div class="stat-label">SQL分类</div><div class="stat-value"><span class="badge badge-warning">${data.sql_category}</span></div></div>
            <div class="stat-card"><div class="stat-label">目标表</div><div class="stat-value" style="font-size: 12px; color: #60a5fa;">${dml.target_table || '未知'}</div></div>
            <div class="stat-card"><div class="stat-label">操作类型</div><div class="stat-value" style="font-size: 12px;">${data.sql_type === 'UPDATE' ? '更新数据' : '删除数据'}</div></div>
            <div class="stat-card"><div class="stat-label">有WHERE</div><div class="stat-value"><span class="badge ${dml.has_where ? 'badge-success' : 'badge-danger'}">${dml.has_where ? '是' : '否'}</span></div></div>
            <div class="stat-card"><div class="stat-label">风险等级</div><div class="stat-value"><span class="badge ${riskClass}">${dml.risk_level === 'high' ? '高危' : dml.risk_level === 'medium' ? '中等' : '低'}</span></div></div>
        </div>`;
    }
    
    // 字段信息
    let fieldsHtml = '';
    if (dml.affected_cols && dml.affected_cols.length > 0) {
        fieldsHtml = `
        <div class="stats-grid" style="grid-template-columns: 1fr; margin-top: 12px;">
            <div class="stat-card compact">
                <div class="stat-label">涉及字段 (${dml.affected_cols.length}个)</div>
                <div class="stat-value" style="font-size: 12px; color: #94a3b8; display: flex; flex-wrap: wrap; gap: 6px;">
                    ${dml.affected_cols.map(function(col) { return '<span class="badge badge-info">' + escapeHTML(col) + '</span>'; }).join('')}
                </div>
            </div>
        </div>`;
    }
    
    // WHERE条件（仅UPDATE/DELETE显示）
    let whereHtml = '';
    if (!isInsert && dml.where_preview) {
        whereHtml = `
        <div class="stats-grid" style="grid-template-columns: 1fr; margin-top: 12px;">
            <div class="stat-card compact">
                <div class="stat-label">WHERE条件</div>
                <div class="stat-value" style="font-size: 12px; color: #fbbf24; font-family: monospace;">${escapeHTML(dml.where_preview)}</div>
            </div>
        </div>`;
    }
    
    // INSERT VALUES 表格
    let valuesTableHtml = '';
    if (isInsert && dml.insert_values && dml.insert_values.length > 0) {
        const cols = dml.affected_cols || [];
        valuesTableHtml = `
        <div style="margin-top: 16px; background: #1e293b; border: 1px solid #334155; border-radius: 8px; overflow: hidden; flex: 1; display: flex; flex-direction: column; min-height: 200px;">
            <div style="padding: 12px 16px; border-bottom: 1px solid #334155; display: flex; justify-content: space-between; align-items: center; flex-shrink: 0;">
                <span style="font-size: 14px; font-weight: 600; color: #f1f5f9;">📊 插入数据预览</span>
                <span style="font-size: 12px; color: #94a3b8;">${dml.insert_values.length} 行数据</span>
            </div>
            <div class="table-scroll" style="flex: 1; overflow-y: auto; overflow-x: auto;">
                <table style="width: 100%; font-size: 12px; border-collapse: collapse;">
                    <thead style="position: sticky; top: 0; z-index: 1;">
                        <tr style="background: #334155;">
                            <th style="padding: 10px 12px; text-align: left; color: #f1f5f9; font-weight: 500; background: #334155;">#</th>
                            ${cols.map(col => `<th style="padding: 10px 12px; text-align: left; color: #f1f5f9; font-weight: 500; background: #334155;">${escapeHTML(col)}</th>`).join('')}
                        </tr>
                    </thead>
                    <tbody>
                        ${dml.insert_values.map((row, idx) => `
                            <tr style="border-top: 1px solid #334155;">
                                <td style="padding: 8px 12px; color: #64748b;">${idx + 1}</td>
                                ${row.map(val => `<td style="padding: 8px 12px; color: #4ade80; font-family: monospace;">${escapeHTML(val)}</td>`).join('')}
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        </div>`;
    }
    
    // 规范化SQL显示
    const normalizedSqlHtml = generateNormalizedSQLBox(data);
    
    resultDiv.innerHTML = `
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div style="flex-shrink: 0;">${statsHtml}</div>
            <div style="flex-shrink: 0;">
                <div class="explanation-box">
                    <div class="explanation-title">📖 SQL解释</div>
                    <div class="explanation-content">
                        ${explanationParts.map(function(p, i) { return '<div>' + (i + 1) + '. ' + p + '</div>'; }).join('')}
                    </div>
                </div>
            </div>
            <div style="flex-shrink: 0;">${normalizedSqlHtml}</div>
            <div style="flex-shrink: 0;">${fieldsHtml}</div>
            <div style="flex-shrink: 0;">${whereHtml}</div>
            <div style="flex: 1; display: flex; flex-direction: column; min-height: 0;">${valuesTableHtml}</div>
        </div>
    `;
}

// 生成DML解释
function generateDMLExplanation(sqlType, dml) {
    var parts = [];
    var tableName = dml.target_table || '表';
    var cols = dml.affected_cols || [];
    
    if (sqlType === 'INSERT') {
        parts.push('向表 <b style="color:#60a5fa;">' + tableName + '</b> 插入数据');
        if (cols.length > 0) {
            var colNames = cols.map(function(c) { return '<b style="color:#4ade80;">' + c + '</b>'; }).join('、');
            parts.push('插入字段: ' + colNames);
        }
        if (dml.data_source === 'VALUES') {
            parts.push('数据来源: <span style="color:#4ade80;">直接指定值 (VALUES)</span>');
        } else if (dml.data_source === 'SELECT') {
            parts.push('数据来源: <span style="color:#fbbf24;">从其他表查询 (SELECT)</span>');
        }
        if (dml.estimate_rows) {
            parts.push('预计插入: <b style="color:#fbbf24;">' + dml.estimate_rows + '</b>');
        }
    } else if (sqlType === 'UPDATE') {
        parts.push('更新表 <b style="color:#60a5fa;">' + tableName + '</b> 的数据');
        if (cols.length > 0) {
            var colNames = cols.map(function(c) { return '<b style="color:#fbbf24;">' + c + '</b>'; }).join('、');
            parts.push('修改字段: ' + colNames);
        }
        if (dml.has_where) {
            parts.push('<span style="color:#4ade80;">✓ 有WHERE条件限制更新范围</span>');
            if (dml.where_preview) {
                parts.push('条件: <code style="color:#fbbf24;">' + escapeHTML(dml.where_preview) + '</code>');
            }
        } else {
            parts.push('<span style="color:#f87171;">⚠️ 无WHERE条件，将更新全表所有数据！</span>');
        }
        if (dml.estimate_rows) {
            parts.push('预计影响: <b>' + dml.estimate_rows + '</b>');
        }
    } else if (sqlType === 'DELETE') {
        parts.push('从表 <b style="color:#60a5fa;">' + tableName + '</b> 删除数据');
        if (dml.has_where) {
            parts.push('<span style="color:#4ade80;">✓ 有WHERE条件限制删除范围</span>');
            if (dml.where_preview) {
                parts.push('条件: <code style="color:#fbbf24;">' + escapeHTML(dml.where_preview) + '</code>');
            }
        } else {
            parts.push('<span style="color:#f87171;">⚠️ 无WHERE条件，将删除全表所有数据！</span>');
        }
        if (dml.estimate_rows) {
            parts.push('预计删除: <b>' + dml.estimate_rows + '</b>');
        }
    }
    
    // 风险提示
    if (dml.risk_level === 'high') {
        parts.push('<span style="color:#f87171; font-weight: bold;">🚨 ' + (dml.risk_reason || '高风险操作，请谨慎执行！') + '</span>');
    } else if (dml.risk_level === 'medium') {
        parts.push('<span style="color:#fbbf24;">⚠️ ' + (dml.risk_reason || '中等风险，请确认操作正确') + '</span>');
    }
    
    return parts;
}
