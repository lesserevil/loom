// Bootstrap Project functionality

async function showBootstrapProjectModal() {
    const modalHTML = `
        <div class="modal" id="bootstrap-modal" role="dialog" aria-labelledby="bootstrap-title" aria-modal="true">
            <div class="modal-content" style="max-width:580px">
                <div class="modal-header">
                    <h2 id="bootstrap-title">New Project</h2>
                    <button type="button" class="modal-close" onclick="closeBootstrapModal()" aria-label="Close">&times;</button>
                </div>
                <div class="modal-body">

                    <!-- Step 1: form -->
                    <form id="bootstrap-form">
                        <div class="form-group">
                            <label for="bootstrap-name">Project Name *</label>
                            <input type="text" id="bootstrap-name" name="name" required
                                   placeholder="My Awesome Project" autocomplete="off" />
                        </div>

                        <div class="form-group">
                            <label for="bootstrap-github-url">GitHub Repository URL *</label>
                            <input type="text" id="bootstrap-github-url" name="github_url" required
                                   placeholder="https://github.com/username/repo" autocomplete="off" />
                            <small>The repository can be empty or already contain code.</small>
                        </div>

                        <div class="form-group">
                            <label for="bootstrap-branch">Branch</label>
                            <input type="text" id="bootstrap-branch" name="branch" value="main" required />
                        </div>

                        <div class="form-group">
                            <label for="bootstrap-description">Project Description *</label>
                            <textarea id="bootstrap-description" name="prd_text" rows="5" required
                                      placeholder="Describe your project in 2–6 sentences. The Product Manager agent will expand this into a full PRD and kick off the engineering chain.

Example: A task management web app for personal productivity. Users can create, edit, and complete tasks, filter by status, and sync across devices. Should be responsive and support email/password login."></textarea>
                            <small>The PM agent will flesh this into a full PRD, the EM writes an SRD, then the TPM creates all the initial beads.</small>
                        </div>

                        <div class="form-actions">
                            <button type="button" class="secondary" onclick="closeBootstrapModal()">Cancel</button>
                            <button type="submit" id="bootstrap-submit" class="primary">Create Project</button>
                        </div>
                    </form>

                    <!-- Step 2: progress -->
                    <div id="bootstrap-status" style="display:none; text-align:center; padding:1rem 0">
                        <div id="bootstrap-status-icon" style="font-size:2rem; margin-bottom:0.5rem">⏳</div>
                        <div id="bootstrap-status-text" style="font-weight:600; margin-bottom:0.5rem">Creating project…</div>
                        <div id="bootstrap-status-details"></div>
                    </div>

                    <!-- Step 3: success with deploy key -->
                    <div id="bootstrap-success" style="display:none">
                        <div style="text-align:center; margin-bottom:1.25rem">
                            <div style="font-size:2.5rem">✅</div>
                            <h3 style="margin:0.5rem 0 0.25rem">Project created!</h3>
                            <p id="bootstrap-success-subtitle" style="color:var(--text-muted); margin:0"></p>
                        </div>

                        <div id="bootstrap-deploy-key-section" style="display:none">
                            <div style="background:var(--warning-bg,#fffbeb); border:1px solid var(--warning-border,#f59e0b); border-radius:6px; padding:1rem; margin-bottom:1rem">
                                <p style="margin:0 0 0.75rem; font-weight:600">⚠️ Action required: add the deploy key to GitHub</p>
                                <p style="margin:0 0 0.75rem; font-size:0.875rem">Loom needs write access to your repository. Add the key below as a deploy key:</p>
                                <ol style="margin:0 0 0.75rem; padding-left:1.25rem; font-size:0.875rem; line-height:1.6">
                                    <li>Go to your repo on GitHub</li>
                                    <li>Click <strong>Settings → Deploy keys → Add deploy key</strong></li>
                                    <li>Paste the key below, check <strong>Allow write access</strong>, and save</li>
                                </ol>
                            </div>

                            <div style="position:relative; margin-bottom:1rem">
                                <label style="font-size:0.8rem; font-weight:600; text-transform:uppercase; letter-spacing:.05em; color:var(--text-muted)">SSH Public Key (deploy key)</label>
                                <textarea id="bootstrap-public-key" readonly rows="4"
                                    style="width:100%; margin-top:0.25rem; font-family:monospace; font-size:0.78rem; background:var(--code-bg,#f8f8f8); border:1px solid var(--border); border-radius:4px; padding:0.5rem; resize:none; box-sizing:border-box"></textarea>
                                <button type="button" onclick="copyBootstrapKey()" id="bootstrap-copy-btn"
                                    style="margin-top:0.4rem; width:100%">Copy Key</button>
                            </div>
                        </div>

                        <div id="bootstrap-success-bead" style="font-size:0.875rem; color:var(--text-muted); margin-bottom:1rem"></div>

                        <div style="text-align:center">
                            <button type="button" class="primary" onclick="closeBootstrapModal()">Done</button>
                        </div>
                    </div>

                </div>
            </div>
        </div>
    `;

    const existing = document.getElementById('bootstrap-modal');
    if (existing) existing.remove();

    document.body.insertAdjacentHTML('beforeend', modalHTML);

    document.getElementById('bootstrap-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await submitBootstrapForm(e.target);
    });

    setTimeout(() => {
        document.getElementById('bootstrap-modal').classList.add('show');
        document.getElementById('bootstrap-name').focus();
    }, 10);
}

function closeBootstrapModal() {
    const modal = document.getElementById('bootstrap-modal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => modal.remove(), 300);
    }
}

async function submitBootstrapForm(form) {
    const formData = new FormData(form);
    const description = (formData.get('prd_text') || '').trim();
    const name = (formData.get('name') || '').trim();
    const githubUrl = (formData.get('github_url') || '').trim();

    if (!name) { showToast('Project name is required', 'error'); return; }
    if (!githubUrl) { showToast('GitHub URL is required', 'error'); return; }
    if (!description) { showToast('Please provide a project description', 'error'); return; }

    form.style.display = 'none';
    const statusDiv = document.getElementById('bootstrap-status');
    statusDiv.style.display = 'block';

    const payload = {
        github_url: githubUrl,
        name: name,
        branch: formData.get('branch') || 'main',
        prd_text: description
    };

    try {
        document.getElementById('bootstrap-status-text').textContent = 'Creating project…';

        const response = await apiCall('/projects/bootstrap', {
            method: 'POST',
            body: JSON.stringify(payload)
        });

        statusDiv.style.display = 'none';
        const successDiv = document.getElementById('bootstrap-success');
        successDiv.style.display = 'block';

        document.getElementById('bootstrap-success-subtitle').textContent =
            `Project "${name}" (ID: ${response.project_id}) is ready.`;

        if (response.initial_bead_id) {
            document.getElementById('bootstrap-success-bead').textContent =
                `Initial bead ${response.initial_bead_id} created — the PM agent will expand your description into a full PRD, then kick off the engineering chain.`;
        }

        if (response.public_key) {
            const keySection = document.getElementById('bootstrap-deploy-key-section');
            keySection.style.display = 'block';
            document.getElementById('bootstrap-public-key').value = response.public_key;
        }

        if (typeof loadProjects === 'function') loadProjects();
        if (typeof render === 'function') render();

    } catch (error) {
        statusDiv.style.display = 'none';
        form.style.display = 'block';
        showToast('Bootstrap failed: ' + (error.message || 'unknown error'), 'error');
    }
}

function copyBootstrapKey() {
    const ta = document.getElementById('bootstrap-public-key');
    if (!ta) return;
    navigator.clipboard.writeText(ta.value).then(() => {
        const btn = document.getElementById('bootstrap-copy-btn');
        const orig = btn.textContent;
        btn.textContent = 'Copied!';
        setTimeout(() => { btn.textContent = orig; }, 2000);
    }).catch(() => {
        ta.select();
        document.execCommand('copy');
    });
}

function updateBootstrapStatus(text, icon) {
    const iconEl = document.getElementById('bootstrap-status-icon');
    const textEl = document.getElementById('bootstrap-status-text');
    if (iconEl) iconEl.textContent = icon;
    if (textEl) textEl.textContent = text;
}
