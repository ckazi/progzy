import { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { authAPI } from '../services/api';
import { setAuth, isAuthenticated } from '../utils/auth';
import TwoFactorVerify from '../components/TwoFactorVerify';

function Login() {
  const navigate = useNavigate();
  const location = useLocation();
  const [formData, setFormData] = useState({
    username: '',
    password: '',
  });
  const [error, setError] = useState(location.state?.error || '');
  const [loading, setLoading] = useState(false);
  const [checkingInit, setCheckingInit] = useState(true);
  const [pendingTwoFA, setPendingTwoFA] = useState(null);

  useEffect(() => {
    if (isAuthenticated()) {
      navigate('/dashboard');
      return;
    }

    const checkInitialization = async () => {
      try {
        const response = await authAPI.checkInit();
        if (!response.data.initialized) {
          navigate('/init-setup');
        }
      } catch (err) {
        console.error('Failed to check initialization:', err);
      } finally {
        setCheckingInit(false);
      }
    };

    checkInitialization();
  }, [navigate]);

  useEffect(() => {
    if (location.state?.error) {
      setError(location.state.error);
    }
  }, [location.state]);

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

    try {
      const response = await authAPI.login(formData);
      if (!response.data.user?.is_admin) {
        setError('Web UI is restricted to administrators only.');
        return;
      }
      if (response.data.requires_2fa) {
        setPendingTwoFA({
          token: response.data.temp_token,
          user: response.data.user,
        });
        setError('');
      } else {
        setAuth(response.data.token, response.data.user);
        navigate('/dashboard', { replace: true });
      }
    } catch (err) {
      setError(err.response?.data?.error || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const handleTwoFASuccess = (payload) => {
    if (payload?.token && payload?.user) {
      setAuth(payload.token, payload.user);
      navigate('/dashboard', { replace: true });
    }
  };

  const handleCancelTwoFA = () => {
    setPendingTwoFA(null);
    setFormData({ username: '', password: '' });
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
        <h1>Proxy Server Login</h1>

        {error && <div className="error">{error}</div>}

        {pendingTwoFA ? (
          <TwoFactorVerify
            token={pendingTwoFA.token}
            user={pendingTwoFA.user}
            onSuccess={handleTwoFASuccess}
            onCancel={handleCancelTwoFA}
          />
        ) : (
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
              <label>Password</label>
              <input
                type="password"
                name="password"
                value={formData.password}
                onChange={handleChange}
                className="input"
                required
              />
            </div>

            <button
              type="submit"
              className="button button-primary"
              style={{ width: '100%' }}
              disabled={loading}
            >
              {loading ? 'Logging in...' : 'Login'}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}

export default Login;
