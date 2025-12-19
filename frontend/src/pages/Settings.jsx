import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import { settingsAPI } from '../services/api';

function Settings() {
  const [settings, setSettings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [editValues, setEditValues] = useState({});
  const [publicIp, setPublicIp] = useState('');
  const [ipError, setIpError] = useState('');

  useEffect(() => {
    fetchSettings();
    fetchPublicIp();
  }, []);

  const fetchSettings = async () => {
    try {
      const response = await settingsAPI.getAll();
      setSettings(response.data);
      const values = {};
      response.data.forEach((setting) => {
        values[setting.key] = setting.value;
      });
      setEditValues(values);
      setError('');
    } catch (err) {
      setError('Failed to fetch settings');
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (key, value) => {
    setEditValues({
      ...editValues,
      [key]: value,
    });
  };

  const fetchPublicIp = async () => {
    try {
      setIpError('');
      const response = await fetch('https://api.ipify.org?format=json');
      if (!response.ok) {
        throw new Error('Failed to fetch IP');
      }
      const data = await response.json();
      setPublicIp(data.ip);
    } catch (err) {
      setIpError('Unable to determine IP');
    }
  };

  const handleSave = async (key) => {
    setError('');
    setSuccess('');

    try {
      await settingsAPI.update(key, editValues[key]);
      setSuccess('Setting updated successfully');
      setTimeout(() => setSuccess(''), 3000);
      fetchSettings();
    } catch (err) {
      setError('Failed to update setting');
    }
  };

  if (loading) {
    return (
      <Layout>
        <div className="loading">Loading settings...</div>
      </Layout>
    );
  }

  return (
    <Layout>
      <h2 style={{ marginBottom: '20px' }}>Proxy Settings</h2>

      {error && <div className="error">{error}</div>}
      {success && <div className="success">{success}</div>}

      <div className="card">
        <table className="table">
          <thead>
            <tr>
              <th>Setting</th>
              <th>Description</th>
              <th>Value</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {settings.map((setting) => (
              <tr key={setting.id}>
                <td>
                  <strong>{setting.key}</strong>
                </td>
                <td>{setting.description}</td>
                <td>
                  {setting.key.startsWith('enable_') || setting.key.startsWith('allow_') ? (
                    <select
                      value={editValues[setting.key]}
                      onChange={(e) => handleChange(setting.key, e.target.value)}
                      className="input"
                      style={{ marginBottom: 0, width: 'auto' }}
                    >
                      <option value="true">Enabled</option>
                      <option value="false">Disabled</option>
                    </select>
                  ) : (
                    <input
                      type="text"
                      value={editValues[setting.key] || ''}
                      onChange={(e) => handleChange(setting.key, e.target.value)}
                      className="input"
                      style={{ marginBottom: 0 }}
                    />
                  )}
                </td>
                <td>
                  <button
                    onClick={() => handleSave(setting.key)}
                    className="button button-primary"
                    style={{ padding: '6px 12px' }}
                  >
                    Save
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="card" style={{ marginTop: '20px' }}>
        <h3 style={{ marginBottom: '12px' }}>Proxy Configuration</h3>
        <p style={{ color: '#666', marginBottom: '12px' }}>
          To use this proxy server, configure your applications with the following settings:
        </p>
        <div style={{ backgroundColor: '#f8f9fa', padding: '16px', borderRadius: '4px', fontFamily: 'monospace' }}>
          <div style={{ marginBottom: '8px' }}>
            <strong>Proxy Host:</strong> {publicIp || ipError || 'Detecting...'}
          </div>
          <div style={{ marginBottom: '8px' }}>
            <strong>Proxy Port:</strong> {editValues['proxy_port'] || '8080'}
          </div>
          <div style={{ marginBottom: '8px' }}>
            <strong>Authentication:</strong> Required (Basic Auth or Bearer Token)
          </div>
          <div>
            <strong>Protocols:</strong> HTTP{editValues['allow_https'] === 'true' && ' / HTTPS'}
          </div>
        </div>
      </div>
    </Layout>
  );
}

export default Settings;
