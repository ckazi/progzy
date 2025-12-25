import { useState, useEffect, useMemo } from 'react';
import Layout from '../components/Layout';
import { logsAPI, statsAPI } from '../services/api';

const defaultLogFilters = {
  startDate: '',
  endDate: '',
  username: '',
  method: '',
  url: '',
  status: '',
  minSent: '',
  maxSent: '',
  minReceived: '',
  maxReceived: '',
  minDuration: '',
  maxDuration: '',
};

const defaultStatsRange = {
  startDate: '',
  endDate: '',
};

function Logs() {
  const [logs, setLogs] = useState([]);
  const [trafficStats, setTrafficStats] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState('logs');
  const [limit, setLimit] = useState(25);
  const [logFilters, setLogFilters] = useState(() => ({ ...defaultLogFilters }));
  const [appliedLogFilters, setAppliedLogFilters] = useState(() => ({ ...defaultLogFilters }));
  const [statsRange, setStatsRange] = useState(() => ({ ...defaultStatsRange }));
  const [appliedStatsRange, setAppliedStatsRange] = useState(() => ({ ...defaultStatsRange }));
  const [sortField, setSortField] = useState('created_at');
  const [sortOrder, setSortOrder] = useState('desc');
  const [retentionDays, setRetentionDays] = useState('30');
  const [retentionMessage, setRetentionMessage] = useState('');
  const [retentionLoading, setRetentionLoading] = useState(false);

  useEffect(() => {
    if (activeTab === 'logs') {
      fetchLogs();
    }
  }, [activeTab, limit, sortField, sortOrder, appliedLogFilters]);

  useEffect(() => {
    if (activeTab === 'stats') {
      fetchStats();
    }
  }, [activeTab, limit, appliedStatsRange]);

  useEffect(() => {
    const loadRetention = async () => {
      try {
        const response = await logsAPI.getRetention();
        if (response.data?.retention_days) {
          setRetentionDays(response.data.retention_days.toString());
        }
      } catch (err) {
        console.error('Failed to fetch retention settings', err);
      }
    };
    loadRetention();
  }, []);

  const buildLogParams = useMemo(() => {
    const filters = appliedLogFilters;
    const params = {
      limit,
      sort_by: sortField,
      sort_order: sortOrder,
    };
    if (filters.startDate) params.start_date = filters.startDate;
    if (filters.endDate) params.end_date = filters.endDate;
    if (filters.username) params.username = filters.username;
    if (filters.method) params.method = filters.method;
    if (filters.url) params.url = filters.url;
    if (filters.status) params.status = filters.status;
    if (filters.minSent) params.min_sent = filters.minSent;
    if (filters.maxSent) params.max_sent = filters.maxSent;
    if (filters.minReceived) params.min_received = filters.minReceived;
    if (filters.maxReceived) params.max_received = filters.maxReceived;
    if (filters.minDuration) params.min_duration = filters.minDuration;
    if (filters.maxDuration) params.max_duration = filters.maxDuration;
    return params;
  }, [appliedLogFilters, limit, sortField, sortOrder]);

  const buildStatsParams = useMemo(() => {
    const params = { limit };
    if (appliedStatsRange.startDate) params.start_date = appliedStatsRange.startDate;
    if (appliedStatsRange.endDate) params.end_date = appliedStatsRange.endDate;
    return params;
  }, [appliedStatsRange, limit]);

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const response = await logsAPI.getRequests(buildLogParams);
      setLogs(response.data || []);
      setError('');
    } catch (err) {
      console.error(err);
      setError('Failed to fetch request logs');
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    setLoading(true);
    try {
      const response = await statsAPI.getTraffic(buildStatsParams);
      setTrafficStats(response.data || []);
      setError('');
    } catch (err) {
      console.error(err);
      setError('Failed to fetch traffic statistics');
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${Math.round((bytes / Math.pow(k, i)) * 100) / 100} ${sizes[i]}`;
  };

  const getStatusColor = (status) => {
    if (status >= 200 && status < 300) return '#27ae60';
    if (status >= 300 && status < 400) return '#3498db';
    if (status >= 400 && status < 500) return '#e67e22';
    return '#e74c3c';
  };

  const handleFilterChange = (e) => {
    const { name, value } = e.target;
    setLogFilters((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleApplyLogFilters = () => {
    setAppliedLogFilters({ ...logFilters });
  };

  const handleResetLogFilters = () => {
    setLogFilters({ ...defaultLogFilters });
    setAppliedLogFilters({ ...defaultLogFilters });
  };

  const handleStatsRangeChange = (e) => {
    const { name, value } = e.target;
    setStatsRange((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleApplyStatsRange = () => {
    setAppliedStatsRange({ ...statsRange });
  };

  const handleSort = (field) => {
    if (sortField === field) {
      setSortOrder((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortField(field);
      setSortOrder('asc');
    }
  };

  const renderSortIndicator = (field) => {
    if (sortField !== field) return null;
    return sortOrder === 'asc' ? '▲' : '▼';
  };

  const handleExport = async (format) => {
    try {
      setError('');
      const params = { ...buildLogParams, format };
      const response = await logsAPI.exportRequests(params);
      const blob = new Blob([response.data], {
        type:
          format === 'pdf'
            ? 'application/pdf'
            : 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      const link = document.createElement('a');
      const url = window.URL.createObjectURL(blob);
      link.href = url;
      const extension = format === 'pdf' ? 'pdf' : 'xlsx';
      link.download = `request-logs-${new Date().toISOString().slice(0, 10)}.${extension}`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error(err);
      setError('Failed to export logs');
    }
  };

  const handleClearLogs = async () => {
    setRetentionMessage('');
    setError('');
    const days = parseInt(retentionDays, 10);
    if (isNaN(days) || days <= 0) {
      setError('Enter a valid number of retention days.');
      return;
    }
    setRetentionLoading(true);
    try {
      const response = await logsAPI.clearLogs({ days });
      if (response.data?.message) {
        setRetentionMessage(response.data.message);
      } else {
        setRetentionMessage(`Logs older than ${days} days cleared.`);
      }
      fetchLogs();
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to clear logs');
    } finally {
      setRetentionLoading(false);
    }
  };

  return (
    <Layout>
      <h2 style={{ marginBottom: '20px' }}>Logs & Statistics</h2>

      <div className="card" style={{ marginBottom: '20px' }}>
        <div style={{ display: 'flex', gap: '12px', marginBottom: '16px' }}>
          <button
            onClick={() => setActiveTab('logs')}
            className={`button ${activeTab === 'logs' ? 'button-primary' : 'button-secondary'}`}
          >
            Request Logs
          </button>
          <button
            onClick={() => setActiveTab('stats')}
            className={`button ${activeTab === 'stats' ? 'button-primary' : 'button-secondary'}`}
          >
            Traffic Statistics
          </button>
        </div>

        <div style={{ display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap' }}>
          <label>Results per view:</label>
          <select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="input"
            style={{ width: '120px', marginBottom: 0 }}
          >
            <option value={25}>25</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
            <option value={200}>200</option>
            <option value={500}>500</option>
          </select>
        </div>
      </div>

      {activeTab === 'logs' && (
        <div className="card" style={{ marginBottom: '20px' }}>
          <h3 style={{ marginBottom: '16px' }}>Filters</h3>
          <div className="form-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '12px' }}>
            <div>
              <label>Date From</label>
              <input type="date" name="startDate" value={logFilters.startDate} onChange={handleFilterChange} className="input" />
            </div>
            <div>
              <label>Date To</label>
              <input type="date" name="endDate" value={logFilters.endDate} onChange={handleFilterChange} className="input" />
            </div>
            <div>
              <label>User</label>
              <input
                name="username"
                value={logFilters.username}
                onChange={handleFilterChange}
                className="input"
                placeholder="Enter username"
              />
            </div>
            <div>
              <label>Method</label>
              <select
                name="method"
                value={logFilters.method}
                onChange={handleFilterChange}
                className="input"
              >
                <option value="">All</option>
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="DELETE">DELETE</option>
                <option value="PATCH">PATCH</option>
                <option value="CONNECT">CONNECT</option>
                <option value="OPTIONS">OPTIONS</option>
                <option value="HEAD">HEAD</option>
              </select>
            </div>
            <div>
              <label>Status</label>
              <input type="number" name="status" value={logFilters.status} onChange={handleFilterChange} className="input" placeholder="e.g. 200" />
            </div>
            <div>
              <label>URL Contains</label>
              <input name="url" value={logFilters.url} onChange={handleFilterChange} className="input" placeholder="example.com" />
            </div>
            <div>
              <label>Sent (Bytes) Min</label>
              <input type="number" name="minSent" value={logFilters.minSent} onChange={handleFilterChange} className="input" min="0" />
            </div>
            <div>
              <label>Sent (Bytes) Max</label>
              <input type="number" name="maxSent" value={logFilters.maxSent} onChange={handleFilterChange} className="input" min="0" />
            </div>
            <div>
              <label>Received (Bytes) Min</label>
              <input type="number" name="minReceived" value={logFilters.minReceived} onChange={handleFilterChange} className="input" min="0" />
            </div>
            <div>
              <label>Received (Bytes) Max</label>
              <input type="number" name="maxReceived" value={logFilters.maxReceived} onChange={handleFilterChange} className="input" min="0" />
            </div>
            <div>
              <label>Duration Min (ms)</label>
              <input type="number" name="minDuration" value={logFilters.minDuration} onChange={handleFilterChange} className="input" min="0" />
            </div>
            <div>
              <label>Duration Max (ms)</label>
              <input type="number" name="maxDuration" value={logFilters.maxDuration} onChange={handleFilterChange} className="input" min="0" />
            </div>
          </div>
          <div style={{ display: 'flex', gap: '12px', marginTop: '16px', flexWrap: 'wrap' }}>
            <button onClick={handleApplyLogFilters} className="button button-primary">
              Apply Filters
            </button>
            <button onClick={handleResetLogFilters} className="button button-secondary">
              Reset
            </button>
            <button onClick={() => handleExport('pdf')} className="button button-secondary" disabled={!logs.length}>
              Export PDF
            </button>
            <button onClick={() => handleExport('excel')} className="button button-secondary" disabled={!logs.length}>
              Export Excel
            </button>
          </div>
        </div>
      )}

      {activeTab === 'logs' && (
        <div className="card" style={{ marginBottom: '20px' }}>
          <h3 style={{ marginBottom: '12px' }}>Log Retention</h3>
          <p style={{ color: '#666', marginBottom: '12px' }}>
            Specify how many days to keep request logs and click &laquo;Clearing logs&raquo; to purge older data immediately.
          </p>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
            <label htmlFor="retentionDays">Days to keep:</label>
            <input
              id="retentionDays"
              type="number"
              min="1"
              className="input"
              value={retentionDays}
              onChange={(e) => setRetentionDays(e.target.value)}
              style={{ maxWidth: '120px', marginBottom: 0 }}
            />
            <button
              onClick={handleClearLogs}
              className="button button-secondary"
              disabled={retentionLoading}
            >
              {retentionLoading ? 'Clearing...' : 'Clearing logs'}
            </button>
          </div>
          {retentionMessage && (
            <div className="success" style={{ marginTop: '12px' }}>
              {retentionMessage}
            </div>
          )}
        </div>
      )}

      {activeTab === 'stats' && (
        <div className="card" style={{ marginBottom: '20px' }}>
          <h3 style={{ marginBottom: '16px' }}>Statistics Period</h3>
          <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
            <div>
              <label>Date From</label>
              <input type="date" name="startDate" value={statsRange.startDate} onChange={handleStatsRangeChange} className="input" />
            </div>
            <div>
              <label>Date To</label>
              <input type="date" name="endDate" value={statsRange.endDate} onChange={handleStatsRangeChange} className="input" />
            </div>
          </div>
          <div style={{ marginTop: '16px' }}>
            <button onClick={handleApplyStatsRange} className="button button-primary">
              Apply Period
            </button>
          </div>
        </div>
      )}

      {error && <div className="error">{error}</div>}

      {loading ? (
        <div className="loading">Loading data...</div>
      ) : (
        <div className="card">
          {activeTab === 'logs' ? (
            logs.length ? (
              <table className="table">
                <thead>
                  <tr>
                    <th onClick={() => handleSort('created_at')} style={{ cursor: 'pointer' }}>
                      Time {renderSortIndicator('created_at')}
                    </th>
                    <th onClick={() => handleSort('username')} style={{ cursor: 'pointer' }}>
                      User {renderSortIndicator('username')}
                    </th>
                    <th onClick={() => handleSort('method')} style={{ cursor: 'pointer' }}>
                      Method {renderSortIndicator('method')}
                    </th>
                    <th onClick={() => handleSort('url')} style={{ cursor: 'pointer' }}>
                      URL {renderSortIndicator('url')}
                    </th>
                    <th onClick={() => handleSort('status_code')} style={{ cursor: 'pointer' }}>
                      Status {renderSortIndicator('status_code')}
                    </th>
                    <th onClick={() => handleSort('bytes_sent')} style={{ cursor: 'pointer' }}>
                      Sent {renderSortIndicator('bytes_sent')}
                    </th>
                    <th onClick={() => handleSort('bytes_received')} style={{ cursor: 'pointer' }}>
                      Received {renderSortIndicator('bytes_received')}
                    </th>
                    <th onClick={() => handleSort('duration_ms')} style={{ cursor: 'pointer' }}>
                      Duration {renderSortIndicator('duration_ms')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log) => (
                    <tr key={log.id}>
                      <td>{new Date(log.created_at).toLocaleString()}</td>
                      <td>{log.username}</td>
                      <td>
                        <span className="badge badge-info">{log.method}</span>
                      </td>
                      <td style={{ maxWidth: '400px', overflow: 'hidden', textOverflow: 'ellipsis' }}>{log.url}</td>
                      <td>
                        <span className="badge" style={{ backgroundColor: getStatusColor(log.status_code), color: 'white' }}>
                          {log.status_code}
                        </span>
                      </td>
                      <td>{formatBytes(log.bytes_sent)}</td>
                      <td>{formatBytes(log.bytes_received)}</td>
                      <td>{log.duration_ms}ms</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <div style={{ textAlign: 'center', padding: '20px' }}>No logs found for the selected filters.</div>
            )
          ) : trafficStats.length ? (
            <table className="table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>User</th>
                  <th>Requests</th>
                  <th>Data Sent</th>
                  <th>Data Received</th>
                  <th>Total Traffic</th>
                </tr>
              </thead>
              <tbody>
                {trafficStats.map((stat) => (
                  <tr key={`${stat.id}-${stat.date}`}>
                    <td>{new Date(stat.date).toLocaleDateString()}</td>
                    <td>{stat.username}</td>
                    <td>{stat.request_count?.toLocaleString()}</td>
                    <td>{formatBytes(stat.bytes_sent)}</td>
                    <td>{formatBytes(stat.bytes_received)}</td>
                    <td>{formatBytes(stat.bytes_sent + stat.bytes_received)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div style={{ textAlign: 'center', padding: '20px' }}>No statistics for the selected period.</div>
          )}
        </div>
      )}
    </Layout>
  );
}

export default Logs;
