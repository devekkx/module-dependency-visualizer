import * as d3 from "https://cdn.jsdelivr.net/npm/d3@7/+esm";

//  MDV - Module Dependency Visualiser  (D3.js v7)
const CFG = {
    nodeRadius:      8,
    mainRadius:      12,
    linkDistance:    120,
    chargeStrength:  -350,
    collideRadius:   22,
    alphaDecay:      0.028,
    transitionMs:    250,
    colors: {
        main:    '#004e2c',
        module:  '#2980b9',
        dev:     '#7f8c8d',
        vuln:    '#e53e3e',
        link:    '#aab8c2',
        hlLink:  '#e67e22',
    },
};

// State
const state = {
    allNodes:  [],
    allLinks:  [],
    search:    '',
    maxDepth:  Infinity,
    filters:   { direct: true, indirect: true, dev: true, vulnOnly: false },
    viewMode:  'graph',  // 'graph' | 'table'
    selected:  null,
    // audit
    audit:     null,  // schema.AuditDTO once loaded
    vulnMap:   {},    // nodeID → [VulnDTO]
};

const VIEW_ICONS = {
    graph:   `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="5" r="2.5"/><circle cx="5" cy="19" r="2.5"/><circle cx="19" cy="19" r="2.5"/><line x1="12" y1="7.5" x2="5.8" y2="16.8"/><line x1="12" y1="7.5" x2="18.2" y2="16.8"/></svg>`,
    table:   `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="9" x2="9" y2="21"/></svg>`,
    tree:    `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="4" r="2"/><circle cx="6" cy="18" r="2"/><circle cx="18" cy="18" r="2"/><line x1="12" y1="6" x2="12" y2="11"/><line x1="12" y1="11" x2="6" y2="16"/><line x1="12" y1="11" x2="18" y2="16"/></svg>`,
    treemap: `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><rect x="3" y="3" width="9" height="11"/><rect x="3" y="16" width="9" height="5"/><rect x="14" y="3" width="7" height="5"/><rect x="14" y="10" width="7" height="11"/></svg>`,
    list:    `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><line x1="9" y1="6" x2="20" y2="6"/><line x1="9" y1="12" x2="20" y2="12"/><line x1="9" y1="18" x2="20" y2="18"/><circle cx="5" cy="6" r="1.5" fill="currentColor" stroke="none"/><circle cx="5" cy="12" r="1.5" fill="currentColor" stroke="none"/><circle cx="5" cy="18" r="1.5" fill="currentColor" stroke="none"/></svg>`,
};

// D3 selections / simulation
let svgEl, gEl, simulation;
let linkSel, nodeSel, labelSel, vulnRingSel;

// Bootstrap
document.addEventListener('DOMContentLoaded', init);

async function init() {
    try {
        const res = await fetch('/api/graph');
        if (!res.ok) throw new Error(`Server returned ${res.status}: ${res.statusText}`);
        const data = await res.json();

        state.allNodes = (data.nodes  || []).map(n => ({ ...n }));
        state.allLinks = (data.edges  || []).map(e => ({
            source: e.from,
            target: e.to,
            kind:   e.kind || 'depends_on',
        }));

        // Pre-load audit data embedded in the document (from mdv analyze --audit).
        if (data.audit) {
            applyAuditData(data.audit);
        }

        document.getElementById('loading').classList.add('hidden');
        document.getElementById('app').classList.remove('hidden');

        applyMeta(data);
        setupSVG();
        setupControls();
        document.getElementById('view-icon').innerHTML = VIEW_ICONS.graph;
        render();
    } catch (err) {
        document.getElementById('loading').classList.add('hidden');
        const banner = document.getElementById('error-banner');
        banner.querySelector('.error-msg').textContent = err.message;
        banner.classList.remove('hidden');
    }
}

function applyMeta(data) {
    const proj = data.project || {};
    const total = state.allNodes.length;
    const edges = state.allLinks.length;

    document.getElementById('stat-nodes').textContent = `${total} node${total !== 1 ? 's' : ''}`;
    document.getElementById('stat-edges').textContent = `${edges} edge${edges !== 1 ? 's' : ''}`;

    const name = proj.name || '';
    const lang = proj.language || '';

    if (name) {
        document.getElementById('project-name').textContent = name;
        document.title = `MDV - ${name}`;
    }
    if (lang) {
        const badge = document.getElementById('lang-badge');
        badge.textContent = lang;
        badge.classList.remove('hidden');
    }
}

// SVG + Zoom
function setupSVG() {
    const container = document.getElementById('graph-container');

    svgEl = d3.select('#graph-container')
        .append('svg')
        .attr('width',  '100%')
        .attr('height', '100%');

    // Arrow markers
    const defs = svgEl.append('defs');
    const mkMarker = (id, color) =>
        defs.append('marker')
            .attr('id',          id)
            .attr('viewBox',     '0 -4 8 8')
            .attr('refX',        28)
            .attr('refY',        0)
            .attr('markerWidth', 6)
            .attr('markerHeight', 6)
            .attr('orient',      'auto')
            .append('path')
            .attr('d',    'M0,-4L8,0L0,4')
            .attr('fill', color);

    mkMarker('arrow',    CFG.colors.link);
    mkMarker('arrow-hl', CFG.colors.hlLink);

    // Zoom
    const zoom = d3.zoom()
        .scaleExtent([0.05, 8])
        .on('zoom', ({ transform }) => gEl.attr('transform', transform));

    svgEl.call(zoom);

    gEl = svgEl.append('g');
    gEl.append('g').attr('class', 'links');
    gEl.append('g').attr('class', 'vuln-rings');
    gEl.append('g').attr('class', 'nodes');
    gEl.append('g').attr('class', 'labels');

    // Dismiss sidebar when clicking the empty canvas.
    svgEl.on('click', () => { state.selected = null; closeSidebar(); });

    // Resize → recenter
    new ResizeObserver(() => {
        if (!simulation) return;
        const { width: w, height: h } = container.getBoundingClientRect();
        simulation.force('center', d3.forceCenter(w / 2, h / 2)).alpha(0.1).restart();
    }).observe(container);
}

// Controls
function setupControls() {
    document.getElementById('search').addEventListener('input', function () {
        state.search = this.value.trim().toLowerCase();
        if (state.viewMode === 'table' || state.viewMode === 'list') {
            render();
        } else {
            updateHighlight();
        }
    });

    const slider    = document.getElementById('depth-slider');
    const depthLbl  = document.getElementById('depth-value');
    slider.addEventListener('input', () => {
        const v = parseInt(slider.value, 10);
        state.maxDepth = (v >= parseInt(slider.max, 10)) ? Infinity : v;
        depthLbl.textContent = state.maxDepth === Infinity ? '∞' : String(state.maxDepth);
        render();
    });

    // Dropdown filter
    const filterTrigger = document.getElementById('filter-trigger');
    const filterMenu    = document.getElementById('filter-menu');

    filterTrigger.addEventListener('click', e => {
        e.stopPropagation();
        filterMenu.classList.toggle('hidden');
    });

    document.addEventListener('click', e => {
        if (!document.getElementById('filter-dropdown').contains(e.target)) {
            filterMenu.classList.add('hidden');
        }
    });

    filterMenu.querySelectorAll('input[type="checkbox"]').forEach(cb => {
        cb.addEventListener('change', () => {
            const f = cb.dataset.filter;
            if (f === 'vuln-only') {
                state.filters.vulnOnly = cb.checked;
            } else if (f === 'direct') {
                state.filters.direct = cb.checked;
            } else if (f === 'indirect') {
                state.filters.indirect = cb.checked;
            } else if (f === 'dev') {
                state.filters.dev = cb.checked;
            }
            updateFilterBadge();
            render();
        });
    });

    document.getElementById('sidebar-close').addEventListener('click', closeSidebar);

    // View dropdown
    setupViewDropdown();

    // Audit button
    document.getElementById('audit-btn').addEventListener('click', openAuditPanel);
    document.getElementById('audit-close').addEventListener('click', closeAuditPanel);
    document.getElementById('audit-overlay').addEventListener('click', e => {
        if (e.target === document.getElementById('audit-overlay')) closeAuditPanel();
    });
}

// View dropdown
function setupViewDropdown() {
    const trigger = document.getElementById('view-trigger');
    const menu    = document.getElementById('view-menu');

    trigger.addEventListener('click', e => {
        e.stopPropagation();
        menu.classList.toggle('hidden');
    });

    document.addEventListener('click', e => {
        if (!document.getElementById('view-dropdown').contains(e.target)) {
            menu.classList.add('hidden');
        }
    });

    menu.querySelectorAll('.view-option').forEach(btn => {
        btn.addEventListener('click', () => {
            menu.classList.add('hidden');
            switchView(btn.dataset.view);
        });
    });
}

const ALL_VIEWS = ['graph', 'table', 'tree', 'treemap', 'list'];

function switchView(view) {
    state.viewMode = view;

    ALL_VIEWS.forEach(v => {
        const el = document.getElementById(`${v}-container`);
        if (el) el.classList.toggle('hidden', v !== view);
    });

    document.getElementById('view-label').textContent =
        view.charAt(0).toUpperCase() + view.slice(1);
    document.getElementById('view-icon').innerHTML = VIEW_ICONS[view] || '';

    document.querySelectorAll('.view-option').forEach(opt => {
        opt.classList.toggle('active', opt.dataset.view === view);
    });

    render();
}

// BFS depth filtering
function bfsFromRoot(maxD) {
    const root = state.allNodes.find(n => n.kind === 'main') || state.allNodes[0];
    if (!root) return new Set();
    if (maxD === Infinity) return new Set(state.allNodes.map(n => n.id));

    // Build adjacency from original specs (before D3 replaces strings with objects).
    const adj = {};
    state.allLinks.forEach(l => {
        const src = typeof l.source === 'object' ? l.source.id : l.source;
        const tgt = typeof l.target === 'object' ? l.target.id : l.target;
        (adj[src] ??= []).push(tgt);
    });

    const visited = new Set();
    const queue = [[root.id, 0]];
    while (queue.length) {
        const [id, d] = queue.shift();
        if (visited.has(id) || d > maxD) continue;
        visited.add(id);
        (adj[id] || []).forEach(n => queue.push([n, d + 1]));
    }
    return visited;
}

function updateFilterBadge() {
    const { direct, indirect, dev, vulnOnly } = state.filters;
    const badge = document.getElementById('filter-badge');
    if (vulnOnly) {
        badge.textContent = 'Vuln only';
        badge.style.cssText = 'background:rgba(229,62,62,0.25);color:#fc8181';
        return;
    }
    const active = [direct && 'Direct', indirect && 'Indirect', dev && 'Dev'].filter(Boolean);
    if (active.length === 3) {
        badge.textContent = 'All';
        badge.style.cssText = '';
    } else if (active.length === 0) {
        badge.textContent = 'None';
        badge.style.cssText = 'background:rgba(255,255,255,0.06);color:#7f8c8d';
    } else {
        badge.textContent = active.join(' & ');
        badge.style.cssText = 'background:rgba(52,152,219,0.2);color:#3498db';
    }
}

function getVisible() {
    const depthSet = bfsFromRoot(state.maxDepth);
    const { direct, indirect, dev, vulnOnly } = state.filters;

    const visNodes = state.allNodes.filter(n => {
        if (!depthSet.has(n.id)) return false;
        if (n.kind === 'main') return true;
        if (vulnOnly) return (state.vulnMap[n.id]?.length ?? 0) > 0;
        if (n.dev) return dev;
        if (n.indirect) return indirect;
        return direct;
    });
    const visIds    = new Set(visNodes.map(n => n.id));
    const visLinks  = state.allLinks.filter(l => {
        const src = typeof l.source === 'object' ? l.source.id : l.source;
        const tgt = typeof l.target === 'object' ? l.target.id : l.target;
        return visIds.has(src) && visIds.has(tgt);
    });
    return { nodes: visNodes, links: visLinks };
}

// Render
function render() {
    const { nodes, links } = getVisible();
    const visCount = nodes.length;

    document.getElementById('stat-visible').textContent =
        visCount !== state.allNodes.length ? `${visCount} visible` : '';

    if (state.viewMode === 'table') {
        document.getElementById('empty-state').classList.add('hidden');
        renderTable(nodes);
        return;
    }
    if (state.viewMode === 'tree') {
        document.getElementById('empty-state').classList.add('hidden');
        renderTree(nodes);
        updateHighlight();
        return;
    }
    if (state.viewMode === 'treemap') {
        document.getElementById('empty-state').classList.add('hidden');
        renderTreemap(nodes);
        updateHighlight();
        return;
    }
    if (state.viewMode === 'list') {
        document.getElementById('empty-state').classList.add('hidden');
        renderList(nodes);
        return;
    }

    document.getElementById('empty-state').classList.toggle('hidden', visCount > 0);

    const container = document.getElementById('graph-container');
    const { width: W, height: H } = container.getBoundingClientRect();

    if (simulation) simulation.stop();

    simulation = d3.forceSimulation(nodes)
        .force('link',    d3.forceLink(links).id(d => d.id).distance(CFG.linkDistance))
        .force('charge',  d3.forceManyBody().strength(CFG.chargeStrength))
        .force('center',  d3.forceCenter(W / 2, H / 2))
        .force('collide', d3.forceCollide(CFG.collideRadius))
        .alphaDecay(CFG.alphaDecay);

    const linkKey = d =>
        `${typeof d.source === 'object' ? d.source.id : d.source}→${typeof d.target === 'object' ? d.target.id : d.target}`;

    // Links
    linkSel = gEl.select('.links').selectAll('line')
        .data(links, linkKey)
        .join(
            enter => enter.append('line')
                .attr('stroke',      CFG.colors.link)
                .attr('stroke-width', 1.4)
                .attr('opacity', 0)
                .attr('marker-end', 'url(#arrow)')
                .call(el => el.transition().duration(CFG.transitionMs).attr('opacity', 1)),
            update => update,
            exit   => exit.transition().duration(CFG.transitionMs).attr('opacity', 0).remove()
        );

    // Vulnerability rings (drawn behind nodes)
    vulnRingSel = gEl.select('.vuln-rings').selectAll('circle')
        .data(nodes.filter(d => state.vulnMap[d.id]?.length > 0), d => d.id)
        .join(
            enter => enter.append('circle')
                .attr('class', 'node-vuln-ring')
                .attr('r', d => (d.kind === 'main' ? CFG.mainRadius : CFG.nodeRadius) + 5),
            update => update,
            exit   => exit.remove()
        );

    // Nodes
    nodeSel = gEl.select('.nodes').selectAll('circle')
        .data(nodes, d => d.id)
        .join(
            enter => {
                const c = enter.append('circle')
                    .attr('class',        'node-circle')
                    .attr('r',            d => d.kind === 'main' ? CFG.mainRadius : CFG.nodeRadius)
                    .attr('fill',         d => nodeColor(d))
                    .attr('stroke',       '#fff')
                    .attr('stroke-width', 2)
                    .attr('cursor',       'pointer')
                    .attr('opacity', 0)
                    .call(el => el.transition().duration(CFG.transitionMs).attr('opacity', 1));

                c.call(dragBehaviour(simulation));
                c.on('click',     (ev, d) => { ev.stopPropagation(); showDetail(d); });
                c.on('mouseover', showTooltip);
                c.on('mouseout',  hideTooltip);
                return c;
            },
            update => update.attr('fill', d => nodeColor(d)),
            exit   => exit.transition().duration(CFG.transitionMs).attr('opacity', 0).remove()
        );

    // Labels
    labelSel = gEl.select('.labels').selectAll('text')
        .data(nodes, d => d.id)
        .join(
            enter => enter.append('text')
                .attr('class',        'node-label')
                .attr('dy',           d => (d.kind === 'main' ? CFG.mainRadius : CFG.nodeRadius) + 14)
                .attr('text-anchor',  'middle')
                .attr('font-size',    '11px')
                .attr('fill',         '#2c3e50')
                .attr('opacity', 0)
                .text(d => d.name)
                .call(el => el.transition().duration(CFG.transitionMs).attr('opacity', 1)),
            update => update.text(d => d.name),
            exit   => exit.transition().duration(CFG.transitionMs).attr('opacity', 0).remove()
        );

    simulation.on('tick', () => {
        linkSel
            .attr('x1', d => d.source.x).attr('y1', d => d.source.y)
            .attr('x2', d => d.target.x).attr('y2', d => d.target.y);
        nodeSel .attr('cx', d => d.x).attr('cy', d => d.y);
        labelSel.attr('x',  d => d.x).attr('y',  d => d.y);
        if (vulnRingSel) {
            vulnRingSel.attr('cx', d => d.x).attr('cy', d => d.y);
        }
    });

    updateHighlight();
}

// Highlight (search)
function updateHighlight() {
    const q = state.search;

    if (state.viewMode === 'graph') {
        if (!nodeSel) return;
        nodeSel.attr('opacity', d =>
            q && !d.name.toLowerCase().includes(q) ? 0.12 : 1
        );
        labelSel.attr('opacity', d =>
            q && !d.name.toLowerCase().includes(q) ? 0.08 : 1
        );
        linkSel.attr('opacity', d => {
            if (!q) return 1;
            const sn = typeof d.source === 'object' ? d.source.name : '';
            const tn = typeof d.target === 'object' ? d.target.name : '';
            return (sn.toLowerCase().includes(q) || tn.toLowerCase().includes(q)) ? 0.7 : 0.05;
        });
    } else if (state.viewMode === 'tree') {
        d3.selectAll('#tree-container .tree-node').attr('opacity', d =>
            q && d.data && !d.data.name.toLowerCase().includes(q) ? 0.12 : 1
        );
    } else if (state.viewMode === 'treemap') {
        d3.selectAll('#treemap-container .tm-cell').attr('opacity', d =>
            q && d.depth > 0 && !d.data.name.toLowerCase().includes(q) ? 0.15 : 1
        );
    }
}

// Drag
function dragBehaviour(sim) {
    return d3.drag()
        .on('start', (ev, d) => {
            if (!ev.active) sim.alphaTarget(0.3).restart();
            d.fx = d.x; d.fy = d.y;
        })
        .on('drag',  (ev, d) => { d.fx = ev.x; d.fy = ev.y; })
        .on('end',   (ev, d) => {
            if (!ev.active) sim.alphaTarget(0);
            d.fx = null; d.fy = null;
        });
}

// Tooltip
const tooltip = document.getElementById('tooltip');

function showTooltip(ev, d) {
    const vulns = state.vulnMap[d.id] || [];
    const vulnHint = vulns.length > 0 ? `<br><span style="color:#fc8181">⚠ ${vulns.length} vuln${vulns.length > 1 ? 's' : ''}</span>` : '';
    tooltip.innerHTML = `<strong>${d.name}</strong><br><span style="opacity:.7">${d.version || ''}</span>${vulnHint}`;
    tooltip.style.display = 'block';
    tooltip.style.left    = (ev.pageX + 14) + 'px';
    tooltip.style.top     = (ev.pageY - 36) + 'px';
}

function hideTooltip() { tooltip.style.display = 'none'; }

// Node detail sidebar
function showDetail(d) {
    state.selected = d.id;
    hideTooltip();

    const deps = state.allLinks
        .filter(l => resolveId(l.source) === d.id)
        .map(l => state.allNodes.find(n => n.id === resolveId(l.target)))
        .filter(Boolean);

    const dependents = state.allLinks
        .filter(l => resolveId(l.target) === d.id)
        .map(l => state.allNodes.find(n => n.id === resolveId(l.source)))
        .filter(Boolean);

    document.getElementById('detail-name').textContent    = d.name;
    document.getElementById('detail-version').textContent = d.version  || '-';
    document.getElementById('detail-kind').textContent    = d.kind     || '-';
    document.getElementById('detail-indirect').textContent =
        d.dev ? 'Dev' : (d.indirect ? 'Yes' : 'No');

    // License
    const licenseRow = document.getElementById('detail-license-row');
    if (state.audit && state.audit.licenses && state.audit.licenses[d.id]) {
        document.getElementById('detail-license').textContent = state.audit.licenses[d.id];
        licenseRow.classList.remove('hidden');
    } else {
        licenseRow.classList.add('hidden');
    }

    // Vulnerabilities for this node
    const vulns = state.vulnMap[d.id] || [];
    const vulnsSection = document.getElementById('detail-vulns-section');
    if (vulns.length > 0) {
        document.getElementById('detail-vulns-count').textContent = vulns.length;
        const list = document.getElementById('detail-vulns-list');
        list.innerHTML = vulns.map(v => `
            <div class="sidebar-vuln">
                <div>
                    <a class="sidebar-vuln-id" href="${v.link || '#'}" target="_blank" rel="noopener">${v.id}</a>
                    <span class="sev-badge sev-${v.severity || 'UNKNOWN'}" style="margin-left:6px">${v.severity || 'UNKNOWN'}</span>
                </div>
                <div class="sidebar-vuln-summary">${v.summary || ''}</div>
                ${v.fixed_in ? `<div style="font-size:.75rem;color:#065f46;margin-top:3px">Fix: ${v.fixed_in}</div>` : ''}
            </div>
        `).join('');
        vulnsSection.classList.remove('hidden');
    } else {
        vulnsSection.classList.add('hidden');
    }

    renderDepList('detail-deps',       deps);
    renderDepList('detail-dependents', dependents);

    document.getElementById('detail-deps-count').textContent       = deps.length;
    document.getElementById('detail-dependents-count').textContent = dependents.length;

    document.getElementById('sidebar').classList.remove('hidden');
}

function renderDepList(elId, nodes) {
    const el = document.getElementById(elId);
    if (nodes.length === 0) {
        el.innerHTML = '<span class="empty-note">none</span>';
        return;
    }
    el.innerHTML = nodes
        .map(n => `<span class="dep-badge" title="${n.id}">${n.name}${n.version ? '@' + n.version : ''}</span>`)
        .join('');
    // Click badge → navigate to that node's detail
    el.querySelectorAll('.dep-badge').forEach((badge, i) => {
        badge.addEventListener('click', () => showDetail(nodes[i]));
    });
}

function closeSidebar() {
    document.getElementById('sidebar').classList.add('hidden');
    state.selected = null;
}

// Table view
function renderTable(nodes) {
    const tbody     = document.getElementById('dep-table-body');
    const emptyEl   = document.getElementById('table-empty-state');

    if (nodes.length === 0) {
        tbody.innerHTML = '';
        emptyEl.classList.remove('hidden');
        return;
    }
    emptyEl.classList.add('hidden');

    const depCounts = {};
    state.allLinks.forEach(l => {
        const src = resolveId(l.source);
        depCounts[src] = (depCounts[src] || 0) + 1;
    });

    tbody.innerHTML = nodes.map(n => {
        const vulns  = state.vulnMap[n.id] || [];
        const deps   = depCounts[n.id] || 0;
        const { label: typeLabel, cls: typeClass } = nodeTypeInfo(n);
        const vulnCell = vulns.length > 0
            ? `<span class="vuln-count-badge">⚠ ${vulns.length}</span>`
            : `<span class="no-vuln">-</span>`;
        return `<tr class="dep-row" data-id="${escHtml(n.id)}">
            <td><span class="dep-name-cell">${escHtml(n.name)}</span></td>
            <td><span class="dep-version-cell">${escHtml(n.version || '-')}</span></td>
            <td><span class="type-badge ${typeClass}">${typeLabel}</span></td>
            <td class="dep-count-cell">${deps || '-'}</td>
            <td>${vulnCell}</td>
        </tr>`;
    }).join('');

    tbody.querySelectorAll('.dep-row').forEach(row => {
        row.addEventListener('click', () => {
            const node = state.allNodes.find(n => n.id === row.dataset.id);
            if (node) showDetail(node);
        });
    });
}

function nodeTypeInfo(n) {
    if (n.kind === 'main') return { label: 'Main',     cls: 'type-main' };
    if (n.dev)             return { label: 'Dev',      cls: 'type-dev' };
    if (n.indirect)        return { label: 'Indirect', cls: 'type-indirect' };
    return                        { label: 'Direct',   cls: 'type-direct' };
}

function escHtml(s) {
    return String(s)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

// Shared hierarchy builder (DFS tree subset of the DAG)
function buildHierarchy(visibleNodes) {
    if (!visibleNodes.length) return null;
    const visIds  = new Set(visibleNodes.map(n => n.id));
    const root    = visibleNodes.find(n => n.kind === 'main') || visibleNodes[0];
    const nodeMap = Object.fromEntries(state.allNodes.map(n => [n.id, n]));
    const adj     = {};
    state.allLinks.forEach(l => {
        const src = resolveId(l.source), tgt = resolveId(l.target);
        if (visIds.has(src) && visIds.has(tgt)) (adj[src] ??= []).push(tgt);
    });
    const visited = new Set();
    function dfs(id) {
        if (visited.has(id)) return null;
        visited.add(id);
        const node = nodeMap[id];
        if (!node || !visIds.has(id)) return null;
        const children = (adj[id] || []).map(dfs).filter(Boolean);
        return { ...node, children };
    }
    return dfs(root.id);
}

// Tree view
function renderTree(nodes) {
    const container = document.getElementById('tree-container');
    container.innerHTML = '';
    const emptyEl = document.getElementById('tree-empty-state');
    const data    = buildHierarchy(nodes);
    if (!data) { emptyEl.classList.remove('hidden'); return; }
    emptyEl.classList.add('hidden');

    const { width: W, height: H } = container.getBoundingClientRect();
    const svg = d3.select('#tree-container').append('svg')
        .attr('width', '100%').attr('height', '100%');

    const zoom = d3.zoom().scaleExtent([0.05, 4])
        .on('zoom', e => g.attr('transform', e.transform));
    svg.call(zoom);
    svg.on('click', () => { state.selected = null; closeSidebar(); });

    const g = svg.append('g');
    g.append('g').attr('class', 'tree-links');
    g.append('g').attr('class', 'tree-nodes');

    const root = d3.hierarchy(data, d => d.children?.length > 0 ? d.children : null);

    // Collapse beyond depth 1 initially
    root.descendants().forEach(d => {
        if (d.depth > 1 && d.children) { d._children = d.children; d.children = null; }
    });

    function update() {
        d3.tree().nodeSize([28, 220])(root);

        g.select('.tree-links').selectAll('path')
            .data(root.links(), d => `${d.source.data.id}→${d.target.data.id}`)
            .join(
                enter => enter.append('path')
                    .attr('fill', 'none').attr('stroke', '#c5cdd6').attr('stroke-width', 1.2)
                    .attr('opacity', 0)
                    .attr('d', d3.linkHorizontal().x(d => d.y).y(d => d.x))
                    .call(el => el.transition().duration(220).attr('opacity', 1)),
                upd => upd.transition().duration(220)
                    .attr('d', d3.linkHorizontal().x(d => d.y).y(d => d.x)),
                exit => exit.transition().duration(220).attr('opacity', 0).remove()
            );

        g.select('.tree-nodes').selectAll('g.tree-node')
            .data(root.descendants(), d => d.data.id)
            .join(
                enter => {
                    const ng = enter.append('g').attr('class', 'tree-node')
                        .attr('transform', d => `translate(${d.y},${d.x})`)
                        .attr('opacity', 0)
                        .call(el => el.transition().duration(220).attr('opacity', 1));

                    ng.append('circle')
                        .attr('r', d => d.data.kind === 'main' ? 9 : 6)
                        .attr('fill', d => nodeColor(d.data))
                        .attr('stroke', d => state.vulnMap[d.data.id]?.length > 0 ? '#e53e3e' : '#fff')
                        .attr('stroke-width', d => state.vulnMap[d.data.id]?.length > 0 ? 3 : 2)
                        .attr('cursor', 'pointer')
                        .on('click', (ev, d) => {
                            ev.stopPropagation();
                            if (d.children)       { d._children = d.children; d.children = null; }
                            else if (d._children) { d.children = d._children; d._children = null; }
                            update();
                            showDetail(d.data);
                        })
                        .on('mouseover', (ev, d) => showTooltip(ev, d.data))
                        .on('mouseout',  hideTooltip);

                    ng.append('text').attr('class', 'tree-collapse-indicator')
                        .attr('x', 0).attr('y', 0).attr('dy', '0.32em')
                        .attr('text-anchor', 'middle').attr('font-size', '8px')
                        .attr('fill', '#fff').attr('pointer-events', 'none');

                    ng.append('text').attr('class', 'tree-node-label')
                        .attr('dy', '0.32em').attr('font-size', '11px')
                        .attr('fill', '#2c3e50').attr('pointer-events', 'none');

                    return ng;
                },
                upd => upd.transition().duration(220)
                    .attr('transform', d => `translate(${d.y},${d.x})`),
                exit => exit.transition().duration(220).attr('opacity', 0).remove()
            );

        // Sync text on all current nodes
        g.select('.tree-nodes').selectAll('g.tree-node').select('.tree-node-label')
            .attr('x', d => d.data.kind === 'main' ? 15 : 12)
            .text(d => d.data.name);
        g.select('.tree-nodes').selectAll('g.tree-node').select('.tree-collapse-indicator')
            .text(d => d._children ? '+' : (d.children?.length > 0 ? '−' : ''));
    }

    update();
    svg.call(zoom.transform, d3.zoomIdentity.translate(60, H / 2));
}

// Treemap view
function renderTreemap(nodes) {
    const container = document.getElementById('treemap-container');
    container.innerHTML = '';
    const emptyEl = document.getElementById('treemap-empty-state');
    const data    = buildHierarchy(nodes);
    if (!data) { emptyEl.classList.remove('hidden'); return; }
    emptyEl.classList.add('hidden');

    const { width: W, height: H } = container.getBoundingClientRect();

    const root = d3.hierarchy(data, d => d.children?.length > 0 ? d.children : null)
        .sum(() => 1)
        .sort((a, b) => b.value - a.value);

    d3.treemap().size([W, H])
        .paddingOuter(3).paddingTop(18).paddingInner(2).round(true)(root);

    const svg = d3.select('#treemap-container').append('svg')
        .attr('width', W).attr('height', H);

    const cw = d => Math.max(0, d.x1 - d.x0);
    const ch = d => Math.max(0, d.y1 - d.y0);

    const cell = svg.selectAll('g.tm-cell')
        .data(root.descendants())
        .join('g').attr('class', 'tm-cell')
        .attr('transform', d => `translate(${d.x0},${d.y0})`);

    cell.append('rect')
        .attr('width',  cw).attr('height', ch)
        .attr('fill',         d => d.depth === 0 ? '#e3e8ef' : nodeColor(d.data))
        .attr('fill-opacity', d => d.depth === 0 ? 1 : (d.children ? 0.28 : 0.72))
        .attr('stroke', '#fff').attr('stroke-width', 1.5)
        .attr('cursor', d => d.depth > 0 ? 'pointer' : 'default')
        .on('click',     (ev, d) => { if (d.depth > 0) { ev.stopPropagation(); showDetail(d.data); } })
        .on('mouseover', (ev, d) => { if (d.depth > 0 && !d.children) showTooltip(ev, d.data); })
        .on('mouseout',  hideTooltip);

    // Label for parent tiles (in the paddingTop header)
    cell.filter(d => d.depth > 0 && d.children && cw(d) > 30)
        .append('text')
        .attr('x', 4).attr('y', 13)
        .attr('font-size', '10px').attr('font-weight', '600')
        .attr('fill', '#fff').attr('pointer-events', 'none')
        .text(d => {
            const max = Math.floor(cw(d) / 7);
            return d.data.name.length > max ? d.data.name.slice(0, max - 1) + '…' : d.data.name;
        });

    // Label for root tile
    cell.filter(d => d.depth === 0 && cw(d) > 40)
        .append('text')
        .attr('x', 6).attr('y', 14)
        .attr('font-size', '10px').attr('font-weight', '600')
        .attr('fill', '#636e72').attr('pointer-events', 'none')
        .text(d => d.data.name);

    // Label for leaf tiles
    cell.filter(d => !d.children && cw(d) > 40 && ch(d) > 16)
        .append('text')
        .attr('x', 4).attr('y', d => ch(d) / 2).attr('dy', '0.35em')
        .attr('font-size', '10px').attr('fill', '#fff').attr('pointer-events', 'none')
        .text(d => {
            const max = Math.floor(cw(d) / 6.5);
            return d.data.name.length > max ? d.data.name.slice(0, max - 1) + '…' : d.data.name;
        });

    // Small red dot for vulnerable leaf tiles
    cell.filter(d => !d.children && state.vulnMap[d.data.id]?.length > 0 && cw(d) > 12 && ch(d) > 12)
        .append('circle')
        .attr('cx', d => cw(d) - 6).attr('cy', 6).attr('r', 3.5)
        .attr('fill', '#e53e3e').attr('pointer-events', 'none');
}

// List view
function renderList(nodes) {
    const container = document.getElementById('list-container');
    container.innerHTML = '';

    const q = state.search;
    const groups = [
        { label: 'Main Module',           cls: 'type-main',     items: nodes.filter(n => n.kind === 'main') },
        { label: 'Direct Dependencies',   cls: 'type-direct',   items: nodes.filter(n => n.kind !== 'main' && !n.dev && !n.indirect) },
        { label: 'Indirect Dependencies', cls: 'type-indirect', items: nodes.filter(n => n.indirect && !n.dev) },
        { label: 'Dev Dependencies',      cls: 'type-dev',      items: nodes.filter(n => n.dev) },
    ];

    let anyVisible = false;
    groups.forEach(group => {
        const items = q
            ? group.items.filter(n => n.name.toLowerCase().includes(q))
            : group.items;
        if (!items.length) return;
        anyVisible = true;

        const section = document.createElement('div');
        section.className = 'list-group';
        section.innerHTML = `
            <div class="list-group-header">
                <span class="type-badge ${group.cls}">${group.label}</span>
                <span class="list-group-count">${items.length}</span>
            </div>
            <div class="list-group-items">
                ${items.map(n => {
                    const vulns = state.vulnMap[n.id] || [];
                    const vulnBadge = vulns.length > 0
                        ? `<span class="vuln-count-badge">⚠ ${vulns.length}</span>` : '';
                    return `<div class="list-item" data-id="${escHtml(n.id)}">
                        <span class="list-item-dot" style="background:${nodeColor(n)}"></span>
                        <span class="list-item-name">${escHtml(n.name)}</span>
                        <span class="list-item-version">${escHtml(n.version || '')}</span>
                        ${vulnBadge}
                    </div>`;
                }).join('')}
            </div>`;

        section.querySelectorAll('.list-item').forEach(row => {
            row.addEventListener('click', () => {
                const node = state.allNodes.find(n => n.id === row.dataset.id);
                if (node) showDetail(node);
            });
        });
        container.appendChild(section);
    });

    if (!anyVisible) {
        container.innerHTML = '<div class="list-empty">No modules found</div>';
    }
}

// Helpers
function nodeColor(d) {
    if (d.kind === 'main') return CFG.colors.main;
    if (d.dev || d.indirect) return CFG.colors.dev;
    return CFG.colors.module;
}

function resolveId(ref) {
    return typeof ref === 'object' ? ref.id : ref;
}

// Audit

async function openAuditPanel() {
    const overlay  = document.getElementById('audit-overlay');
    const loading  = document.getElementById('audit-loading');
    const results  = document.getElementById('audit-results');
    const errEl    = document.getElementById('audit-error');
    const btn      = document.getElementById('audit-btn');

    overlay.classList.remove('hidden');
    errEl.classList.add('hidden');
    results.classList.add('hidden');

    if (state.audit) {
        // Already loaded - just show results.
        renderAuditResults(state.audit);
        loading.classList.add('hidden');
        results.classList.remove('hidden');
        return;
    }

    loading.classList.remove('hidden');
    btn.classList.add('running');
    btn.textContent = 'Running…';

    try {
        const res = await fetch('/api/audit');
        if (!res.ok) {
            const msg = await res.text();
            throw new Error(msg || `Audit failed (${res.status})`);
        }
        const data = await res.json();
        applyAuditData(data);
        renderAuditResults(data);
        loading.classList.add('hidden');
        results.classList.remove('hidden');
    } catch (err) {
        loading.classList.add('hidden');
        errEl.textContent = err.message;
        errEl.classList.remove('hidden');
    } finally {
        btn.classList.remove('running');
        btn.innerHTML = `<svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg> Audit`;
    }
}

function closeAuditPanel() {
    document.getElementById('audit-overlay').classList.add('hidden');
}

// Apply audit data to state and update the graph visually.
function applyAuditData(data) {
    state.audit = data;
    state.vulnMap = {};
    (data.vulnerabilities || []).forEach(v => {
        (state.vulnMap[v.node_id] ??= []).push(v);
    });

    // Show legend entry and vuln-only filter when vulnerabilities are present.
    if (Object.keys(state.vulnMap).length > 0) {
        document.getElementById('legend-vuln').classList.remove('hidden');
        document.getElementById('filter-option-vuln').classList.remove('hidden');
        document.getElementById('filter-divider-vuln').classList.remove('hidden');
    }

    // Re-render to show vulnerability rings.
    if (nodeSel) render();
}

function renderAuditResults(data) {
    const vulns     = data.vulnerabilities  || [];
    const conflicts = data.conflicts        || [];
    const licenses  = data.licenses         || {};

    // --- Vulnerabilities ---
    document.getElementById('audit-vuln-count').textContent = vulns.length;
    const vulnList = document.getElementById('audit-vuln-list');
    if (vulns.length === 0) {
        vulnList.innerHTML = '<p class="audit-empty">No vulnerabilities found.</p>';
    } else {
        vulnList.innerHTML = vulns.map(v => `
            <div class="vuln-card">
                <div class="vuln-card-header">
                    <a class="vuln-id" href="${v.link || '#'}" target="_blank" rel="noopener">${v.id}</a>
                    <span class="sev-badge sev-${v.severity || 'UNKNOWN'}">${v.severity || 'UNKNOWN'}</span>
                </div>
                <div class="vuln-summary">${v.summary || '-'}</div>
                <div class="vuln-module">${v.node_id}</div>
                ${v.fixed_in ? `<div class="vuln-fix">Fix: upgrade to ${v.fixed_in}</div>` : ''}
            </div>
        `).join('');
    }

    // --- Conflicts ---
    document.getElementById('audit-conflict-count').textContent = conflicts.length;
    const conflictList = document.getElementById('audit-conflict-list');
    if (conflicts.length === 0) {
        conflictList.innerHTML = '<p class="audit-empty">No version conflicts detected.</p>';
    } else {
        conflictList.innerHTML = conflicts.map(c => `
            <div class="conflict-row">
                <span class="conflict-module" title="${c.module}">${c.module}</span>
                <span class="conflict-versions">${(c.versions || []).join(' · ')}</span>
            </div>
        `).join('');
    }

    // --- Licenses ---
    const licenseEntries = Object.entries(licenses);
    document.getElementById('audit-license-count').textContent = licenseEntries.length;
    const licenseList = document.getElementById('audit-license-list');
    if (licenseEntries.length === 0) {
        licenseList.innerHTML = '<p class="audit-empty">No license data available.</p>';
    } else {
        licenseList.innerHTML = licenseEntries.map(([nodeID, lic]) => `
            <div class="license-row">
                <span class="license-module" title="${nodeID}">${nodeID}</span>
                <span class="license-spdx">${lic}</span>
            </div>
        `).join('');
    }
}
