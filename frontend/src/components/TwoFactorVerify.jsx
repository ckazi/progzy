import { useEffect, useRef, useState } from 'react';
import { twoFactorAPI } from '../services/api';

function TwoFactorVerify({ token, user, onSuccess, onCancel }) {
  const [code, setCode] = useState('');
  const [usingBackup, setUsingBackup] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [timeLeft, setTimeLeft] = useState(30);
  const inputRef = useRef(null);

  useEffect(() => {
    const interval = setInterval(() => {
      const seconds = 30 - (Math.floor(Date.now() / 1000) % 30);
      setTimeLeft(seconds === 0 ? 30 : seconds);
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    inputRef.current?.focus();
    setCode('');
    setError('');
  }, [usingBackup]);

  useEffect(() => {
    if (!usingBackup && code.length === 6) {
      handleSubmit();
    }
  }, [code, usingBackup]);

  const sanitizeValue = (value) => {
    if (usingBackup) {
      return value.replace(/[^0-9a-z]/gi, '').toUpperCase().slice(0, 8);
    }
    return value.replace(/\D/g, '').slice(0, 6);
  };

  const handleChange = (e) => {
    setCode(sanitizeValue(e.target.value));
    setError('');
  };

  const handleSubmit = async (e) => {
    e?.preventDefault();
    if (loading) {
      return;
    }

    if (!code) {
      setError('Enter the authenticator or backup code.');
      return;
    }

    setLoading(true);
    setError('');
    const payload =
      usingBackup || code.length === 8
        ? { backup_code: code }
        : {
            code,
          };

    try {
      const response = await twoFactorAPI.verifyLogin(payload, token);
      onSuccess?.(response.data);
    } catch (err) {
      const status = err.response?.status;
      if (status === 429) {
        setError('Too many attempts. Try again later.');
      } else if (status === 401) {
        setError('Incorrect code. Please try again.');
      } else {
        setError('Failed to verify the code.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2>Two-Factor Verification</h2>
      <p style={{ color: '#9fb0c7' }}>
        Enter a one-time code for account <strong>{user?.username}</strong>.
      </p>
      <form onSubmit={handleSubmit} style={{ marginTop: '16px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
        <input
          type="text"
          inputMode={usingBackup ? 'text' : 'numeric'}
          value={code}
          onChange={handleChange}
          ref={inputRef}
          className="input"
          placeholder={usingBackup ? '8-character backup code' : '6-digit code'}
          maxLength={usingBackup ? 8 : 6}
          autoComplete="one-time-code"
        />

        <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '14px' }}>
          <input
            type="checkbox"
            checked={usingBackup}
            onChange={() => setUsingBackup((prev) => !prev)}
          />
          Use backup code
        </label>

        {!usingBackup && (
          <div style={{ fontSize: '14px', color: '#9fb0c7' }}>Code refreshes in {timeLeft}s.</div>
        )}

        {error && <div className="error">{error}</div>}

        <div style={{ display: 'flex', gap: '12px' }}>
          <button type="submit" className="button button-primary" disabled={loading} style={{ flex: 1 }}>
            {loading ? 'Verifying...' : 'Confirm'}
          </button>
          <button type="button" className="button button-secondary" onClick={onCancel} style={{ flex: 1 }}>
            Change account
          </button>
        </div>
      </form>
    </div>
  );
}

export default TwoFactorVerify;
