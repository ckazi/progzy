import { useState } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { getUser, clearAuth, isAdmin } from '../utils/auth';

function Layout({ children }) {
  const navigate = useNavigate();
  const location = useLocation();
  const user = getUser();
  const [menuOpen, setMenuOpen] = useState(false);

  const handleLogout = () => {
    clearAuth();
    navigate('/login');
  };

  const handleEditProfile = () => {
    navigate('/profile');
  };

  const isActive = (path) => location.pathname === path ? 'active' : '';

  return (
    <div>
      <header
        className="header"
        style={{
          background: '#0b1a2a',
          borderBottom: '1px solid rgba(255,255,255,0.1)',
        }}
      >
        <Link
          to="/dashboard"
          className="button button-secondary"
          style={{ fontSize: '20px', fontWeight: '600', background: '#24313f', border: 'none', textDecoration: 'none' }}
        >
          Progzy UI
        </Link>
        <div style={{ display: 'flex', alignItems: 'center', gap: '20px' }}>
          <nav className="nav">
            <Link to="/dashboard" className={isActive('/dashboard')}>Dashboard</Link>
            {isAdmin() && (
              <>
                <Link to="/users" className={isActive('/users')}>Users</Link>
                <Link to="/audit" className={isActive('/audit')}>Audit</Link>
              </>
            )}
            <Link to="/logs" className={isActive('/logs')}>Logs</Link>
            {isAdmin() && <Link to="/settings" className={isActive('/settings')}>Settings</Link>}
          </nav>
          <div style={{ position: 'relative' }}>
            <button
              onClick={() => setMenuOpen((prev) => !prev)}
              className="button button-secondary"
              style={{ background: '#369ad8', border: 'none' }}
            >
              {user?.username}
            </button>
            {menuOpen && (
              <div
                style={{
                  position: 'absolute',
                  right: 0,
                  top: 'calc(100% + 8px)',
                  background: '#10253c',
                  border: '1px solid rgba(255,255,255,0.1)',
                  borderRadius: '6px',
                  minWidth: '160px',
                  boxShadow: '0 10px 25px rgba(0,0,0,0.3)',
                  zIndex: 100,
                }}
              >
                <button
                  onClick={handleEditProfile}
                  className="button"
                  style={{ width: '100%', textAlign: 'left', background: 'transparent', border: 'none', color: '#fff', padding: '10px 14px', borderBottom: '1px solid rgba(255,255,255,0.1)' }}
                >
                  Profile & Security
                </button>
                <button
                  onClick={handleLogout}
                  className="button"
                  style={{ width: '100%', textAlign: 'left', background: 'transparent', border: 'none', color: '#fff', padding: '10px 14px' }}
                >
                  Logout
                </button>
              </div>
            )}
          </div>
        </div>
      </header>
      <div className="container">
        {children}
      </div>
      <footer
        style={{
          marginTop: '32px',
          padding: '24px',
          background: '#0b1a2b',
          color: '#fff',
          textAlign: 'center',
          borderTop: '1px solid rgba(255,255,255,0.1)',
        }}
      >
        <div style={{ fontSize: '18px', marginBottom: '12px', fontWeight: 600 }}>
          Are you enjoying this project?{' '}
          <span role="img" aria-label="beer">
            🍻
          </span>{' '}
          Buy me a beer!
        </div>
        <div
          style={{
            fontFamily: 'monospace',
            textAlign: 'center',
            maxWidth: '960px',
            margin: '0 auto',
            background: 'rgba(255,255,255,0.05)',
            padding: '12px',
            borderRadius: '6px',
            lineHeight: '1.8',
            display: 'inline-flex',
            flexDirection: 'column',
            gap: '8px',
          }}
        >
          <span>btc: 1A7MMHinNsscFZrCo4TpoRumHHsGgHrgAc</span>
          <span>ton: UQBjfsrp9ChrhRG441atYvhfdkqWStS47YLSifm9TYw1VHYM</span>
          <span>doge: DGuL61gT5ZzEUyfDLmPhHvJsnjxbgjZ4Um</span>
          <span>monero: 41ugNNZ5erdfj8ofHFhkb2gtwnpsB25digy6DWP1kCgRTJVbg6p7E6YMWbza7iCSMWaeuk9Qkeqzya8mCQcQDymH7P2tgZ5</span>
        </div>
        <div style={{ marginTop: '8px' }}>
          <span role="img" aria-label="cheers">
            🍻🍻🍻
          </span>
        </div>
      </footer>
    </div>
  );
}

export default Layout;
