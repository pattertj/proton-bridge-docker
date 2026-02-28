const { useState, useEffect, useCallback, useRef } = React;

// --- Auth token management ---
function getToken() {
  return localStorage.getItem('webui_token') || '';
}
function setToken(token) {
  localStorage.setItem('webui_token', token);
}
function clearToken() {
  localStorage.removeItem('webui_token');
}

// --- API helpers ---
async function api(path, options = {}) {
  const token = getToken();
  const headers = { 'Content-Type': 'application/json' };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(`/api${path}`, { headers, ...options });
  if (res.status === 401) {
    throw new AuthError('unauthorized');
  }
  return res.json();
}

class AuthError extends Error {
  constructor(msg) { super(msg); this.name = 'AuthError'; }
}

// --- Copy to clipboard ---
function useCopy() {
  const [copied, setCopied] = useState(null);
  const copy = useCallback((text, id) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(id);
      setTimeout(() => setCopied(null), 2000);
    });
  }, []);
  return { copied, copy };
}

// --- Components ---

function StatusBadge({ state }) {
  const labels = {
    running: 'Bridge Running',
    setup: 'Setup Mode',
    stopped: 'Stopped',
    transitioning: 'Switching...',
  };
  return (
    <span className={`status-badge ${state}`}>
      <span className="status-dot" />
      {labels[state] || state}
    </span>
  );
}

function CredRow({ label, value, onCopy, copyId, copied }) {
  return (
    <div className="cred-row">
      <span className="cred-label">{label}</span>
      <span className="cred-value">{value || '—'}</span>
      {value && (
        <button
          className={`copy-btn ${copied === copyId ? 'copied' : ''}`}
          onClick={() => onCopy(value, copyId)}
        >
          {copied === copyId ? 'Copied' : 'Copy'}
        </button>
      )}
    </div>
  );
}

function LoginForm({ onLogin, loading, error }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = (e) => {
    e.preventDefault();
    onLogin(username, password);
  };

  return (
    <div className="card">
      <h2>Sign in to Proton Mail</h2>
      <p style={{ fontSize: '0.85rem', color: '#888', marginBottom: 16 }}>
        Enter your Proton account credentials to set up the Bridge.
      </p>
      {error && <div className="error-msg">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Email</label>
          <input
            type="email"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="you@proton.me"
            autoComplete="email"
            required
          />
        </div>
        <div className="form-group">
          <label>Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Your Proton password"
            autoComplete="current-password"
            required
          />
        </div>
        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading && <span className="spinner" />}
          {loading ? 'Connecting...' : 'Sign In'}
        </button>
      </form>
      <div className="callout" style={{ marginTop: 16 }}>
        Bridge requires a paid Proton plan (Mail Plus, Unlimited, or Business).
      </div>
    </div>
  );
}

function TwoFactorForm({ onSubmit, loading, error }) {
  const [code, setCode] = useState('');
  const inputRef = useRef(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(code);
  };

  return (
    <div className="card">
      <h2>Two-Factor Authentication</h2>
      <p style={{ fontSize: '0.85rem', color: '#888', marginBottom: 16 }}>
        Enter the 6-digit code from your authenticator app.
      </p>
      {error && <div className="error-msg">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>2FA Code</label>
          <input
            ref={inputRef}
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            maxLength={6}
            value={code}
            onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
            placeholder="000000"
            autoComplete="one-time-code"
            required
          />
        </div>
        <button type="submit" className="btn btn-primary" disabled={loading || code.length < 6}>
          {loading && <span className="spinner" />}
          {loading ? 'Verifying...' : 'Verify'}
        </button>
      </form>
    </div>
  );
}

function SyncingView({ progress }) {
  return (
    <div className="card">
      <h2>Syncing Mailbox</h2>
      <p style={{ fontSize: '0.85rem', color: '#888', marginBottom: 8 }}>
        {progress?.account || 'Your account'} is syncing...
      </p>
      <div className="progress-container">
        <div className="progress-bar">
          <div
            className="progress-fill"
            style={{ width: `${progress?.percentage || 0}%` }}
          />
        </div>
        <div className="progress-text">
          {(progress?.percentage || 0).toFixed(1)}%
        </div>
      </div>
      <div className="callout">
        Initial sync may take several minutes depending on mailbox size. IMAP/SMTP will be available once sync completes.
      </div>
    </div>
  );
}

function AccountCard({ account, onToggleMode, onLogout, copied, onCopy }) {
  return (
    <div className="card">
      <h3>{account.email || 'Account'}</h3>
      {account.status && (
        <div style={{ fontSize: '0.8rem', color: '#888', marginBottom: 12 }}>
          Status: {account.status}
        </div>
      )}

      <div className="section-label">IMAP Settings</div>
      <CredRow label="Host" value="Container IP / hostname" copyId={`${account.id}-imap-host`} copied={copied} onCopy={onCopy} />
      <CredRow label="Port" value={String(account.imapPort)} copyId={`${account.id}-imap-port`} copied={copied} onCopy={onCopy} />
      <CredRow label="Username" value={account.username} copyId={`${account.id}-imap-user`} copied={copied} onCopy={onCopy} />
      <CredRow label="Password" value={account.password} copyId={`${account.id}-imap-pass`} copied={copied} onCopy={onCopy} />
      <CredRow label="Security" value={account.security} copyId={`${account.id}-imap-sec`} copied={copied} onCopy={onCopy} />

      <div className="section-label">SMTP Settings</div>
      <CredRow label="Host" value="Container IP / hostname" copyId={`${account.id}-smtp-host`} copied={copied} onCopy={onCopy} />
      <CredRow label="Port" value={String(account.smtpPort)} copyId={`${account.id}-smtp-port`} copied={copied} onCopy={onCopy} />
      <CredRow label="Username" value={account.username} copyId={`${account.id}-smtp-user`} copied={copied} onCopy={onCopy} />
      <CredRow label="Password" value={account.password} copyId={`${account.id}-smtp-pass`} copied={copied} onCopy={onCopy} />
      <CredRow label="Security" value={account.security} copyId={`${account.id}-smtp-sec`} copied={copied} onCopy={onCopy} />

      <div className="callout warning">
        The password above is a Bridge-generated password, NOT your Proton account password.
        Use it in your email clients and apps.
      </div>

      {account.username && account.password && (
        <button
          className="btn btn-secondary btn-sm"
          style={{ marginTop: 8, width: '100%' }}
          onClick={() => {
            const settings = [
              `IMAP Host: Container IP / hostname`,
              `IMAP Port: ${account.imapPort}`,
              `SMTP Host: Container IP / hostname`,
              `SMTP Port: ${account.smtpPort}`,
              `Username: ${account.username}`,
              `Password: ${account.password}`,
              `Security: ${account.security}`,
            ].join('\n');
            onCopy(settings, `${account.id}-all`);
          }}
        >
          {copied === `${account.id}-all` ? 'Copied!' : 'Copy All Settings'}
        </button>
      )}

      <div className="account-actions">
        <button className="btn btn-secondary btn-sm" onClick={() => onToggleMode(account.id)}>
          Toggle Mode
        </button>
        <button className="btn btn-danger btn-sm" onClick={() => onLogout(account.id)}>
          Remove Account
        </button>
      </div>
    </div>
  );
}

function Dashboard({ accounts, status, onRefresh, onAddAccount, onToggleMode, onLogout }) {
  const { copied, copy } = useCopy();

  return (
    <div>
      {accounts.map((acc) => (
        <AccountCard
          key={acc.id}
          account={acc}
          onToggleMode={onToggleMode}
          onLogout={onLogout}
          copied={copied}
          onCopy={copy}
        />
      ))}

      <div className="callout">
        Use the host ports configured in your Docker/Unraid settings (not the internal ports shown by Bridge).
        The container IP or hostname on your Docker network should be used as the host.
      </div>

      <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
        <button className="btn btn-secondary" onClick={onRefresh}>
          Refresh Credentials
        </button>
        <button className="btn btn-primary" onClick={onAddAccount}>
          Add Account
        </button>
      </div>
    </div>
  );
}

function TokenForm({ onSubmit, error }) {
  const [token, setTokenVal] = useState('');

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(token);
  };

  return (
    <div className="card">
      <h2>Authentication Required</h2>
      <p style={{ fontSize: '0.85rem', color: '#888', marginBottom: 16 }}>
        Enter the access token configured in your WEBUI_AUTH_TOKEN environment variable.
      </p>
      {error && <div className="error-msg">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Access Token</label>
          <input
            type="password"
            value={token}
            onChange={(e) => setTokenVal(e.target.value)}
            placeholder="Your WEBUI_AUTH_TOKEN"
            autoComplete="off"
            required
          />
        </div>
        <button type="submit" className="btn btn-primary" disabled={!token}>
          Unlock
        </button>
      </form>
    </div>
  );
}

function ErrorView({ message, onRetry }) {
  return (
    <div className="card">
      <h2>Something went wrong</h2>
      <div className="error-msg" style={{ marginTop: 12 }}>{message}</div>
      <button className="btn btn-primary" style={{ marginTop: 12 }} onClick={onRetry}>
        Retry
      </button>
    </div>
  );
}

// --- Main App ---

function App() {
  const [view, setView] = useState('loading'); // loading, auth, login, twofa, syncing, dashboard, error
  const [status, setStatus] = useState(null);
  const [accounts, setAccounts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [appError, setAppError] = useState(null);
  const [authError, setAuthError] = useState(null);

  const pollRef = useRef(null);

  // Handle auth errors globally
  const handleAuthError = useCallback(() => {
    clearToken();
    setAuthError(null);
    setView('auth');
  }, []);

  // Fetch status (returns { data, authRequired })
  const fetchStatus = useCallback(async () => {
    try {
      const data = await api('/status');
      setStatus(data);
      return { data, authRequired: false };
    } catch (e) {
      if (e instanceof AuthError) { handleAuthError(); return { data: null, authRequired: true }; }
      console.error('Status fetch failed:', e);
      return { data: null, authRequired: false };
    }
  }, [handleAuthError]);

  // Fetch accounts
  const fetchAccounts = useCallback(async () => {
    try {
      const data = await api('/accounts');
      setAccounts(data.accounts || []);
      return data.accounts || [];
    } catch (e) {
      if (e instanceof AuthError) { handleAuthError(); return []; }
      console.error('Accounts fetch failed:', e);
      return [];
    }
  }, [handleAuthError]);

  // Initial load
  useEffect(() => {
    (async () => {
      const { data: s, authRequired } = await fetchStatus();
      if (!s) {
        if (authRequired) return; // auth view already set
        setView('error');
        setAppError('Could not connect to Bridge web UI server.');
        return;
      }

      if (s.setupComplete && s.accounts?.length > 0) {
        const stateAccounts = s.accounts.map((email, i) => ({
          id: String(i),
          email,
          imapPort: 1143,
          smtpPort: 1025,
          security: 'STARTTLS',
        }));
        setAccounts(stateAccounts);
        setView('dashboard');
      } else {
        setView('login');
      }
    })();
  }, []);

  // Poll status every 5s when on dashboard
  useEffect(() => {
    if (view === 'dashboard' || view === 'syncing') {
      pollRef.current = setInterval(async () => {
        const { data: s } = await fetchStatus();
        if (s) {
          // Check for sync progress
          const syncing = s.syncProgress && Object.values(s.syncProgress).some(p => p.syncing);
          if (syncing && view !== 'syncing') {
            setView('syncing');
          } else if (!syncing && view === 'syncing') {
            setView('dashboard');
          }
        }
      }, 5000);
      return () => clearInterval(pollRef.current);
    }
  }, [view, fetchStatus]);

  // Login handler
  const handleLogin = async (username, password) => {
    setLoading(true);
    setError(null);
    try {
      const result = await api('/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      });

      if (result.needs2fa) {
        setView('twofa');
      } else if (result.success) {
        setView('dashboard');
        fetchAccounts();
      } else if (result.error) {
        setError(result.error);
      } else {
        setError(result.message || 'Unexpected response');
      }
    } catch (e) {
      if (e instanceof AuthError) { handleAuthError(); return; }
      setError('Connection failed: ' + e.message);
    } finally {
      setLoading(false);
    }
  };

  // 2FA handler
  const handleTwoFactor = async (code) => {
    setLoading(true);
    setError(null);
    try {
      const result = await api('/login/2fa', {
        method: 'POST',
        body: JSON.stringify({ code }),
      });

      if (result.success) {
        setView('dashboard');
        fetchAccounts();
      } else if (result.error) {
        setError(result.error);
      } else {
        setError(result.message || 'Unexpected response');
      }
    } catch (e) {
      if (e instanceof AuthError) { handleAuthError(); return; }
      setError('Connection failed: ' + e.message);
    } finally {
      setLoading(false);
    }
  };

  // Token submit handler
  const handleTokenSubmit = async (token) => {
    setToken(token);
    setAuthError(null);
    try {
      const s = await api('/status');
      setStatus(s);
      if (s.setupComplete && s.accounts?.length > 0) {
        const stateAccounts = s.accounts.map((email, i) => ({
          id: String(i), email, imapPort: 1143, smtpPort: 1025, security: 'STARTTLS',
        }));
        setAccounts(stateAccounts);
        setView('dashboard');
      } else {
        setView('login');
      }
    } catch (e) {
      if (e instanceof AuthError) {
        clearToken();
        setAuthError('Invalid token. Check your WEBUI_AUTH_TOKEN environment variable.');
        return;
      }
      setView('error');
      setAppError('Could not connect to Bridge web UI server.');
    }
  };

  // Toggle mode
  const handleToggleMode = async (id) => {
    await api(`/accounts/${id}/mode`, { method: 'POST' });
    setTimeout(fetchAccounts, 3000);
  };

  // Logout
  const handleLogout = async (id) => {
    if (!confirm('Remove this account from Bridge?')) return;
    await api(`/logout/${id}`, { method: 'POST' });
    const { data: s } = await fetchStatus();
    if (!s?.accounts?.length) {
      setView('login');
      setAccounts([]);
    } else {
      setTimeout(fetchAccounts, 3000);
    }
  };

  return (
    <div className="app">
      <div className="header">
        <h1>Proton Mail Bridge</h1>
        <div className="subtitle">IMAP/SMTP Gateway</div>
        {status && <StatusBadge state={status.bridgeState} />}
      </div>

      {view === 'loading' && (
        <div className="card" style={{ textAlign: 'center', padding: 40 }}>
          <span className="spinner" style={{ width: 24, height: 24 }} />
          <div style={{ marginTop: 12, color: '#888' }}>Connecting to Bridge...</div>
        </div>
      )}

      {view === 'auth' && (
        <TokenForm onSubmit={handleTokenSubmit} error={authError} />
      )}

      {view === 'login' && (
        <LoginForm onLogin={handleLogin} loading={loading} error={error} />
      )}

      {view === 'twofa' && (
        <TwoFactorForm onSubmit={handleTwoFactor} loading={loading} error={error} />
      )}

      {view === 'syncing' && (
        <SyncingView
          progress={status?.syncProgress ? Object.values(status.syncProgress)[0] : null}
        />
      )}

      {view === 'dashboard' && (
        <Dashboard
          accounts={accounts}
          status={status}
          onRefresh={fetchAccounts}
          onAddAccount={() => { setError(null); setView('login'); }}
          onToggleMode={handleToggleMode}
          onLogout={handleLogout}
        />
      )}

      {view === 'error' && (
        <ErrorView
          message={appError || 'An error occurred'}
          onRetry={() => window.location.reload()}
        />
      )}
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
