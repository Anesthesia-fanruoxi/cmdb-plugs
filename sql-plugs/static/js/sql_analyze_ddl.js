// DDL结果显示
function showDDLResult(resultDiv, data) {
    const ddl = data.ddl_info || {};
    const details = ddl.details || {};
    const riskClass = ddl.risk_level === 'high' ? 'badge-danger' : ddl.risk_level === 'medium' ? 'badge-warning' : 'badge-success';
    const columns = details.columns || [];
    
    // 统计信息
    const colsWithComment = columns.filter(c => c.comment).length;
    const colsWithDefault = columns.filter(c => c.default).length;
    const colsNotNull = columns.filter(c => c.nullable === 'NOT NULL').length;
    const hasTableComment = !!details.table_comment;
    const hasPrimaryKey = !!details.primary_key;
    const indexes = details.indexes || [];
    const hasIndex = details.has_index || indexes.length > 0;
    
    // 生成通俗易懂的SQL解释
    const explanationParts = generateDDLExplanation(ddl, details);
    const explanationHtml = explanationParts.length > 0 ? `
        <div class="explanation-box">
            <div class="explanation-title">📖 SQL解释</div>
            <div class="explanation-content">
                ${explanationParts.map((p, i) => `<div>${i + 1}. ${p}</div>`).join('')}
            </div>
        </div>` : '';
    
    // 索引表格HTML - 始终显示，即使没有索引
    let indexTableHtml = '';
    if (ddl.operation === 'CREATE' && ddl.object_type === 'TABLE') {
        if (indexes.length > 0) {
            indexTableHtml = `
            <div style="margin-top: 12px; background: #1e293b; border: 1px solid #334155; border-radius: 8px; overflow: hidden;">
                <div style="padding: 10px 16px; border-bottom: 1px solid #334155; display: flex; justify-content: space-between; align-items: center;">
                    <span style="font-size: 13px; font-weight: 600; color: #f1f5f9;">📇 索引信息</span>
                    <span style="font-size: 12px; color: #94a3b8;">${indexes.length} 个索引</span>
                </div>
                <div style="max-height: 150px; overflow-y: auto;">
                    <table style="width: 100%; font-size: 12px; border-collapse: collapse;">
                        <thead>
                            <tr style="background: #334155;">
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">#</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">索引名</th>
                                <th style="padding: 8px 12px; text-align: center; color: #f1f5f9; font-weight: 500;">类型</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">包含字段</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${indexes.map((idx, i) => `
                                <tr style="border-top: 1px solid #334155;">
                                    <td style="padding: 6px 12px; color: #64748b;">${i + 1}</td>
                                    <td style="padding: 6px 12px; color: #60a5fa; font-family: monospace;">${escapeHTML(idx.name || '-')}</td>
                                    <td style="padding: 6px 12px; text-align: center;"><span class="badge ${idx.type === 'PRIMARY' ? 'badge-danger' : idx.type === 'UNIQUE' ? 'badge-warning' : 'badge-info'}">${idx.type || 'INDEX'}</span></td>
                                    <td style="padding: 6px 12px; color: #4ade80; font-family: monospace;">${escapeHTML(idx.column_str || (idx.columns || []).join(', '))}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                </div>
            </div>`;
        } else {
            // 没有索引时显示提示
            indexTableHtml = `
            <div style="margin-top: 12px; background: #1e293b; border: 1px solid #fbbf24; border-radius: 8px; padding: 12px 16px;">
                <div style="display: flex; align-items: center; gap: 8px;">
                    <span style="font-size: 13px; font-weight: 600; color: #fbbf24;">📇 索引信息</span>
                    <span style="font-size: 12px; color: #fbbf24;">⚠️ 该表未定义任何索引（建议添加适当的索引以提高查询性能）</span>
                </div>
            </div>`;
        }
    }
    
    // 表级别卡片HTML
    let tableCardsHtml = '';
    if (ddl.operation === 'CREATE' && ddl.object_type === 'TABLE') {
        tableCardsHtml = `
        <div class="stats-grid" style="grid-template-columns: repeat(9, 1fr); margin-top: 12px;">
            <div class="stat-card compact"><div class="stat-label">表名</div><div class="stat-value" style="color: #60a5fa;">${ddl.object_name || '-'}</div></div>
            <div class="stat-card compact"><div class="stat-label">字段数</div><div class="stat-value">${columns.length}</div></div>
            <div class="stat-card compact"><div class="stat-label">表注释</div><div class="stat-value"><span class="badge ${hasTableComment ? 'badge-success' : 'badge-danger'}">${hasTableComment ? '有' : '无'}</span></div></div>
            <div class="stat-card compact"><div class="stat-label">主键</div><div class="stat-value"><span class="badge ${hasPrimaryKey ? 'badge-success' : 'badge-warning'}">${hasPrimaryKey ? '有' : '无'}</span></div></div>
            <div class="stat-card compact"><div class="stat-label">索引</div><div class="stat-value"><span class="badge ${hasIndex ? 'badge-success' : 'badge-warning'}">${hasIndex ? indexes.length + '个' : '无'}</span></div></div>
            <div class="stat-card compact"><div class="stat-label">引擎</div><div class="stat-value" style="font-size: 11px;">${details.engine || '-'}</div></div>
            <div class="stat-card compact"><div class="stat-label">字符集</div><div class="stat-value" style="font-size: 11px;">${details.charset || '-'}</div></div>
            <div class="stat-card compact"><div class="stat-label">排序规则</div><div class="stat-value" style="font-size: 10px;">${details.collation || '-'}</div></div>
            <div class="stat-card compact"><div class="stat-label">风险等级</div><div class="stat-value"><span class="badge ${riskClass}">${ddl.risk_level === 'high' ? '高' : ddl.risk_level === 'medium' ? '中' : '低'}</span></div></div>
        </div>
        ${hasTableComment ? `<div class="stats-grid" style="grid-template-columns: 1fr; margin-top: 8px;"><div class="stat-card compact"><div class="stat-label">表注释内容</div><div class="stat-value" style="color: #4ade80;">"${escapeHTML(details.table_comment)}"</div></div></div>` : ''}
        <div class="stats-grid" style="grid-template-columns: repeat(3, 1fr); margin-top: 8px;">
            <div class="stat-card compact"><div class="stat-label">有注释的字段</div><div class="stat-value">${colsWithComment} / ${columns.length} <span style="color: ${colsWithComment === columns.length ? '#4ade80' : '#f87171'};">(${columns.length > 0 ? Math.round(colsWithComment/columns.length*100) : 0}%)</span></div></div>
            <div class="stat-card compact"><div class="stat-label">有默认值的字段</div><div class="stat-value">${colsWithDefault} / ${columns.length}</div></div>
            <div class="stat-card compact"><div class="stat-label">NOT NULL字段</div><div class="stat-value">${colsNotNull} / ${columns.length}</div></div>
        </div>
        ${indexTableHtml}`;
    } else if (ddl.operation === 'ALTER' && ddl.object_type === 'TABLE') {
        const addCols = details.add_columns || [];
        const modCols = details.modify_columns || [];
        const dropCols = details.drop_columns || [];
        const addIdxs = details.add_indexes || [];
        const dropIdxs = details.drop_indexes || [];
        
        tableCardsHtml = `
        <div class="stats-grid" style="grid-template-columns: repeat(7, 1fr); margin-top: 12px;">
            <div class="stat-card compact"><div class="stat-label">目标表</div><div class="stat-value" style="color: #60a5fa;">${ddl.object_name || '-'}</div></div>
            <div class="stat-card compact"><div class="stat-label">新增字段</div><div class="stat-value"><span class="badge ${addCols.length > 0 ? 'badge-success' : 'badge-secondary'}">${addCols.length}</span></div></div>
            <div class="stat-card compact"><div class="stat-label">修改字段</div><div class="stat-value"><span class="badge ${modCols.length > 0 ? 'badge-warning' : 'badge-secondary'}">${modCols.length}</span></div></div>
            <div class="stat-card compact"><div class="stat-label">删除字段</div><div class="stat-value"><span class="badge ${dropCols.length > 0 ? 'badge-danger' : 'badge-secondary'}">${dropCols.length}</span></div></div>
            <div class="stat-card compact"><div class="stat-label">新增索引</div><div class="stat-value"><span class="badge ${addIdxs.length > 0 ? 'badge-info' : 'badge-secondary'}">${addIdxs.length}</span></div></div>
            <div class="stat-card compact"><div class="stat-label">删除索引</div><div class="stat-value"><span class="badge ${dropIdxs.length > 0 ? 'badge-danger' : 'badge-secondary'}">${dropIdxs.length}</span></div></div>
            <div class="stat-card compact"><div class="stat-label">风险等级</div><div class="stat-value"><span class="badge ${riskClass}">${ddl.risk_level === 'high' ? '高危' : ddl.risk_level === 'medium' ? '中等' : '低'}</span></div></div>
        </div>`;
        
        // 新增字段详情表格
        if (addCols.length > 0) {
            const addColsWithComment = addCols.filter(c => c.comment).length;
            const addColsWithDefault = addCols.filter(c => c.default).length;
            tableCardsHtml += `
            <div style="margin-top: 12px; background: #1e293b; border: 1px solid #334155; border-radius: 8px; overflow: hidden;">
                <div style="padding: 10px 16px; border-bottom: 1px solid #334155; display: flex; justify-content: space-between; align-items: center;">
                    <span style="font-size: 13px; font-weight: 600; color: #4ade80;">➕ 新增字段详情</span>
                    <span style="font-size: 12px; color: #94a3b8;">
                        有注释: <span class="badge ${addColsWithComment === addCols.length ? 'badge-success' : 'badge-warning'}">${addColsWithComment}/${addCols.length}</span>
                        有默认值: <span class="badge badge-info">${addColsWithDefault}/${addCols.length}</span>
                    </span>
                </div>
                <div style="max-height: 200px; overflow-y: auto;">
                    <table style="width: 100%; font-size: 12px; border-collapse: collapse;">
                        <thead>
                            <tr style="background: #334155;">
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">#</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">字段名</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">类型</th>
                                <th style="padding: 8px 12px; text-align: center; color: #f1f5f9; font-weight: 500;">可空</th>
                                <th style="padding: 8px 12px; text-align: center; color: #f1f5f9; font-weight: 500;">默认值</th>
                                <th style="padding: 8px 12px; text-align: center; color: #f1f5f9; font-weight: 500;">有注释</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">注释内容</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">其他属性</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${addCols.map((col, idx) => `
                                <tr style="border-top: 1px solid #334155;">
                                    <td style="padding: 6px 12px; color: #64748b;">${idx + 1}</td>
                                    <td style="padding: 6px 12px; color: #60a5fa; font-family: monospace;">${escapeHTML(col.name || '')}</td>
                                    <td style="padding: 6px 12px; color: #fbbf24; font-family: monospace;">${escapeHTML(col.type || '')}</td>
                                    <td style="padding: 6px 12px; text-align: center;"><span class="badge ${col.nullable === 'NOT NULL' ? 'badge-danger' : 'badge-secondary'}">${col.nullable === 'NOT NULL' ? 'NOT NULL' : 'NULL'}</span></td>
                                    <td style="padding: 6px 12px; text-align: center;"><span class="badge ${col.default ? 'badge-success' : 'badge-secondary'}">${col.default ? escapeHTML(col.default) : '-'}</span></td>
                                    <td style="padding: 6px 12px; text-align: center;"><span class="badge ${col.comment ? 'badge-success' : 'badge-danger'}">${col.comment ? '✓' : '✗'}</span></td>
                                    <td style="padding: 6px 12px; color: #4ade80; max-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${escapeHTML(col.comment || '')}">${escapeHTML(col.comment || '-')}</td>
                                    <td style="padding: 6px 12px; color: #c084fc; font-size: 11px;">${escapeHTML(col.extra || '-')}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                </div>
            </div>`;
        }
        
        // 修改字段详情表格
        if (modCols.length > 0) {
            tableCardsHtml += `
            <div style="margin-top: 12px; background: #1e293b; border: 1px solid #334155; border-radius: 8px; overflow: hidden;">
                <div style="padding: 10px 16px; border-bottom: 1px solid #334155; display: flex; justify-content: space-between; align-items: center;">
                    <span style="font-size: 13px; font-weight: 600; color: #fbbf24;">✏️ 修改字段详情</span>
                    <span style="font-size: 12px; color: #94a3b8;">${modCols.length} 个字段</span>
                </div>
                <div style="max-height: 200px; overflow-y: auto;">
                    <table style="width: 100%; font-size: 12px; border-collapse: collapse;">
                        <thead>
                            <tr style="background: #334155;">
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">#</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">字段名</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">新类型</th>
                                <th style="padding: 8px 12px; text-align: center; color: #f1f5f9; font-weight: 500;">可空</th>
                                <th style="padding: 8px 12px; text-align: center; color: #f1f5f9; font-weight: 500;">默认值</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">注释</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">其他属性</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${modCols.map((col, idx) => `
                                <tr style="border-top: 1px solid #334155;">
                                    <td style="padding: 6px 12px; color: #64748b;">${idx + 1}</td>
                                    <td style="padding: 6px 12px; color: #60a5fa; font-family: monospace;">${escapeHTML(col.name || '')}</td>
                                    <td style="padding: 6px 12px; color: #fbbf24; font-family: monospace;">${escapeHTML(col.type || '')}</td>
                                    <td style="padding: 6px 12px; text-align: center;"><span class="badge ${col.nullable === 'NOT NULL' ? 'badge-danger' : 'badge-secondary'}">${col.nullable || '-'}</span></td>
                                    <td style="padding: 6px 12px; text-align: center;"><span class="badge ${col.default ? 'badge-success' : 'badge-secondary'}">${col.default ? escapeHTML(col.default) : '-'}</span></td>
                                    <td style="padding: 6px 12px; color: #4ade80;">${escapeHTML(col.comment || '-')}</td>
                                    <td style="padding: 6px 12px; color: #c084fc; font-size: 11px;">${escapeHTML(col.extra || '-')}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                </div>
            </div>`;
        }
        
        // 新增索引详情
        if (addIdxs.length > 0) {
            tableCardsHtml += `
            <div style="margin-top: 12px; background: #1e293b; border: 1px solid #334155; border-radius: 8px; overflow: hidden;">
                <div style="padding: 10px 16px; border-bottom: 1px solid #334155; display: flex; justify-content: space-between; align-items: center;">
                    <span style="font-size: 13px; font-weight: 600; color: #60a5fa;">📇 新增索引详情</span>
                    <span style="font-size: 12px; color: #94a3b8;">${addIdxs.length} 个索引</span>
                </div>
                <div style="max-height: 150px; overflow-y: auto;">
                    <table style="width: 100%; font-size: 12px; border-collapse: collapse;">
                        <thead>
                            <tr style="background: #334155;">
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">#</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">索引名</th>
                                <th style="padding: 8px 12px; text-align: center; color: #f1f5f9; font-weight: 500;">类型</th>
                                <th style="padding: 8px 12px; text-align: left; color: #f1f5f9; font-weight: 500;">包含字段</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${addIdxs.map((idx, i) => `
                                <tr style="border-top: 1px solid #334155;">
                                    <td style="padding: 6px 12px; color: #64748b;">${i + 1}</td>
                                    <td style="padding: 6px 12px; color: #60a5fa; font-family: monospace;">${escapeHTML(idx.name || '-')}</td>
                                    <td style="padding: 6px 12px; text-align: center;"><span class="badge ${idx.type === 'UNIQUE' ? 'badge-warning' : 'badge-info'}">${idx.type || 'INDEX'}</span></td>
                                    <td style="padding: 6px 12px; color: #4ade80; font-family: monospace;">${escapeHTML(idx.column_str || (idx.columns || []).join(', '))}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                </div>
            </div>`;
        }
        
        // 删除字段列表
        if (dropCols.length > 0) {
            tableCardsHtml += `
            <div style="margin-top: 12px; background: #1e293b; border: 1px solid #f87171; border-radius: 8px; padding: 12px 16px;">
                <div style="font-size: 13px; font-weight: 600; color: #f87171; margin-bottom: 8px;">❌ 删除字段</div>
                <div style="display: flex; flex-wrap: wrap; gap: 8px;">
                    ${dropCols.map(col => `<span class="badge badge-danger" style="font-size: 12px;">${escapeHTML(col)}</span>`).join('')}
                </div>
                <div style="margin-top: 8px; font-size: 11px; color: #f87171;">⚠️ 删除字段将导致数据永久丢失！</div>
            </div>`;
        }
        
        // 删除索引列表
        if (dropIdxs.length > 0) {
            tableCardsHtml += `
            <div style="margin-top: 12px; background: #1e293b; border: 1px solid #fbbf24; border-radius: 8px; padding: 12px 16px;">
                <div style="font-size: 13px; font-weight: 600; color: #fbbf24; margin-bottom: 8px;">🗑️ 删除索引</div>
                <div style="display: flex; flex-wrap: wrap; gap: 8px;">
                    ${dropIdxs.map(idx => `<span class="badge badge-warning" style="font-size: 12px;">${escapeHTML(idx)}</span>`).join('')}
                </div>
            </div>`;
        }
    }
    
    // 字段表格HTML
    let columnsTableHtml = '';
    if (columns.length > 0) {
        columnsTableHtml = `
        <div style="margin-top: 16px; background: #1e293b; border: 1px solid #334155; border-radius: 8px; overflow: hidden; flex: 1; display: flex; flex-direction: column; min-height: 200px;">
            <div style="padding: 12px 16px; border-bottom: 1px solid #334155; display: flex; justify-content: space-between; align-items: center; flex-shrink: 0;">
                <span style="font-size: 14px; font-weight: 600; color: #f1f5f9;">📊 字段详情</span>
                <span style="font-size: 12px; color: #94a3b8;">${columns.length} 个字段</span>
            </div>
            <div class="table-scroll" style="flex: 1; overflow-y: auto; overflow-x: auto;">
                <table style="width: 100%; font-size: 12px; border-collapse: collapse;">
                    <thead style="position: sticky; top: 0; z-index: 1;">
                        <tr style="background: #334155;">
                            <th style="padding: 10px 12px; text-align: left; color: #f1f5f9; font-weight: 500; background: #334155;">#</th>
                            <th style="padding: 10px 12px; text-align: left; color: #f1f5f9; font-weight: 500; background: #334155;">字段名</th>
                            <th style="padding: 10px 12px; text-align: left; color: #f1f5f9; font-weight: 500; background: #334155;">类型</th>
                            <th style="padding: 10px 12px; text-align: center; color: #f1f5f9; font-weight: 500; background: #334155;">可空</th>
                            <th style="padding: 10px 12px; text-align: center; color: #f1f5f9; font-weight: 500; background: #334155;">默认值</th>
                            <th style="padding: 10px 12px; text-align: center; color: #f1f5f9; font-weight: 500; background: #334155;">有注释</th>
                            <th style="padding: 10px 12px; text-align: left; color: #f1f5f9; font-weight: 500; background: #334155;">注释内容</th>
                            <th style="padding: 10px 12px; text-align: left; color: #f1f5f9; font-weight: 500; background: #334155;">其他属性</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${columns.map((col, idx) => `
                            <tr style="border-top: 1px solid #334155;">
                                <td style="padding: 8px 12px; color: #64748b;">${idx + 1}</td>
                                <td style="padding: 8px 12px; color: #60a5fa; font-family: monospace;">${escapeHTML(col.name || '')}</td>
                                <td style="padding: 8px 12px; color: #fbbf24; font-family: monospace;">${escapeHTML(col.type || '')}</td>
                                <td style="padding: 8px 12px; text-align: center;"><span class="badge ${col.nullable === 'NOT NULL' ? 'badge-danger' : 'badge-secondary'}">${col.nullable === 'NOT NULL' ? 'NOT NULL' : 'NULL'}</span></td>
                                <td style="padding: 8px 12px; text-align: center;"><span class="badge ${col.default ? 'badge-success' : 'badge-secondary'}">${col.default ? escapeHTML(col.default) : '-'}</span></td>
                                <td style="padding: 8px 12px; text-align: center;"><span class="badge ${col.comment ? 'badge-success' : 'badge-danger'}">${col.comment ? '✓' : '✗'}</span></td>
                                <td style="padding: 8px 12px; color: #4ade80; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${escapeHTML(col.comment || '')}">${escapeHTML(col.comment || '-')}</td>
                                <td style="padding: 8px 12px; color: #c084fc; font-size: 11px;">${escapeHTML(col.extra || '-')}</td>
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
            <div class="stats-grid" style="grid-template-columns: repeat(5, 1fr); flex-shrink: 0;">
                <div class="stat-card"><div class="stat-label">SQL类型</div><div class="stat-value"><span class="badge badge-danger">${data.sql_type}</span></div></div>
                <div class="stat-card"><div class="stat-label">SQL分类</div><div class="stat-value"><span class="badge badge-danger">${data.sql_category}</span></div></div>
                <div class="stat-card"><div class="stat-label">操作类型</div><div class="stat-value" style="font-size: 12px;">${ddl.operation || '-'}</div></div>
                <div class="stat-card"><div class="stat-label">对象类型</div><div class="stat-value" style="font-size: 12px;">${ddl.object_type || '-'}</div></div>
                <div class="stat-card"><div class="stat-label">风险等级</div><div class="stat-value"><span class="badge ${riskClass}">${ddl.risk_level === 'high' ? '高危' : ddl.risk_level === 'medium' ? '中等' : '低'}</span></div></div>
            </div>
            <div style="flex-shrink: 0;">${explanationHtml}</div>
            <div style="flex-shrink: 0;">${normalizedSqlHtml}</div>
            <div style="flex-shrink: 0;">${tableCardsHtml}</div>
            <div style="flex: 1; display: flex; flex-direction: column; min-height: 0;">${columnsTableHtml}</div>
        </div>
    `;
}

// 生成DDL解释
function generateDDLExplanation(ddl, details) {
    const parts = [];
    const tableName = ddl.object_name || '表';
    
    function describeColChange(col) {
        const changes = [];
        if (col.type) changes.push(`类型为 <span style="color:#fbbf24;">${col.type}</span>`);
        if (col.nullable === 'NOT NULL') changes.push(`<span style="color:#f87171;">不允许为空</span>`);
        if (col.nullable === 'NULL') changes.push(`允许为空`);
        if (col.default) changes.push(`默认值为 <span style="color:#4ade80;">${col.default}</span>`);
        if (col.comment) changes.push(`注释为 <span style="color:#4ade80;">"${col.comment}"</span>`);
        if (col.extra) changes.push(`${col.extra}`);
        return changes.length > 0 ? changes.join('，') : '修改定义';
    }
    
    if (ddl.operation === 'CREATE') {
        if (ddl.object_type === 'TABLE') {
            const colCount = (details.columns || []).length;
            const idxs = details.indexes || [];
            parts.push(`创建一个名为 <b style="color:#60a5fa;">${tableName}</b> 的新表`);
            if (colCount > 0) parts.push(`包含 <b style="color:#fbbf24;">${colCount}</b> 个字段`);
            if (details.primary_key) parts.push(`主键为 <b style="color:#c084fc;">${details.primary_key}</b>`);
            // 索引信息
            if (idxs.length > 0) {
                const uniqueIdxs = idxs.filter(i => i.type === 'UNIQUE').length;
                const normalIdxs = idxs.filter(i => i.type === 'INDEX').length;
                let idxDesc = `包含 <b style="color:#60a5fa;">${idxs.length}</b> 个索引`;
                if (uniqueIdxs > 0 || normalIdxs > 0) {
                    const idxParts = [];
                    if (details.primary_key) idxParts.push('1个主键');
                    if (uniqueIdxs > 0) idxParts.push(`${uniqueIdxs}个唯一索引`);
                    if (normalIdxs > 0) idxParts.push(`${normalIdxs}个普通索引`);
                    idxDesc += ` (${idxParts.join('、')})`;
                }
                parts.push(idxDesc);
            } else {
                parts.push(`<span style="color:#fbbf24;">⚠️ 未定义索引（除主键外）</span>`);
            }
            if (details.engine) parts.push(`使用 <b>${details.engine}</b> 引擎`);
            if (details.charset) parts.push(`字符集为 <b>${details.charset}</b>`);
            if (details.table_comment) parts.push(`表注释: <span style="color:#4ade80;">"${details.table_comment}"</span>`);
        } else if (ddl.object_type === 'INDEX') {
            parts.push(`在表上创建索引 <b style="color:#60a5fa;">${tableName}</b>`);
        } else if (ddl.object_type === 'DATABASE') {
            parts.push(`创建数据库 <b style="color:#60a5fa;">${tableName}</b>`);
        }
    } else if (ddl.operation === 'ALTER') {
        parts.push(`修改表 <b style="color:#60a5fa;">${tableName}</b> 的结构：`);
        
        const addCols = details.add_columns || [];
        const modCols = details.modify_columns || [];
        const dropCols = details.drop_columns || [];
        const addIdxs = details.add_indexes || [];
        const dropIdxs = details.drop_indexes || [];
        
        addCols.forEach(col => {
            parts.push(`<span style="color:#4ade80;">➕ 新增字段</span> <b style="color:#60a5fa;">${col.name}</b>：${describeColChange(col)}`);
        });
        modCols.forEach(col => {
            parts.push(`<span style="color:#fbbf24;">✏️ 修改字段</span> <b style="color:#60a5fa;">${col.name}</b>：${describeColChange(col)}`);
        });
        dropCols.forEach(col => {
            parts.push(`<span style="color:#f87171;">❌ 删除字段</span> <b style="color:#f87171;">${col}</b> <span style="color:#f87171;">⚠️ 数据将丢失</span>`);
        });
        addIdxs.forEach(idx => {
            parts.push(`<span style="color:#60a5fa;">📇 新增索引</span> ${idx}`);
        });
        dropIdxs.forEach(idx => {
            parts.push(`<span style="color:#f87171;">🗑️ 删除索引</span> ${idx}`);
        });
        
        if (details.rename_info) parts.push(`<span style="color:#c084fc;">🔄 重命名表</span>：${details.rename_info}`);
        if (details.change_comment) parts.push(`<span style="color:#4ade80;">💬 修改表注释</span>：${details.change_comment}`);
    } else if (ddl.operation === 'DROP') {
        parts.push(`删除${ddl.object_type === 'TABLE' ? '表' : ddl.object_type === 'INDEX' ? '索引' : ddl.object_type === 'DATABASE' ? '数据库' : '对象'} <b style="color:#f87171;">${tableName}</b>`);
        parts.push(`<span style="color:#f87171;">⚠️ 此操作不可恢复，所有数据将永久删除！</span>`);
    } else if (ddl.operation === 'TRUNCATE') {
        parts.push(`清空表 <b style="color:#f87171;">${tableName}</b> 的所有数据`);
        parts.push(`<span style="color:#f87171;">⚠️ 表结构保留，但数据将全部删除！</span>`);
    } else if (ddl.operation === 'RENAME') {
        parts.push(`重命名表 <b style="color:#60a5fa;">${tableName}</b>`);
    }
    
    return parts;
}
