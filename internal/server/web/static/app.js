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
        main:    '#e74c3c',
        module:  '#2980b9',
        dev:     '#7f8c8d',
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
    showDev:   true,
    selected:  null,
};

// D3 selections / simulation
let svgEl, gEl, simulation;
let linkSel, nodeSel, labelSel;

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

        document.getElementById('loading').classList.add('hidden');
        document.getElementById('app').classList.remove('hidden');

        applyMeta(data);
        setupSVG();
        setupControls();
        render();
    } catch (err) {
        document.getElementById('loading').classList.add('hidden');
        const banner = document.getElementById('error-banner');
        banner.querySelector('.error-msg').textContent = err.message;
        banner.classList.remove('hidden');
    }
}

function applyMeta(data) {
    const meta = data.metadata || {};
    const total = state.allNodes.length;
    const edges = state.allLinks.length;

    document.getElementById('stat-nodes').textContent = `${total} node${total !== 1 ? 's' : ''}`;
    document.getElementById('stat-edges').textContent = `${edges} edge${edges !== 1 ? 's' : ''}`;

    const name = meta.project_name || '';
    const lang = meta.language || '';

    if (name) {
        document.getElementById('project-name').textContent = name;
        document.title = `MDV — ${name}`;
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
        updateHighlight();
    });

    const slider    = document.getElementById('depth-slider');
    const depthLbl  = document.getElementById('depth-value');
    slider.addEventListener('input', () => {
        const v = parseInt(slider.value, 10);
        state.maxDepth = (v >= parseInt(slider.max, 10)) ? Infinity : v;
        depthLbl.textContent = state.maxDepth === Infinity ? '∞' : String(state.maxDepth);
        render();
    });

    document.getElementById('show-dev').addEventListener('change', function () {
        state.showDev = this.checked;
        render();
    });

    document.getElementById('sidebar-close').addEventListener('click', closeSidebar);
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

function getVisible() {
    const depthSet  = bfsFromRoot(state.maxDepth);
    const visNodes  = state.allNodes.filter(n =>
        depthSet.has(n.id) && (state.showDev || !n.indirect)
    );
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
    });

    updateHighlight();
}

// Highlight (search)
function updateHighlight() {
    if (!nodeSel) return;
    const q = state.search;

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
        const match = sn.toLowerCase().includes(q) || tn.toLowerCase().includes(q);
        return match ? 0.7 : 0.05;
    });
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
    tooltip.innerHTML = `<strong>${d.name}</strong><br><span style="opacity:.7">${d.version || ''}</span>`;
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
    document.getElementById('detail-version').textContent = d.version  || '—';
    document.getElementById('detail-kind').textContent    = d.kind     || '—';
    document.getElementById('detail-indirect').textContent = d.indirect ? 'Yes' : 'No';

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

// Helpers
function nodeColor(d) {
    if (d.kind === 'main') return CFG.colors.main;
    if (d.indirect)        return CFG.colors.dev;
    return CFG.colors.module;
}

function resolveId(ref) {
    return typeof ref === 'object' ? ref.id : ref;
}
