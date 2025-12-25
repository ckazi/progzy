import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { authAPI } from '../services/api';
import { setAuth } from '../utils/auth';

function InitSetup() {
  const minPasswordLength = 6;
  const navigate = useNavigate();
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    confirmPassword: '',
    email: '',
  });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [checkingInit, setCheckingInit] = useState(true);

  useEffect(() => {
    const checkInitialization = async () => {
      try {
        const response = await authAPI.checkInit();
        if (response.data.initialized) {
          navigate('/login', { replace: true });
        }
      } catch (err) {
        console.error('Failed to check initialization:', err);
      } finally {
        setCheckingInit(false);
      }
    };

    checkInitialization();
  }, [navigate]);

  const handleChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    if (formData.password !== formData.confirmPassword) {
      setError('Passwords do not match');
      setLoading(false);
      return;
    }
    if (formData.password.length < minPasswordLength) {
      setError(`Password must be at least ${minPasswordLength} characters long`);
      setLoading(false);
      return;
    }

    try {
      const payload = {
        username: formData.username,
        password: formData.password,
        email: formData.email,
      };
      const response = await authAPI.initSetup(payload);
      setAuth(response.data.token, response.data.user);
      navigate('/dashboard');
    } catch (err) {
      setError(err.response?.data?.error || 'Setup failed');
    } finally {
      setLoading(false);
    }
  };

  if (checkingInit) {
    return (
      <div className="auth-container">
        <div className="loading">Checking system status...</div>
      </div>
    );
  }

  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1>Initial Setup</h1>
        <p style={{ textAlign: 'center', marginBottom: '24px', color: '#666' }}>
          Create your admin account to get started
        </p>

        {error && <div className="error">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Username</label>
            <input
              type="text"
              name="username"
              value={formData.username}
              onChange={handleChange}
              className="input"
              required
              autoFocus
            />
          </div>

          <div className="form-group">
            <label>Password (min {minPasswordLength} chars)</label>
            <input
              type="password"
              name="password"
              value={formData.password}
              onChange={handleChange}
              className="input"
              required
            />
          </div>

          <div className="form-group">
            <label>Confirm Password</label>
            <input
              type="password"
              name="confirmPassword"
              value={formData.confirmPassword}
              onChange={handleChange}
              className="input"
              required
            />
          </div>

          <div className="form-group">
            <label>Email (optional)</label>
            <input
              type="email"
              name="email"
              value={formData.email}
              onChange={handleChange}
              className="input"
            />
          </div>

          <button
            type="submit"
            className="button button-primary"
            style={{ width: '100%' }}
            disabled={loading}
          >
            {loading ? 'Setting up...' : 'Create Admin Account'}
          </button>
        </form>
      </div>
    </div>
  );
}

export default InitSetup;
