import { useState, useEffect, useCallback } from 'react';
import Layout from '../components/Layout';
import { auditAPI } from '../services/api';

const defaultFilters = {
  startDate: '',
  endDate: '',
  username: '',
  action: '',
  details: '',
  ip: '',
};

function Audit() {
  const [logs, setLogs] = useState([]);
  const [limit, setLimit] = useState(200);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filters, setFilters] = useState(() => ({ ...defaultFilters }));
  const [appliedFilters, setAppliedFilters] = useState(() => ({ ...defaultFilters }));
  const [sortField, setSortField] = useState('created_at');
  const [sortOrder, setSortOrder] = useState('desc');

  const buildParams = useCallback(() => {
    const params = {
      limit,
      sort_by: sortField,
      sort_order: sortOrder,
    };
    if (appliedFilters.startDate) params.start_date = appliedFilters.startDate;
    if (appliedFilters.endDate) params.end_date = appliedFilters.endDate;
    if (appliedFilters.username) params.username = appliedFilters.username;
    if (appliedFilters.action) params.action = appliedFilters.action;
    if (appliedFilters.details) params.details = appliedFilters.details;
    if (appliedFilters.ip) params.ip_address = appliedFilters.ip;
    return params;
  }, [appliedFilters, limit, sortField, sortOrder]);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const response = await auditAPI.getLogs(buildParams());
      setLogs(response.data || []);
      setError('');
    } catch (err) {
      setError('Failed to load audit logs');
    } finally {
      setLoading(false);
    }
  }, [buildParams]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const handleFilterChange = (e) => {
    const { name, value } = e.target;
    setFilters((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleApplyFilters = () => {
    setAppliedFilters({ ...filters });
  };

  const handleResetFilters = () => {
    setFilters({ ...defaultFilters });
    setAppliedFilters({ ...defaultFilters });
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

  const formatTime = (value) => {
    if (!value) return '-';
    return new Date(value).toLocaleString();
  };

  const detailCellStyle = {
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
    fontSize: '13px',
  };

  return (
    <Layout>
      <h2 style={{ marginBottom: '20px' }}>Audit Log</h2>

      <div className="card" style={{ marginBottom: '20px' }}>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
            gap: '12px',
          }}
        >
          <div>
            <label>Start Date</label>
            <input
              type="date"
              name="startDate"
              value={filters.startDate}
              onChange={handleFilterChange}
              className="input"
            />
          </div>
          <div>
            <label>End Date</label>
            <input
              type="date"
              name="endDate"
              value={filters.endDate}
              onChange={handleFilterChange}
              className="input"
            />
          </div>
          <div>
            <label>Admin</label>
            <input
              type="text"
              name="username"
              value={filters.username}
              onChange={handleFilterChange}
              className="input"
              placeholder="Username fragment"
            />
          </div>
          <div>
            <label>Action</label>
            <input
              type="text"
              name="action"
              value={filters.action}
              onChange={handleFilterChange}
              className="input"
              placeholder="Action"
            />
          </div>
          <div>
            <label>Details</label>
            <input
              type="text"
              name="details"
              value={filters.details}
              onChange={handleFilterChange}
              className="input"
              placeholder="Details contains"
            />
          </div>
          <div>
            <label>IP</label>
            <input
              type="text"
              name="ip"
              value={filters.ip}
              onChange={handleFilterChange}
              className="input"
              placeholder="IP fragment"
            />
          </div>
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'flex-end',
            gap: '12px',
            flexWrap: 'wrap',
            marginTop: '16px',
          }}
        >
          <div>
            <label>Records</label>
            <select
              className="input"
              style={{ width: '140px', marginBottom: 0 }}
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
            >
              <option value={100}>100</option>
              <option value={200}>200</option>
              <option value={500}>500</option>
              <option value={1000}>1000</option>
            </select>
          </div>
          <button onClick={handleApplyFilters} className="button button-primary">
            Apply Filters
          </button>
          <button onClick={handleResetFilters} className="button button-secondary">
            Reset
          </button>
          <button onClick={fetchLogs} className="button button-secondary">
            Refresh
          </button>
        </div>
      </div>

      {error && <div className="error">{error}</div>}

      <div className="card">
        {loading ? (
          <div className="loading">Loading audit log...</div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table className="table" style={{ tableLayout: 'fixed', width: '100%', minWidth: '800px' }}>
              <thead>
                <tr>
                  <th style={{ cursor: 'pointer', width: '16%' }} onClick={() => handleSort('created_at')}>
                    Time {renderSortIndicator('created_at')}
                  </th>
                  <th style={{ cursor: 'pointer', width: '12%' }} onClick={() => handleSort('username')}>
                    Admin {renderSortIndicator('username')}
                  </th>
                  <th style={{ cursor: 'pointer', width: '12%' }} onClick={() => handleSort('action')}>
                    Action {renderSortIndicator('action')}
                  </th>
                  <th style={{ cursor: 'pointer', width: '44%' }} onClick={() => handleSort('details')}>
                    Details {renderSortIndicator('details')}
                  </th>
                  <th style={{ cursor: 'pointer', width: '16%' }} onClick={() => handleSort('ip_address')}>
                    IP {renderSortIndicator('ip_address')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.id}>
                    <td>{formatTime(log.created_at)}</td>
                    <td>{log.username || `#${log.user_id || 'n/a'}`}</td>
                    <td>{log.action}</td>
                    <td style={detailCellStyle}>{log.details || '-'}</td>
                    <td>{log.ip_address || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Layout>
  );
}

export default Audit;
