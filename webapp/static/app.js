/* ═══════════════════════════════════════════════════════════════
   Core Banking Dashboard — Application Logic
   ═══════════════════════════════════════════════════════════════ */

const API = '';  // same origin, proxied by Go server

// ─── State ──────────────────────────────────────────────────────
let accounts = [];
let currentView = 'dashboard';

// ─── Init ───────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
    setupNavigation();
    setupModals();
    setupForms();
    refresh();
});

// ─── Navigation ─────────────────────────────────────────────────
function setupNavigation() {
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', e => {
            e.preventDefault();
            const view = item.dataset.view;
            switchView(view);
        });
    });

    document.getElementById('btn-refresh').addEventListener('click', refresh);
}

function switchView(view) {
    currentView = view;

    // Update nav
    document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
    document.querySelector(`.nav-item[data-view="${view}"]`).classList.add('active');

    // Update views
    document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
    document.getElementById(`view-${view}`).classList.add('active');

    // Update header
    const titles = {
        dashboard:    ['Dashboard',    'Overview of your banking operations'],
        accounts:     ['Accounts',     'Manage all bank accounts'],
        transactions: ['Transactions', 'View ledger entries and history'],
    };
    document.getElementById('page-title').textContent = titles[view][0];
    document.getElementById('page-subtitle').textContent = titles[view][1];

    // Load data for the view
    if (view === 'transactions' && accounts.length > 0) {
        const filter = document.getElementById('txn-account-filter');
        if (filter.value) loadTransactions(filter.value);
    }
}

// ─── Data Loading ───────────────────────────────────────────────
async function refresh() {
    const btn = document.getElementById('btn-refresh');
    btn.disabled = true;
    btn.querySelector('svg').style.animation = 'spin 0.8s linear infinite';

    try {
        await loadAccounts();
        await checkHealth();
    } finally {
        btn.disabled = false;
        btn.querySelector('svg').style.animation = '';
    }
}

async function loadAccounts() {
    // The ledger doesn't have a "list all" endpoint, so we'll track
    // accounts client-side after creation. For initial load, we try
    // to get them from localStorage.
    const stored = localStorage.getItem('cb_accounts');
    if (stored) {
        const ids = JSON.parse(stored);
        const fetched = [];
        for (const id of ids) {
            try {
                const resp = await fetch(`${API}/api/accounts/${id}`);
                if (resp.ok) fetched.push(await resp.json());
            } catch (_) { /* account may have been deleted */ }
        }
        accounts = fetched;
    }
    renderAccounts();
    renderStats();
    populateSelects();
}

function storeAccountId(id) {
    const stored = JSON.parse(localStorage.getItem('cb_accounts') || '[]');
    if (!stored.includes(id)) {
        stored.push(id);
        localStorage.setItem('cb_accounts', JSON.stringify(stored));
    }
}

async function checkHealth() {
    const el = document.getElementById('stat-status');
    try {
        const resp = await fetch(`${API}/health`);
        if (resp.ok) {
            el.textContent = 'All Systems Go';
            el.className = 'stat-value stat-ok';
        } else {
            el.textContent = 'Degraded';
            el.style.color = 'var(--amber)';
        }
    } catch {
        el.textContent = 'Offline';
        el.style.color = 'var(--red)';
    }
}

async function loadTransactions(accountId) {
    const container = document.getElementById('transactions-list');
    if (!accountId) {
        container.innerHTML = '<div class="empty-state">Select an account to view transactions</div>';
        return;
    }
    container.innerHTML = '<div class="empty-state">Loading…</div>';

    try {
        const resp = await fetch(`${API}/api/ledger/entries/${accountId}`);
        const entries = await resp.json();
        if (!entries || entries.length === 0) {
            container.innerHTML = '<div class="empty-state">No transactions yet</div>';
            return;
        }
        renderTransactions(entries.reverse()); // newest first
    } catch {
        container.innerHTML = '<div class="empty-state">Failed to load transactions</div>';
    }
}

// ─── Rendering ──────────────────────────────────────────────────
function renderAccounts() {
    // Dashboard account list
    const list = document.getElementById('accounts-list');
    if (accounts.length === 0) {
        list.innerHTML = '<div class="empty-state">No accounts yet — create one!</div>';
    } else {
        list.innerHTML = accounts.map(a => `
            <div class="account-item">
                <div class="account-info">
                    <div class="account-owner">${esc(a.owner)}</div>
                    <div class="account-id">${a.id.substring(0, 8)}…</div>
                </div>
                <div class="account-balance">
                    <div class="amount">${formatMoney(a.balance)}</div>
                    <span class="currency">${esc(a.currency)}</span>
                </div>
            </div>
        `).join('');
    }

    // Accounts view table
    const tableBody = document.getElementById('accounts-table-body');
    if (accounts.length === 0) {
        tableBody.innerHTML = '<div class="empty-state">No accounts</div>';
    } else {
        tableBody.innerHTML = `
            <table class="accounts-table">
                <thead><tr>
                    <th>Owner</th><th>ID</th><th>Currency</th><th>Balance</th><th>Created</th>
                </tr></thead>
                <tbody>
                    ${accounts.map(a => `
                        <tr>
                            <td><strong>${esc(a.owner)}</strong></td>
                            <td class="mono">${a.id}</td>
                            <td>${esc(a.currency)}</td>
                            <td style="font-weight:700;color:var(--cyan)">${formatMoney(a.balance)}</td>
                            <td class="mono">${formatDate(a.created_at)}</td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    }
}

function renderStats() {
    document.getElementById('stat-accounts').textContent = accounts.length;
    const total = accounts.reduce((sum, a) => sum + (a.balance || 0), 0);
    document.getElementById('stat-balance').textContent = formatMoney(total) + ' EUR';
}

function renderTransactions(entries) {
    const container = document.getElementById('transactions-list');
    container.innerHTML = entries.map(e => {
        const isPositive = e.amount >= 0;
        const iconClass = e.type === 'DEPOSIT' ? 'deposit' : (isPositive ? 'transfer-in' : 'transfer-out');
        const icon = e.type === 'DEPOSIT' ? '💰' : (isPositive ? '📥' : '📤');

        return `
            <div class="txn-item">
                <div class="txn-icon ${iconClass}">${icon}</div>
                <div class="txn-details">
                    <div class="txn-desc">${esc(e.description || e.type)}</div>
                    <div class="txn-date">${formatDateTime(e.created_at)}</div>
                </div>
                <div>
                    <div class="txn-amount ${isPositive ? 'positive' : 'negative'}">
                        ${isPositive ? '+' : ''}${formatMoney(e.amount)}
                    </div>
                    <div class="txn-balance-after">Bal: ${formatMoney(e.balance)}</div>
                </div>
            </div>
        `;
    }).join('');
}

function populateSelects() {
    const options = accounts.map(a =>
        `<option value="${a.id}">${esc(a.owner)} (${formatMoney(a.balance)} ${a.currency})</option>`
    ).join('');

    document.getElementById('deposit-account').innerHTML = options || '<option value="">No accounts</option>';
    document.getElementById('transfer-from').innerHTML = options || '<option value="">No accounts</option>';
    document.getElementById('transfer-to').innerHTML = options || '<option value="">No accounts</option>';

    // Transactions filter
    const filterOpts = '<option value="">Select account…</option>' +
        accounts.map(a => `<option value="${a.id}">${esc(a.owner)}</option>`).join('');
    document.getElementById('txn-account-filter').innerHTML = filterOpts;
}

// ─── Modals ─────────────────────────────────────────────────────
function setupModals() {
    const modal = document.getElementById('modal-new-account');
    const open = () => modal.classList.add('active');
    const close = () => modal.classList.remove('active');

    document.getElementById('btn-new-account').addEventListener('click', open);
    document.getElementById('btn-new-account-2').addEventListener('click', open);
    document.getElementById('modal-close').addEventListener('click', close);
    modal.addEventListener('click', e => { if (e.target === modal) close(); });
}

// ─── Forms ──────────────────────────────────────────────────────
function setupForms() {
    // New Account
    document.getElementById('form-new-account').addEventListener('submit', async e => {
        e.preventDefault();
        const owner = document.getElementById('new-owner').value.trim();
        const currency = document.getElementById('new-currency').value;

        try {
            const resp = await fetch(`${API}/api/accounts`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ owner, currency }),
            });
            const data = await resp.json();
            if (resp.ok) {
                storeAccountId(data.id);
                toast(`Account created for ${owner}`, 'success');
                document.getElementById('modal-new-account').classList.remove('active');
                document.getElementById('form-new-account').reset();
                await refresh();
            } else {
                toast(data.error || 'Failed to create account', 'error');
            }
        } catch {
            toast('Network error', 'error');
        }
    });

    // Deposit
    document.getElementById('form-deposit').addEventListener('submit', async e => {
        e.preventDefault();
        const accountId = document.getElementById('deposit-account').value;
        const amount = parseFloat(document.getElementById('deposit-amount').value);

        try {
            const resp = await fetch(`${API}/api/process/deposit`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ account_id: accountId, amount }),
            });
            const data = await resp.json();
            if (resp.ok && data.status === 'COMPLETED') {
                toast(`Deposited ${formatMoney(amount)} successfully`, 'success');
                document.getElementById('deposit-amount').value = '';
                await refresh();
            } else {
                toast(data.message || data.error || 'Deposit failed', 'error');
            }
        } catch {
            toast('Network error', 'error');
        }
    });

    // Transfer
    document.getElementById('form-transfer').addEventListener('submit', async e => {
        e.preventDefault();
        const from = document.getElementById('transfer-from').value;
        const to = document.getElementById('transfer-to').value;
        const amount = parseFloat(document.getElementById('transfer-amount').value);

        if (from === to) {
            toast('Cannot transfer to the same account', 'error');
            return;
        }

        try {
            const resp = await fetch(`${API}/api/process/transfer`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ from_account_id: from, to_account_id: to, amount }),
            });
            const data = await resp.json();
            if (resp.ok && data.status === 'COMPLETED') {
                toast(`Transferred ${formatMoney(amount)} successfully`, 'success');
                document.getElementById('transfer-amount').value = '';
                await refresh();
            } else {
                toast(data.message || data.error || 'Transfer failed', 'error');
            }
        } catch {
            toast('Network error', 'error');
        }
    });

    // Transaction filter
    document.getElementById('txn-account-filter').addEventListener('change', e => {
        loadTransactions(e.target.value);
    });
}

// ─── Toast ──────────────────────────────────────────────────────
function toast(message, type = 'success') {
    const container = document.getElementById('toast-container');
    const el = document.createElement('div');
    el.className = `toast toast-${type}`;
    el.textContent = message;
    container.appendChild(el);
    setTimeout(() => el.remove(), 4000);
}

// ─── Helpers ────────────────────────────────────────────────────
function formatMoney(amount) {
    return new Intl.NumberFormat('en-EU', {
        style: 'currency',
        currency: 'EUR',
        minimumFractionDigits: 2,
    }).format(amount);
}

function formatDate(iso) {
    return new Date(iso).toLocaleDateString('en-GB', {
        day: '2-digit', month: 'short', year: 'numeric'
    });
}

function formatDateTime(iso) {
    return new Date(iso).toLocaleString('en-GB', {
        day: '2-digit', month: 'short', year: 'numeric',
        hour: '2-digit', minute: '2-digit',
    });
}

function esc(str) {
    const d = document.createElement('div');
    d.textContent = str;
    return d.innerHTML;
}

// Add spin animation for refresh button
const styleSheet = document.createElement('style');
styleSheet.textContent = `@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`;
document.head.appendChild(styleSheet);
