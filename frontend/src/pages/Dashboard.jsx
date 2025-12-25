import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import { statsAPI, systemAPI } from '../services/api';

function Dashboard() {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [publicIp, setPublicIp] = useState('');
  const [ipError, setIpError] = useState('');

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 30000);
    fetchPublicIp();
    return () => clearInterval(interval);
  }, []);

  const fetchStats = async () => {
    try {
      const response = await statsAPI.getDashboard();
      setStats(response.data);
      setError('');
    } catch (err) {
      setError('Failed to fetch statistics');
    } finally {
      setLoading(false);
    }
  };

  const fetchPublicIp = async () => {
    try {
      setIpError('');
      const response = await systemAPI.getPublicIp();
      setPublicIp(response.data.ip);
    } catch (err) {
      setIpError('Unable to detect public IP automatically');
    }
  };

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  if (loading) {
    return (
      <Layout>
        <div className="loading">Loading dashboard...</div>
      </Layout>
    );
  }

  return (
    <Layout>
      <h2 style={{ marginBottom: '20px' }}>Dashboard</h2>

      {error && <div className="error">{error}</div>}

      {stats && (
        <>
          <div className="stats-grid">
            <div className="stat-card">
              <h3>Total Users</h3>
              <div className="value">{stats.total_users}</div>
            </div>

            <div className="stat-card">
              <h3>Active Users</h3>
              <div className="value">{stats.active_users}</div>
            </div>

            <div className="stat-card">
              <h3>Total Requests</h3>
              <div className="value">{stats.total_requests.toLocaleString()}</div>
            </div>

            <div className="stat-card">
              <h3>Data Sent</h3>
              <div className="value">{formatBytes(stats.total_bytes_sent)}</div>
            </div>

            <div className="stat-card">
              <h3>Data Received</h3>
              <div className="value">{formatBytes(stats.total_bytes_received)}</div>
            </div>
          </div>

          <div className="card">
            <h3 style={{ marginBottom: '16px' }}>Quick Info</h3>
            <p style={{ color: '#666', lineHeight: '1.6' }}>
              Point any HTTP/HTTPS client to your proxy host and authenticate with user credentials or JWT tokens.
              Share these details with your users:
            </p>
            <ul style={{ lineHeight: '1.6', color: '#555', marginLeft: '18px' }}>
              <li>Proxy host: use your public IP ({publicIp || ipError || 'detecting...'}), or internal IP on LAN.</li>
              <li>Proxy ports: HTTP proxy on <strong>18080</strong>, Web UI on <strong>13000</strong> (API via <strong>/api</strong>).</li>
              <li>Authentication: use username/password or obtain a JWT token from the Web UI.</li>
              <li>Whitelist/open the ports in your firewall or router to allow remote clients.</li>
            </ul>
            <button onClick={fetchPublicIp} className="button button-secondary" style={{ marginTop: '12px' }}>
              Refresh Public IP
            </button>
          </div>
        </>
      )}
    </Layout>
  );
}

export default Dashboard;
