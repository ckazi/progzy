import { useState } from 'react';

function BackupCodesModal({ codes = [], onClose }) {
  const [status, setStatus] = useState('');

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(codes.join('\n'));
      setStatus('Codes copied to clipboard.');
    } catch (err) {
      setStatus('Failed to copy codes.');
    }
  };

  const handleDownload = () => {
    const blob = new Blob([codes.join('\n')], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `backup-codes-${Date.now()}.txt`;
    link.click();
    URL.revokeObjectURL(url);
    setStatus('Backup codes file downloaded.');
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 2000,
      }}
    >
      <div
        style={{
          background: '#0f2338',
          padding: '24px',
          borderRadius: '8px',
          width: '400px',
          color: '#fff',
          boxShadow: '0 20px 60px rgba(0,0,0,0.5)',
        }}
      >
        <h3 style={{ marginBottom: '12px' }}>Backup Codes</h3>
        <p style={{ fontSize: '14px', color: '#f7b955', marginBottom: '12px' }}>
          Store these codes in a safe place. Each code can be used only once.
        </p>
        <div
          style={{
            background: 'rgba(255,255,255,0.05)',
            borderRadius: '6px',
            padding: '12px',
            fontFamily: 'monospace',
            marginBottom: '16px',
            maxHeight: '160px',
            overflowY: 'auto',
          }}
        >
          {codes.map((code) => (
            <div key={code} style={{ letterSpacing: '2px', padding: '4px 0' }}>
              {code}
            </div>
          ))}
        </div>
        {status && <div style={{ marginBottom: '12px', color: '#6ddba0' }}>{status}</div>}
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: '12px' }}>
          <button className="button button-secondary" onClick={handleDownload} style={{ flex: 1 }}>
            Download codes
          </button>
          <button className="button button-secondary" onClick={handleCopy} style={{ flex: 1 }}>
            Copy
          </button>
        </div>
        <button
          className="button button-primary"
          onClick={onClose}
          style={{ width: '100%', marginTop: '16px' }}
        >
          Close
        </button>
      </div>
    </div>
  );
}

export default BackupCodesModal;
