/**
 * Conversations View — Cytoscape action-flow graph
 *
 * Each conversation is rendered as a left-to-right DAG:
 *   [System] → [Action 1] → [Action 2] → ... → [Done]
 *
 * Nodes are colored by outcome:
 *   green  = action executed successfully
 *   red    = error
 *   amber  = text/parse failure
 *   blue   = system prompt
 *   purple = terminal (done, git_commit, git_push)
 *
 * Clicking a node shows action details + notes + result in the side panel.
 */

/* global cytoscape, apiCall, escapeHtml, state, uiState, LoomCharts, formatAgentDisplayName */

let convCy = null;
let convData = null; // {conversations: [...], selected: conversation object}

const CONV_COLORS = {
    system: '#2563eb',
    success: '#16a34a',
    error: '#dc2626',
    text: '#d97706',
    terminal: '#7c3aed'
};

const TERMINAL_ACTIONS = new Set(['done', 'git_commit', 'git_push', 'create_pr']);

// ── Bootstrap ───────────────────────────────────────────────────────

async function renderConversationsView() {
    const container = document.getElementById('conversations-container');
    if (!container) return;

    // Populate project dropdown
    const projSelect = document.getElementById('conv-project-select');
    if (projSelect && state.projects) {
        const currentVal = projSelect.value;
        projSelect.innerHTML = state.projects.map(function (p) {
            return '<option value="' + escapeHtml(p.id) + '"' +
                (p.id === currentVal ? ' selected' : '') +
                '>' + escapeHtml(p.name || p.id) + '</option>';
        }).join('');

        if (!projSelect.dataset.bound) {
            projSelect.addEventListener('change', function () { loadConversationsList(); });
            projSelect.dataset.bound = '1';
        }
    }

    // Wire reset button
    var resetBtn = document.getElementById('conv-reset-view');
    if (resetBtn && !resetBtn.dataset.bound) {
        resetBtn.addEventListener('click', function () { if (convCy) convCy.fit(undefined, 30); });
        resetBtn.dataset.bound = '1';
    }

    // Wire conversation selector
    var convSelect = document.getElementById('conv-select');
    if (convSelect && !convSelect.dataset.bound) {
        convSelect.addEventListener('change', function () {
            var sid = convSelect.value;
            if (sid && convData && convData.conversations) {
                var c = convData.conversations.find(function (x) { return x.session_id === sid; });
                if (c) renderConversationGraph(c);
            }
        });
        convSelect.dataset.bound = '1';
    }

    await loadConversationsList();
}

async function loadConversationsList() {
    var projSelect = document.getElementById('conv-project-select');
    var projectId = projSelect ? projSelect.value : '';
    if (!projectId && state.projects && state.projects.length) {
        projectId = state.projects[0].id;
    }
    if (!projectId) return;

    try {
        var result = await apiCall('/conversations?project_id=' + encodeURIComponent(projectId) + '&limit=50', { suppressToast: true, skipAutoFile: true });
        var conversations = Array.isArray(result) ? result : (result && result.conversations) || [];

        // Sort by updated_at descending, then filter out empty
        conversations = conversations
            .filter(function (c) { return c.messages && c.messages.length > 0; })
            .sort(function (a, b) { return (b.updated_at || '').localeCompare(a.updated_at || ''); });

        convData = { conversations: conversations };

        // Populate conversation dropdown
        var convSelect = document.getElementById('conv-select');
        if (convSelect) {
            convSelect.innerHTML = conversations.map(function (c) {
                var agent = (c.metadata && c.metadata.agent_name) || '';
                var agentLabel = agent ? formatAgentDisplayName(agent) : '';
                var label = c.bead_id || c.session_id.substring(0, 8);
                if (agentLabel) label = agentLabel + ' — ' + label;
                label += ' (' + c.messages.length + ' msgs)';
                return '<option value="' + escapeHtml(c.session_id) + '">' + escapeHtml(label) + '</option>';
            }).join('');
        }

        if (conversations.length > 0) {
            renderConversationGraph(conversations[0]);
        } else {
            var graphEl = document.getElementById('conv-graph');
            if (graphEl) graphEl.innerHTML = '<div class="empty-state" style="padding:3rem;"><p>No conversations with messages</p></div>';
        }
    } catch (err) {
        console.error('[Conversations] Failed to load:', err);
    }
}

// ── Graph Rendering ─────────────────────────────────────────────────

function renderConversationGraph(conversation) {
    var graphEl = document.getElementById('conv-graph');
    if (!graphEl || !conversation || !conversation.messages) return;

    // Parse messages into nodes + edges
    var nodes = [];
    var edges = [];
    var stepIndex = 0;

    var msgs = conversation.messages;

    for (var i = 0; i < msgs.length; i++) {
        var msg = msgs[i];
        if (msg.role === 'system') {
            nodes.push({
                data: {
                    id: 'n-' + i,
                    label: 'System\nPrompt',
                    nodeType: 'system',
                    color: CONV_COLORS.system,
                    shape: 'diamond',
                    msgIndex: i,
                    msg: msg
                }
            });
            continue;
        }

        if (msg.role === 'assistant') {
            var parsed = parseAssistantMessage(msg.content);
            stepIndex++;

            var color = CONV_COLORS.success;
            var shape = 'round-rectangle';
            if (parsed.isText) {
                color = CONV_COLORS.text;
                shape = 'round-triangle';
            } else if (TERMINAL_ACTIONS.has(parsed.action)) {
                color = CONV_COLORS.terminal;
                shape = 'star';
            }

            // Check if next user message indicates error
            if (i + 1 < msgs.length && msgs[i + 1].role === 'user') {
                var resultContent = msgs[i + 1].content || '';
                if (resultContent.includes('error') || resultContent.includes('— error')) {
                    color = CONV_COLORS.error;
                }
            }

            var label = parsed.action || 'text';
            if (parsed.path) label += '\n' + truncPath(parsed.path);
            label = '#' + stepIndex + ' ' + label;

            nodes.push({
                data: {
                    id: 'n-' + i,
                    label: label,
                    nodeType: parsed.isText ? 'text' : 'action',
                    action: parsed.action,
                    path: parsed.path,
                    notes: parsed.notes,
                    color: color,
                    shape: shape,
                    msgIndex: i,
                    msg: msg,
                    resultMsg: (i + 1 < msgs.length && msgs[i + 1].role === 'user') ? msgs[i + 1] : null,
                    step: stepIndex
                }
            });

            // Edge from previous node
            if (nodes.length >= 2) {
                var prev = nodes[nodes.length - 2];
                edges.push({
                    data: {
                        id: 'e-' + prev.data.id + '-n-' + i,
                        source: prev.data.id,
                        target: 'n-' + i
                    }
                });
            }
        }
        // Skip 'user' messages as separate nodes; they are result data attached to the preceding assistant node
    }

    // Destroy previous instance
    if (convCy) {
        convCy.destroy();
        convCy = null;
    }

    // Ensure dagre is registered
    if (typeof cytoscape !== 'undefined' && typeof cytoscapeDagre !== 'undefined' && !cytoscape._dagreRegistered) {
        cytoscape.use(cytoscapeDagre);
        cytoscape._dagreRegistered = true;
    }

    try {
        convCy = cytoscape({
            container: graphEl,
            elements: { nodes: nodes, edges: edges },
            layout: {
                name: typeof dagre !== 'undefined' ? 'dagre' : 'breadthfirst',
                rankDir: 'LR',
                nodeSep: 30,
                rankSep: 60,
                edgeSep: 20,
                padding: 30,
                animate: true,
                animationDuration: 500
            },
            style: [
                {
                    selector: 'node',
                    style: {
                        'background-color': 'data(color)',
                        'label': 'data(label)',
                        'text-valign': 'center',
                        'text-halign': 'center',
                        'font-size': '9px',
                        'font-family': '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                        'color': '#fff',
                        'text-wrap': 'wrap',
                        'text-max-width': '90px',
                        'width': 60,
                        'height': 40,
                        'shape': 'data(shape)',
                        'border-width': 0,
                        'text-outline-width': 1,
                        'text-outline-color': 'data(color)',
                        'transition-property': 'border-width, border-color, width, height',
                        'transition-duration': '0.15s'
                    }
                },
                {
                    selector: 'node[nodeType="system"]',
                    style: {
                        'width': 50,
                        'height': 50,
                        'font-size': '8px'
                    }
                },
                {
                    selector: 'node[shape="star"]',
                    style: {
                        'width': 55,
                        'height': 55,
                        'font-size': '8px'
                    }
                },
                {
                    selector: 'node:selected',
                    style: {
                        'border-width': 3,
                        'border-color': '#1e293b',
                        'width': 70,
                        'height': 48
                    }
                },
                {
                    selector: 'edge',
                    style: {
                        'width': 2,
                        'line-color': '#cbd5e1',
                        'target-arrow-color': '#94a3b8',
                        'target-arrow-shape': 'triangle',
                        'curve-style': 'bezier',
                        'arrow-scale': 0.8
                    }
                }
            ],
            minZoom: 0.2,
            maxZoom: 3,
            wheelSensitivity: 0.3,
            userZoomingEnabled: true,
            userPanningEnabled: true
        });

        // Node click → show details
        convCy.on('tap', 'node', function (evt) {
            var data = evt.target.data();
            convCy.elements().unselect();
            evt.target.select();
            showConvNodeDetail(data, conversation);
        });

        // Background click → clear selection
        convCy.on('tap', function (evt) {
            if (evt.target === convCy) {
                convCy.elements().unselect();
                var panel = document.getElementById('conv-detail');
                if (panel) panel.innerHTML = '<div class="empty-state"><p>Select a node in the graph</p></div>';
            }
        });

        // Fit to viewport
        convCy.fit(undefined, 30);

    } catch (err) {
        console.error('[Conversations] Cytoscape error:', err);
        graphEl.innerHTML = '<div class="empty-state"><p>Failed to render graph: ' + escapeHtml(err.message) + '</p></div>';
    }

    // Clear detail panel
    var panel = document.getElementById('conv-detail');
    if (panel) {
        var agentName = (conversation.metadata && conversation.metadata.agent_name) || '';
        panel.innerHTML =
            '<div class="detail-section">' +
                '<h4>' + escapeHtml(formatAgentDisplayName(agentName) || conversation.bead_id || 'Conversation') + '</h4>' +
                '<div class="small">' +
                    '<strong>Bead:</strong> ' + escapeHtml(conversation.bead_id || 'n/a') + '<br>' +
                    '<strong>Messages:</strong> ' + conversation.messages.length + '<br>' +
                    '<strong>Actions:</strong> ' + nodes.filter(function (n) { return n.data.nodeType === 'action'; }).length + '<br>' +
                    '<strong>Errors:</strong> ' + nodes.filter(function (n) { return n.data.color === CONV_COLORS.error; }).length +
                '</div>' +
            '</div>' +
            '<p class="small" style="color:var(--text-muted);">Click a node to inspect its action and result.</p>';
    }
}

// ── Detail Panel ────────────────────────────────────────────────────

function showConvNodeDetail(data, conversation) {
    var panel = document.getElementById('conv-detail');
    if (!panel) return;

    if (data.nodeType === 'system') {
        panel.innerHTML =
            '<div class="detail-section">' +
                '<h4>System Prompt</h4>' +
                '<div class="small">' + escapeHtml((data.msg.timestamp || '').substring(0, 19).replace('T', ' ')) + '</div>' +
            '</div>' +
            '<pre>' + escapeHtml((data.msg.content || '').substring(0, 2000)) + (data.msg.content.length > 2000 ? '\n... (truncated)' : '') + '</pre>';
        return;
    }

    var html = '<div class="detail-section">';
    html += '<h4>Step #' + (data.step || '?') + ' — ' + escapeHtml(data.action || 'text') + '</h4>';
    html += '<div class="small">' + escapeHtml((data.msg.timestamp || '').substring(0, 19).replace('T', ' ')) + '</div>';
    if (data.path) html += '<div class="small"><strong>Path:</strong> ' + escapeHtml(data.path) + '</div>';
    html += '</div>';

    // Notes / reasoning
    if (data.notes) {
        html += '<div class="detail-section">';
        html += '<h4>Reasoning</h4>';
        html += '<div class="small" style="line-height:1.5;">' + escapeHtml(data.notes) + '</div>';
        html += '</div>';
    }

    // Full assistant JSON
    html += '<div class="detail-section">';
    html += '<h4>Action</h4>';
    var actionJson = data.msg.content || '';
    try {
        actionJson = JSON.stringify(JSON.parse(actionJson), null, 2);
    } catch (e) {
        // Not JSON; show as-is
    }
    html += '<pre>' + escapeHtml(actionJson.substring(0, 3000)) + '</pre>';
    html += '</div>';

    // Result from following user message
    if (data.resultMsg) {
        html += '<div class="detail-section">';
        html += '<h4>Result</h4>';
        var resultText = data.resultMsg.content || '';
        // Truncate long results
        if (resultText.length > 3000) resultText = resultText.substring(0, 3000) + '\n... (truncated)';
        html += '<pre>' + escapeHtml(resultText) + '</pre>';
        html += '</div>';
    }

    panel.innerHTML = html;
}

// ── Helpers ─────────────────────────────────────────────────────────

function parseAssistantMessage(content) {
    try {
        var j = JSON.parse(content);
        return {
            action: j.action || '',
            path: j.path || j.file || '',
            notes: j.notes || '',
            query: j.query || '',
            message: j.message || j.reason || '',
            isText: false,
            raw: j
        };
    } catch (e) {
        return {
            action: '',
            path: '',
            notes: '',
            query: '',
            message: content.substring(0, 200),
            isText: true,
            raw: null
        };
    }
}

function truncPath(p) {
    if (!p) return '';
    if (p.length <= 20) return p;
    var parts = p.split('/');
    if (parts.length <= 2) return p.substring(p.length - 20);
    return '.../' + parts[parts.length - 1];
}
