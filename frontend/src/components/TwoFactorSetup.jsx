import { useEffect, useState } from 'react';
import BackupCodesModal from './BackupCodesModal';
import { twoFactorAPI } from '../services/api';
import { getUser } from '../utils/auth';

function TwoFactorSetup({ initialEnabled = false, onStatusChange }) {
  const [enabled, setEnabled] = useState(initialEnabled);
  const [setupData, setSetupData] = useState(null);
  const [verificationCode, setVerificationCode] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [regenCode, setRegenCode] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [backupCodes, setBackupCodes] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [profileEmail, setProfileEmail] = useState('');

  useEffect(() => {
    setEnabled(initialEnabled);
  }, [initialEnabled]);

  useEffect(() => {
    const user = getUser();
    if (user?.email) {
      setProfileEmail(user.email);
    }
  }, []);

  const startSetup = async () => {
    setLoading(true);
    setError('');
    setMessage('');
    try {
      const response = await twoFactorAPI.setup();
      setSetupData(response.data);
      setVerificationCode('');
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to generate a secret key.');
    } finally {
      setLoading(false);
    }
  };

  const verifySetup = async () => {
    if (!verificationCode) {
      setError('Enter the 6-digit code from your authenticator app.');
      return;
    }

    setLoading(true);
    setError('');
    try {
      const response = await twoFactorAPI.verifySetup(verificationCode);
      setBackupCodes(response.data.backup_codes || []);
      setModalVisible(true);
      setSetupData(null);
      setVerificationCode('');
      setEnabled(true);
      setMessage(response.data.message || 'Two-factor authentication activated.');
      onStatusChange?.(true);
    } catch (err) {
      setError(err.response?.data?.error || 'The code did not match. Try again.');
    } finally {
      setLoading(false);
    }
  };

  const buildPayload = (value) => {
    if (!value) {
      return null;
    }
    return value.length === 8 ? { backup_code: value } : { code: value };
  };

  const disableTwoFA = async () => {
    const payload = buildPayload(disableCode);
    if (!payload) {
      setError('Enter a valid code to disable 2FA.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await twoFactorAPI.disable(payload);
      setEnabled(false);
      setDisableCode('');
      setMessage('Two-factor authentication disabled.');
      onStatusChange?.(false);
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to disable 2FA.');
    } finally {
      setLoading(false);
    }
  };

  const regenerateCodes = async () => {
    const payload = buildPayload(regenCode);
    if (!payload) {
      setError('Enter a valid code to generate backup codes.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const response = await twoFactorAPI.regenerateCodes(payload);
      setBackupCodes(response.data.codes || []);
      setModalVisible(true);
      setRegenCode('');
      setMessage('New backup codes generated.');
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to regenerate backup codes.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h3 style={{ marginBottom: '8px' }}>Two-Factor Authentication</h3>
          <div style={{ color: enabled ? '#6ddba0' : '#ffb347' }}>
            Status: {enabled ? 'Enabled' : 'Disabled'}
          </div>
        </div>
        {!enabled && (
          <button className="button button-primary" onClick={startSetup} disabled={loading}>
            Enable Two-Factor Authentication
          </button>
        )}
      </div>

      {error && <div className="error" style={{ marginTop: '12px' }}>{error}</div>}
      {message && <div className="success" style={{ marginTop: '12px' }}>{message}</div>}

      {setupData && (
        <div className="card" style={{ marginTop: '16px' }}>
          <h4>Step 1. Scan the QR code</h4>
          <div style={{ display: 'flex', gap: '24px', flexWrap: 'wrap', alignItems: 'center', marginTop: '12px' }}>
            <img src={setupData.qr_code} alt="2FA QR" style={{ width: '220px', background: '#fff', padding: '12px', borderRadius: '6px' }} />
            <div style={{ flex: 1 }}>
              <p>Or add manually:</p>
              <div style={{ background: '#0b1a2a', padding: '12px', borderRadius: '6px', fontFamily: 'monospace' }}>
                {setupData.secret}
              </div>
              <p style={{ marginTop: '8px', fontSize: '14px', color: '#9fb0c7' }}>
                Email: {profileEmail || 'not provided'} <br />
                URL: {setupData.otpauth_url}
              </p>
            </div>
          </div>
          <h4 style={{ marginTop: '16px' }}>Step 2. Confirm the code</h4>
          <div style={{ display: 'flex', gap: '12px', marginTop: '8px', flexWrap: 'wrap' }}>
            <input
              type="text"
              className="input"
              maxLength={6}
              value={verificationCode}
              onChange={(e) => setVerificationCode(e.target.value.replace(/\D/g, ''))}
              placeholder="6-digit code"
            />
            <button className="button button-primary" onClick={verifySetup} disabled={loading}>
              Confirm & activate
            </button>
          </div>
        </div>
      )}

      {enabled && (
        <div className="card" style={{ marginTop: '16px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div>
            <h4>Disable 2FA</h4>
            <p style={{ fontSize: '14px', color: '#9fb0c7' }}>Enter a verification or backup code to disable protection.</p>
            <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
              <input
                type="text"
                className="input"
                placeholder="App code or backup code"
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value.trim())}
              />
              <button className="button button-secondary" onClick={disableTwoFA} disabled={loading}>
                Disable 2FA
              </button>
            </div>
          </div>

          <div>
            <h4>Regenerate backup codes</h4>
            <p style={{ fontSize: '14px', color: '#9fb0c7' }}>Generate a new set of codes. Older codes will become invalid.</p>
            <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
              <input
                type="text"
                className="input"
                placeholder="App code or backup code"
                value={regenCode}
                onChange={(e) => setRegenCode(e.target.value.trim())}
              />
              <button className="button button-secondary" onClick={regenerateCodes} disabled={loading}>
                Generate new codes
              </button>
            </div>
          </div>
        </div>
      )}

      {modalVisible && (
        <BackupCodesModal
          codes={backupCodes}
          onClose={() => setModalVisible(false)}
        />
      )}
    </div>
  );
}

export default TwoFactorSetup;
