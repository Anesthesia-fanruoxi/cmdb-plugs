// DQL结果显示
function showDQLResult(resultDiv, data) {
    const features = data.features || {};
    const structure = data.structure || {};
    const indexSuggestions = data.index_suggestions || [];
    
    // 特性标签
    const featureBadges = [];
    if (features.has_where) featureBadges.push('<span class="badge badge-info">WHERE</span>');
    if (features.has_join) featureBadges.push(`<span class="badge badge-info">JOIN×${features.join_count || 1}</span>`);
    if (features.has_group_by) featureBadges.push('<span class="badge badge-info">GROUP BY</span>');
    if (features.has_having) featureBadges.push('<span class="badge badge-info">HAVING</span>');
    if (features.has_order_by) featureBadges.push('<span class="badge badge-info">ORDER BY</span>');
    if (features.has_distinct) featureBadges.push('<span class="badge badge-warning">DISTINCT</span>');
    if (features.has_subquery) featureBadges.push('<span class="badge badge-warning">子查询</span>');
    if (features.has_union) featureBadges.push('<span class="badge badge-warning">UNION</span>');
    if (features.has_aggregate) featureBadges.push('<span class="badge badge-success">聚合函数</span>');
    if (features.has_cte) featureBadges.push('<span class="badge badge-success">CTE</span>');
    if ((structure.window_functions || []).length > 0) featureBadges.push('<span class="badge badge-success">窗口函数</span>');
    
    const structureHtml = generateStructureHtml(data, structure, indexSuggestions);
    const normalizedSqlHtml = generateNormalizedSQLBox(data);
    
    resultDiv.innerHTML = `
        <div style="display: flex; flex-direction: column; height: 100%; min-height: 0;">
            <div class="stats-grid" style="grid-template-columns: repeat(6, 1fr); flex-shrink: 0;">
                <div class="stat-card"><div class="stat-label">SQL类型</div><div class="stat-value"><span class="badge badge-info">${data.sql_type}</span></div></div>
                <div class="stat-card"><div class="stat-label">SQL分类</div><div class="stat-value"><span class="badge badge-info">${data.sql_category}</span></div></div>
                <div class="stat-card"><div class="stat-label">涉及表</div><div class="stat-value">${data.tables ? data.tables.length : 0}个</div></div>
                <div class="stat-card"><div class="stat-label">查询字段</div><div class="stat-value">${data.columns ? data.columns.length : 0}个</div></div>
                <div class="stat-card"><div class="stat-label">有过滤条件</div><div class="stat-value"><span class="badge ${data.has_filter ? 'badge-success' : 'badge-warning'}">${data.has_filter ? '是' : '否'}</span></div></div>
                <div class="stat-card"><div class="stat-label">用户LIMIT</div><div class="stat-value">${data.user_limit > 0 ? data.user_limit : '无'}</div></div>
            </div>
            <div class="stats-grid" style="grid-template-columns: 1fr; margin-top: 12px; flex-shrink: 0;">
                <div class="stat-card compact">
                    <div class="stat-label">SQL特性</div>
                    <div class="stat-value" style="font-size: 12px; display: flex; gap: 6px; flex-wrap: wrap;">
                        ${featureBadges.length > 0 ? featureBadges.join('') : '<span style="color: #64748b;">简单查询，无特殊特性</span>'}
                    </div>
                </div>
            </div>
            <div style="flex-shrink: 0;">${normalizedSqlHtml}</div>
            ${structureHtml}
        </div>
    `;
}

// 生成SQL结构HTML
function generateStructureHtml(data, structure, indexSuggestions) {
    if (!structure) return '<div style="color: #f87171;">结构数据为空</div>';
    
    const hasContent = structure.select_clause || structure.from_clause || structure.where_clause || 
                       structure.group_by_clause || structure.having_clause || structure.order_by_clause || 
                       structure.limit_clause || (structure.subqueries && structure.subqueries.length > 0) ||
                       (structure.window_functions && structure.window_functions.length > 0) ||
                       (structure.ctes && structure.ctes.length > 0);
    
    if (!hasContent) {
        return '<div style="margin-top: 16px; padding: 12px; background: #1e293b; border: 1px solid #334155; border-radius: 8px; color: #94a3b8; flex: 1;">暂无结构分析数据</div>';
    }
    
    const hasSubquery = structure.subqueries && structure.subqueries.length > 0;
    const hasWindow = structure.window_functions && structure.window_functions.length > 0;
    const hasIndex = indexSuggestions && indexSuggestions.length > 0;
    
    let html = '<div style="margin-top: 12px; flex: 1; display: flex; flex-direction: column; min-height: 0;">';
    html += '<div style="font-size: 14px; font-weight: 600; color: #f1f5f9; margin-bottom: 10px; flex-shrink: 0;">🔍 SQL结构分析</div>';
    html += '<div style="flex: 1; display: flex; flex-direction: column; gap: 10px; min-height: 0; overflow-y: auto; padding-right: 4px;">';
    
    // CTE - 更详细
    if (structure.ctes && structure.ctes.length > 0) {
        let cteContent = '<div style="display: flex; flex-wrap: wrap; gap: 8px;">';
        structure.ctes.forEach((c, idx) => {
            cteContent += `<div style="padding: 8px 12px; background: #334155; border-radius: 6px; border-left: 3px solid #c084fc;">
                <div style="display: flex; align-items: center; gap: 8px;">
                    <span style="color: #64748b; font-size: 11px;">#${idx + 1}</span>
                    <span class="badge badge-success" style="font-size: 12px;">${escapeHTML(c.name)}</span>
                </div>
                <div style="margin-top: 4px; font-size: 11px; color: #94a3b8;">公共表表达式</div>
            </div>`;
        });
        cteContent += '</div>';
        html += generateClauseBox(`WITH (CTE) - ${structure.ctes.length}个`, '#c084fc', cteContent, false);
    }
    
    // 涉及表 - 使用新的 tables_with_alias 数据
    if (data.tables_with_alias && data.tables_with_alias.length > 0) {
        let content = '<div style="display: flex; flex-direction: column; gap: 8px;">';
        data.tables_with_alias.forEach((table, idx) => {
            const hasAlias = table.alias && table.alias !== '' && table.alias !== table.name;
            const isSubquery = table.is_subquery || false;
            const isCTE = table.is_cte || false;
            
            // 表类型判断
            let tableType = '真实表';
            let typeColor = '#4ade80';
            let typeBg = 'rgba(74, 222, 128, 0.15)';
            
            if (isSubquery) {
                tableType = '派生表';
                typeColor = '#fbbf24';
                typeBg = 'rgba(251, 191, 36, 0.15)';
            } else if (isCTE) {
                tableType = 'CTE';
                typeColor = '#c084fc';
                typeBg = 'rgba(192, 132, 252, 0.15)';
            }
            
            content += `<div style="padding: 10px 12px; background: #334155; border-radius: 6px;">
                <div style="display: flex; align-items: center; gap: 8px; flex-wrap: wrap;">
                    <span style="color: #64748b; font-size: 11px;">#${idx + 1}</span>
                    <span style="color: #60a5fa; font-weight: 600; font-size: 13px;">${escapeHTML(table.name)}</span>
                    ${hasAlias && !isSubquery ? `<span style="color: #64748b; font-size: 12px;">→</span> <span style="color: #4ade80; font-weight: 500; font-size: 13px;">${escapeHTML(table.alias)}</span>` : (isSubquery ? '' : '<span style="color: #64748b; font-size: 11px;">(无别名)</span>')}
                    <span style="padding: 2px 8px; border-radius: 4px; font-size: 10px; background: ${typeBg}; color: ${typeColor};">${tableType}</span>
                </div>
            </div>`;
        });
        content += '</div>';
        html += generateClauseBox(`📋 涉及表 (${data.tables_with_alias.length}个)`, '#4ade80', content, false);
    } else if (structure.from_clause) {
        // 降级：使用旧的 structure.from_clause 数据
        const fc = structure.from_clause;
        let content = '';
        if (fc.main_table && fc.main_table.name) {
            content += `<div style="padding: 10px 12px; background: #334155; border-radius: 6px; margin-bottom: 8px;">
                <div style="display: flex; align-items: center; gap: 8px; flex-wrap: wrap;">
                    <span style="color: #94a3b8; font-size: 12px;">主表:</span>
                    <span class="badge badge-info" style="font-size: 12px;">${escapeHTML(fc.main_table.name)}</span>
                    ${fc.main_table.alias ? `<span style="color: #64748b;">AS</span> <span style="color: #4ade80; font-weight: 500;">${fc.main_table.alias}</span>` : ''}
                </div>
            </div>`;
        }
        if (fc.joins && fc.joins.length > 0) {
            content += `<div style="font-size: 12px; color: #94a3b8; margin-bottom: 8px;">JOIN连接 (${fc.joins.length}个):</div>`;
            content += '<div style="display: flex; flex-direction: column; gap: 8px;">';
            fc.joins.forEach((j, idx) => {
                content += `<div style="padding: 10px 12px; background: #334155; border-radius: 6px; font-size: 12px;">
                    <div style="display: flex; align-items: center; gap: 8px; flex-wrap: wrap;">
                        <span style="color: #64748b; font-size: 11px;">#${idx + 1}</span>
                        <span class="badge ${j.type === 'LEFT' ? 'badge-warning' : j.type === 'RIGHT' ? 'badge-danger' : j.type === 'CROSS' ? 'badge-secondary' : 'badge-info'}">${j.type || 'INNER'} JOIN</span>
                        <span style="color: #60a5fa; font-weight: 600;">${escapeHTML(j.table)}</span>
                        ${j.alias ? `<span style="color: #64748b;">AS</span> <span style="color: #4ade80; font-weight: 500;">${j.alias}</span>` : ''}
                    </div>
                    ${j.condition ? `<div style="margin-top: 6px; padding: 6px 10px; background: #1e293b; border-radius: 4px; color: #fbbf24; font-family: monospace; font-size: 11px;">ON ${escapeHTML(j.condition)}</div>` : ''}
                </div>`;
            });
            content += '</div>';
        }
        html += generateClauseBox(`FROM (${fc.joins ? fc.joins.length + 1 : 1}表)`, '#4ade80', content, false);
    }
    
    // SELECT + 子查询 + 窗口函数 + 索引建议 并排
    if (structure.select_clause) {
        const sc = structure.select_clause;
        let selectContent = '';
        if (sc.has_star) {
            selectContent += '<div style="color: #fbbf24; margin-bottom: 8px;">⚠️ 使用了 SELECT *（建议明确指定字段）</div>';
        }
        if (sc.fields && sc.fields.length > 0) {
            selectContent += `<div style="margin-bottom: 8px; font-size: 12px; color: #94a3b8;">共 <span style="color: #60a5fa; font-weight: 600;">${sc.fields.length}</span> 个字段</div>`;
            selectContent += generateFieldsTable(sc.fields);
        }
        if (sc.aggregates && sc.aggregates.length > 0) {
            const uniqueAggs = [...new Set(sc.aggregates)];
            selectContent += `<div style="margin-top: 10px;"><span style="color: #4ade80;">聚合函数:</span> ${uniqueAggs.map(a => `<span class="badge badge-success">${a}</span>`).join(' ')}</div>`;
        }
        
        // 计算有多少个面板
        const panels = [];
        panels.push({ type: 'select', content: selectContent });
        if (hasSubquery) panels.push({ type: 'subquery' });
        if (hasWindow) panels.push({ type: 'window' });
        if (hasIndex) panels.push({ type: 'index' });
        
        if (panels.length > 1) {
            // 多个面板并排
            html += '<div style="display: flex; gap: 10px; flex: 1; min-height: 180px;">';
            
            panels.forEach(panel => {
                html += '<div style="flex: 1; display: flex; flex-direction: column; min-width: 0;">';
                
                if (panel.type === 'select') {
                    html += generateClauseBox('SELECT', '#60a5fa', selectContent, true);
                } else if (panel.type === 'subquery') {
                    let subContent = '<div style="display: flex; flex-direction: column; gap: 10px;">';
                    structure.subqueries.forEach(sq => {
                        subContent += `<div style="padding: 10px; background: #334155; border-radius: 6px; border-left: 3px solid #fbbf24;">`;
                        subContent += `<div style="margin-bottom: 6px;"><span class="badge badge-warning" style="font-size: 11px;">${sq.location || '子查询'}</span></div>`;
                        subContent += `<div style="font-family: monospace; font-size: 11px; color: #94a3b8; word-break: break-all;">${escapeHTML(sq.raw.length > 100 ? sq.raw.substring(0, 100) + '...' : sq.raw)}</div></div>`;
                    });
                    subContent += '</div>';
                    html += generateClauseBox(`子查询 (${structure.subqueries.length})`, '#fbbf24', subContent, true);
                } else if (panel.type === 'window') {
                    let winContent = '<div style="display: flex; flex-direction: column; gap: 10px;">';
                    structure.window_functions.forEach(wf => {
                        winContent += `<div style="padding: 10px; background: #334155; border-radius: 6px; border-left: 3px solid #c084fc;">`;
                        winContent += `<div style="margin-bottom: 6px;"><span class="badge badge-success" style="font-size: 11px;">${wf.function}</span> <span style="color: #94a3b8; font-size: 11px;">OVER</span></div>`;
                        if (wf.partition_by) winContent += `<div style="font-size: 11px;"><span style="color: #94a3b8;">PARTITION:</span> <span style="color: #60a5fa;">${escapeHTML(wf.partition_by)}</span></div>`;
                        if (wf.order_by) winContent += `<div style="font-size: 11px;"><span style="color: #94a3b8;">ORDER:</span> <span style="color: #4ade80;">${escapeHTML(wf.order_by)}</span></div>`;
                        winContent += '</div>';
                    });
                    winContent += '</div>';
                    html += generateClauseBox(`窗口函数 (${structure.window_functions.length})`, '#c084fc', winContent, true);
                } else if (panel.type === 'index') {
                    html += generateIndexPanel(indexSuggestions);
                }
                
                html += '</div>';
            });
            
            html += '</div>';
        } else {
            // 只有SELECT，单独一行
            html += generateClauseBox('SELECT', '#60a5fa', selectContent, true);
            // 如果有索引建议但没有子查询和窗口函数，索引建议单独显示
            if (hasIndex) {
                html += generateIndexPanel(indexSuggestions);
            }
        }
    }
    
    // WHERE子句 - 限制高度
    if (structure.where_clause) {
        const wc = structure.where_clause;
        let content = '';
        if (wc.conditions && wc.conditions.length > 0) {
            content += '<div style="font-size: 11px; color: #94a3b8; margin-bottom: 6px;">过滤条件:</div>';
            content += '<div style="display: flex; flex-direction: column; gap: 4px; max-height: 200px; overflow-y: auto;">';
            wc.conditions.forEach(c => {
                let display = c.length > 100 ? c.substring(0, 100) + '...' : c;
                content += `<div style="padding: 6px 10px; background: #334155; border-radius: 4px; font-family: monospace; font-size: 11px; color: #fbbf24;" title="${escapeHTML(c)}">${escapeHTML(display)}</div>`;
            });
            content += '</div>';
        }
        if (wc.fields && wc.fields.length > 0) {
            content += `<div style="margin-top: 8px;"><span style="color: #94a3b8; font-size: 11px;">涉及字段:</span> <span style="display: inline-flex; flex-wrap: wrap; gap: 4px; margin-left: 4px;">${wc.fields.map(f => `<span class="badge badge-secondary" style="font-size: 10px;">${escapeHTML(f)}</span>`).join('')}</span></div>`;
        }
        html += generateClauseBoxCompact('WHERE', '#fbbf24', content);
    }
    
    // GROUP BY子句 - 限制高度
    if (structure.group_by_clause) {
        const gc = structure.group_by_clause;
        let content = '<div style="display: flex; flex-wrap: wrap; gap: 6px; max-height: 80px; overflow-y: auto;">';
        if (gc.fields && gc.fields.length > 0) {
            gc.fields.forEach(f => { content += `<span class="badge badge-info">${escapeHTML(f)}</span>`; });
        }
        content += '</div>';
        html += generateClauseBoxCompact('GROUP BY', '#c084fc', content);
    }
    
    // HAVING子句 - 限制高度
    if (structure.having_clause) {
        const hc = structure.having_clause;
        let content = '<div style="display: flex; flex-direction: column; gap: 4px; max-height: 120px; overflow-y: auto;">';
        if (hc.conditions && hc.conditions.length > 0) {
            hc.conditions.forEach(c => {
                content += `<div style="padding: 6px 10px; background: #334155; border-radius: 4px; font-family: monospace; font-size: 11px; color: #f472b6;">${escapeHTML(c)}</div>`;
            });
        }
        content += '</div>';
        html += generateClauseBoxCompact('HAVING', '#f472b6', content);
    }
    
    // ORDER BY子句 - 限制高度
    if (structure.order_by_clause) {
        const oc = structure.order_by_clause;
        let content = '<div style="display: flex; flex-wrap: wrap; gap: 8px; max-height: 80px; overflow-y: auto;">';
        if (oc.fields && oc.fields.length > 0) {
            oc.fields.forEach(f => {
                const dirColor = f.direction === 'DESC' ? '#f87171' : '#4ade80';
                content += `<span style="background: #334155; padding: 4px 10px; border-radius: 4px; font-size: 12px;"><span style="color: #60a5fa;">${escapeHTML(f.field)}</span> <span style="color: ${dirColor}; font-weight: 500;">${f.direction}</span></span>`;
            });
        }
        content += '</div>';
        html += generateClauseBoxCompact('ORDER BY', '#60a5fa', content);
    }
    
    // LIMIT子句 - 紧凑
    if (structure.limit_clause) {
        const lc = structure.limit_clause;
        let content = `<span style="color: #4ade80; font-size: 14px; font-weight: 600;">LIMIT ${lc.limit}</span>`;
        if (lc.offset > 0) content += ` <span style="color: #94a3b8; margin-left: 8px;">OFFSET</span> <span style="color: #fbbf24; font-weight: 500;">${lc.offset}</span>`;
        html += generateClauseBoxCompact('LIMIT', '#4ade80', content);
    }
    
    html += '</div></div>';
    return html;
}

// 生成索引建议面板（用于并排显示）
function generateIndexPanel(suggestions) {
    const priorityColors = {
        'high': { bg: 'rgba(239, 68, 68, 0.15)', color: '#f87171', text: '高' },
        'medium': { bg: 'rgba(251, 191, 36, 0.15)', color: '#fbbf24', text: '中' },
        'low': { bg: 'rgba(74, 222, 128, 0.15)', color: '#4ade80', text: '低' }
    };
    
    let content = '<div style="display: flex; flex-direction: column; gap: 10px;">';
    
    suggestions.forEach(s => {
        const priority = priorityColors[s.priority] || priorityColors['low'];
        content += `<div style="padding: 10px; background: #0f172a; border: 1px solid #334155; border-radius: 6px;">
            <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-wrap: wrap;">
                <span style="padding: 3px 8px; border-radius: 4px; font-size: 11px; background: ${priority.bg}; color: ${priority.color};">${priority.text}</span>
                <span style="color: #60a5fa; font-weight: 600; font-size: 13px;">${escapeHTML(s.table)}</span>
            </div>
            <div style="margin-bottom: 8px; display: flex; flex-wrap: wrap; gap: 4px;">
                ${s.columns.map(c => `<span class="badge badge-info" style="font-size: 11px;">${escapeHTML(c)}</span>`).join('')}
            </div>
            <div style="display: flex; align-items: center; gap: 8px;">
                <code style="flex: 1; padding: 6px 8px; background: #1e293b; border: 1px solid #334155; border-radius: 4px; font-size: 11px; color: #4ade80; font-family: monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${escapeHTML(s.create_sql)}">${escapeHTML(s.create_sql)}</code>
                <button onclick="copyToClipboard('${escapeHTML(s.create_sql).replace(/'/g, "\\'")}')" style="padding: 4px 8px; background: #334155; border: none; border-radius: 4px; color: #94a3b8; font-size: 11px; cursor: pointer;">复制</button>
            </div>
        </div>`;
    });
    
    content += '</div>';
    return generateClauseBox(`💡 索引建议 (${suggestions.length})`, '#f59e0b', content, true);
}

// 生成字段表格（表头固定）
function generateFieldsTable(fields) {
    if (!fields || fields.length === 0) return '';
    
    const typeLabels = {
        'column': { text: '字段', color: '#60a5fa', bg: 'rgba(96, 165, 250, 0.15)' },
        'function': { text: '函数', color: '#fbbf24', bg: 'rgba(251, 191, 36, 0.15)' },
        'aggregate': { text: '聚合', color: '#4ade80', bg: 'rgba(74, 222, 128, 0.15)' },
        'window': { text: '窗口', color: '#c084fc', bg: 'rgba(192, 132, 252, 0.15)' },
        'expression': { text: '表达式', color: '#f472b6', bg: 'rgba(244, 114, 182, 0.15)' },
        'star': { text: '*', color: '#f87171', bg: 'rgba(248, 113, 113, 0.15)' }
    };
    
    let html = `<div style="border: 1px solid #334155; border-radius: 6px; overflow: hidden;">
        <table style="width: 100%; border-collapse: collapse; font-size: 12px;">
            <thead>
                <tr style="background: #0f172a;">
                    <th style="padding: 8px 10px; text-align: left; color: #94a3b8; font-weight: 500; width: 36px; border-bottom: 1px solid #334155;">#</th>
                    <th style="padding: 8px 10px; text-align: left; color: #94a3b8; font-weight: 500; border-bottom: 1px solid #334155;">表达式</th>
                    <th style="padding: 8px 10px; text-align: left; color: #94a3b8; font-weight: 500; width: 60px; border-bottom: 1px solid #334155;">类型</th>
                    <th style="padding: 8px 10px; text-align: left; color: #94a3b8; font-weight: 500; width: 90px; border-bottom: 1px solid #334155;">来源表</th>
                    <th style="padding: 8px 10px; text-align: left; color: #94a3b8; font-weight: 500; width: 90px; border-bottom: 1px solid #334155;">别名</th>
                </tr>
            </thead>
            <tbody>`;
    
    fields.forEach((f, idx) => {
        const typeInfo = typeLabels[f.field_type] || typeLabels['expression'];
        const expression = f.expression || '';
        const displayExpr = expression.length > 50 ? expression.substring(0, 50) + '...' : expression;
        const sourceTable = f.source_table || '-';
        const alias = f.alias || '-';
        
        html += `<tr style="background: ${idx % 2 === 0 ? 'transparent' : 'rgba(51, 65, 85, 0.3)'};">
                <td style="padding: 6px 10px; color: #64748b; border-bottom: 1px solid #262f3d;">${idx + 1}</td>
                <td style="padding: 6px 10px; color: #e2e8f0; font-family: monospace; font-size: 11px; border-bottom: 1px solid #262f3d;" title="${escapeHTML(expression)}">${escapeHTML(displayExpr)}</td>
                <td style="padding: 6px 10px; border-bottom: 1px solid #262f3d;"><span style="display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 10px; color: ${typeInfo.color}; background: ${typeInfo.bg};">${typeInfo.text}</span></td>
                <td style="padding: 6px 10px; color: ${sourceTable !== '-' ? '#60a5fa' : '#64748b'}; font-size: 11px; border-bottom: 1px solid #262f3d;">${escapeHTML(sourceTable)}</td>
                <td style="padding: 6px 10px; color: ${alias !== '-' ? '#4ade80' : '#64748b'}; font-size: 11px; border-bottom: 1px solid #262f3d;">${escapeHTML(alias)}</td>
            </tr>`;
    });
    
    html += '</tbody></table></div>';
    return html;
}

// 生成子句框
function generateClauseBox(title, color, content, expandable) {
    const flexStyle = expandable ? 'flex: 1; display: flex; flex-direction: column;' : '';
    const contentStyle = expandable ? 'flex: 1; overflow-y: auto;' : '';
    return `<div style="background: #1e293b; border: 1px solid #334155; border-left: 4px solid ${color}; border-radius: 8px; overflow: hidden; ${flexStyle}">
        <div style="padding: 8px 12px; background: rgba(51, 65, 85, 0.5); border-bottom: 1px solid #334155; flex-shrink: 0;">
            <span style="font-size: 12px; font-weight: 600; color: ${color};">${title}</span>
        </div>
        <div style="padding: 10px 12px; font-size: 12px; ${contentStyle}">
            ${content}
        </div>
    </div>`;
}

// 生成紧凑子句框（用于WHERE、GROUP BY等）
function generateClauseBoxCompact(title, color, content) {
    return `<div style="background: #1e293b; border: 1px solid #334155; border-left: 4px solid ${color}; border-radius: 8px; overflow: hidden; flex-shrink: 0;">
        <div style="padding: 6px 12px; background: rgba(51, 65, 85, 0.5); border-bottom: 1px solid #334155;">
            <span style="font-size: 12px; font-weight: 600; color: ${color};">${title}</span>
        </div>
        <div style="padding: 8px 12px; font-size: 12px;">
            ${content}
        </div>
    </div>`;
}

// 默认结果显示
function showDefaultResult(resultDiv, data) {
    resultDiv.innerHTML = `
        <div class="stats-grid" style="grid-template-columns: repeat(3, 1fr);">
            <div class="stat-card"><div class="stat-label">SQL类型</div><div class="stat-value"><span class="badge badge-secondary">${data.sql_type}</span></div></div>
            <div class="stat-card"><div class="stat-label">SQL分类</div><div class="stat-value"><span class="badge badge-secondary">${data.sql_category}</span></div></div>
            <div class="stat-card"><div class="stat-label">数据库</div><div class="stat-value" style="font-size: 13px;">${data.db_name || '默认'}</div></div>
        </div>
    `;
}

// 复制到剪贴板
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('复制成功', 'SQL已复制到剪贴板', 'success');
    }).catch(() => {
        showToast('复制失败', '请手动复制', 'error');
    });
}
