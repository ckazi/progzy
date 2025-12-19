import { useEffect, useState } from 'react';
import Layout from '../components/Layout';
import TwoFactorSetup from '../components/TwoFactorSetup';
import { usersAPI } from '../services/api';
import { getUser } from '../utils/auth';

function Profile() {
  const [user, setUser] = useState(getUser());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const loadProfile = async () => {
      const current = getUser();
      if (!current?.id) {
        setLoading(false);
        return;
      }
      try {
        const response = await usersAPI.getById(current.id);
        setUser(response.data);
        localStorage.setItem('user', JSON.stringify(response.data));
      } catch (err) {
        setError('Failed to load user profile.');
      } finally {
        setLoading(false);
      }
    };
    loadProfile();
  }, []);

  const handleStatusChange = (enabled) => {
    setUser((prev) => ({ ...prev, twofa_enabled: enabled }));
    const stored = getUser() || {};
    localStorage.setItem('user', JSON.stringify({ ...stored, twofa_enabled: enabled }));
  };

  if (loading) {
    return (
      <Layout>
        <div className="loading">Loading profile...</div>
      </Layout>
    );
  }

  return (
    <Layout>
      <h2 style={{ marginBottom: '16px' }}>Security Profile</h2>
      {error && <div className="error">{error}</div>}
      <div className="card" style={{ marginBottom: '16px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', flexWrap: 'wrap', gap: '12px' }}>
          <div>
            <div style={{ fontSize: '14px', color: '#9fb0c7' }}>Username</div>
            <div style={{ fontSize: '18px' }}>{user?.username}</div>
          </div>
          <div>
            <div style={{ fontSize: '14px', color: '#9fb0c7' }}>Email</div>
            <div style={{ fontSize: '18px' }}>{user?.email || 'not set'}</div>
          </div>
          <div>
            <div style={{ fontSize: '14px', color: '#9fb0c7' }}>Role</div>
            <div style={{ fontSize: '18px' }}>{user?.is_admin ? 'Administrator' : 'User'}</div>
          </div>
        </div>
      </div>
      <div className="card">
        <TwoFactorSetup initialEnabled={user?.twofa_enabled} onStatusChange={handleStatusChange} />
      </div>
    </Layout>
  );
}

export default Profile;
