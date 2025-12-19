import { useState, useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import Layout from '../components/Layout';
import { usersAPI } from '../services/api';

function Users() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    confirmPassword: '',
    email: '',
    comment: '',
    is_admin: false,
    is_active: true,
    proxy_type: 'default',
  });
  const [whitelistText, setWhitelistText] = useState('');
  const [blacklistText, setBlacklistText] = useState('');
  const [modalLoading, setModalLoading] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  const [pendingEditId, setPendingEditId] = useState(null);

  useEffect(() => {
    fetchUsers();
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const editId = params.get('edit');
    if (editId) {
      setPendingEditId(parseInt(editId, 10));
    } else {
      setPendingEditId(null);
    }
  }, [location.search]);

  useEffect(() => {
    if (pendingEditId && users.length) {
      const targetUser = users.find((u) => u.id === pendingEditId);
      if (targetUser) {
        handleEdit(targetUser);
        navigate(location.pathname, { replace: true });
        setPendingEditId(null);
      }
    }
  }, [pendingEditId, users, navigate, location.pathname]);

  const fetchUsers = async () => {
    try {
      const response = await usersAPI.getAll();
      setUsers(response.data);
      setError('');
    } catch (err) {
      setError('Failed to fetch users');
    } finally {
      setLoading(false);
    }
  };

  const populateFormFromUser = (details) => {
    setEditingUser(details);
    setFormData({
      username: details.username,
      password: '',
      confirmPassword: '',
      email: details.email || '',
      comment: details.comment || '',
      is_admin: details.is_admin,
      is_active: details.is_active,
      proxy_type: details.proxy_type || 'default',
    });
    setWhitelistText((details.whitelist || []).join('\n'));
    setBlacklistText((details.blacklist || []).join('\n'));
  };

  const handleCreate = () => {
    setEditingUser(null);
    setFormData({
      username: '',
      password: '',
      confirmPassword: '',
      email: '',
      comment: '',
      is_admin: false,
      is_active: true,
      proxy_type: 'default',
    });
    setWhitelistText('');
    setBlacklistText('');
    setShowModal(true);
  };

  const handleEdit = async (user) => {
    setError('');
    setShowModal(true);
    setModalLoading(true);
    try {
      const response = await usersAPI.getById(user.id);
      populateFormFromUser(response.data);
    } catch (err) {
      console.error('Failed to load user details, using cached data', err);
      populateFormFromUser(user);
      setError('Failed to load the latest user data; using current values.');
    } finally {
      setModalLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    if (!editingUser && formData.password !== formData.confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (editingUser && formData.password && formData.password !== formData.confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    const proxyType = formData.proxy_type || 'default';
    const parseList = (text) =>
      text
        .split('\n')
        .map((line) => line.trim())
        .filter((line) => line.length > 0);
    const whitelist = parseList(whitelistText);
    const blacklist = parseList(blacklistText);

    try {
      if (editingUser) {
        const updateData = {
          email: formData.email,
          comment: formData.comment,
          is_admin: formData.is_admin,
          is_active: formData.is_active,
          proxy_type: proxyType,
        };
        if (formData.password) {
          updateData.password = formData.password;
        }
        if (!formData.is_admin) {
          updateData.whitelist = whitelist;
          updateData.blacklist = blacklist;
        }
        await usersAPI.update(editingUser.id, updateData);
      } else {
        const newUser = {
          username: formData.username,
          password: formData.password,
          email: formData.email,
          comment: formData.comment,
          is_admin: formData.is_admin,
          proxy_type: proxyType,
        };
        if (!formData.is_admin) {
          newUser.whitelist = whitelist;
          newUser.blacklist = blacklist;
        }
        await usersAPI.create(newUser);
      }
      setShowModal(false);
      fetchUsers();
    } catch (err) {
      setError(err.response?.data?.error || 'Operation failed');
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('Are you sure you want to delete this user?')) return;

    try {
      await usersAPI.delete(id);
      fetchUsers();
    } catch (err) {
      setError('Failed to delete user');
    }
  };

  const handleChange = (e) => {
    const value = e.target.type === 'checkbox' ? e.target.checked : e.target.value;
    if (e.target.name === 'is_admin') {
      setFormData((prev) => ({
        ...prev,
        is_admin: value,
        proxy_type: value ? 'default' : prev.proxy_type,
      }));
      if (value) {
        setWhitelistText('');
        setBlacklistText('');
      }
      return;
    }
    setFormData({
      ...formData,
      [e.target.name]: value,
    });
  };

  if (loading) {
    return (
      <Layout>
        <div className="loading">Loading users...</div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <h2>User Management</h2>
        <button onClick={handleCreate} className="button button-primary">
          Add New User
        </button>
      </div>

      {error && <div className="error">{error}</div>}

      <div className="card">
        <table className="table">
          <thead>
            <tr>
              <th>Username</th>
              <th>Email</th>
              <th>Comment</th>
              <th>Role</th>
              <th>Proxy Type</th>
              <th>Status</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id}>
                <td>{user.username}</td>
                <td>{user.email || '-'}</td>
                <td>{user.comment || '-'}</td>
                <td>
                  {user.is_admin ? (
                    <span className="badge badge-info">Admin</span>
                  ) : (
                    <span className="badge">User</span>
                  )}
                </td>
                <td>{user.proxy_type || 'default'}</td>
                <td>
                  {user.is_active ? (
                    <span className="badge badge-success">Active</span>
                  ) : (
                    <span className="badge badge-danger">Inactive</span>
                  )}
                </td>
                <td>{new Date(user.created_at).toLocaleDateString()}</td>
                <td>
                  <button
                    onClick={() => handleEdit(user)}
                    className="button button-primary"
                    style={{ marginRight: '8px', padding: '6px 12px' }}
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDelete(user.id)}
                    className="button button-danger"
                    style={{ padding: '6px 12px' }}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>{editingUser ? 'Edit User' : 'Create New User'}</h2>
            {modalLoading ? (
              <div className="loading">Loading user...</div>
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
                  disabled={!!editingUser}
                />
              </div>

              <div className="form-group">
                <label>Password {editingUser && '(leave empty to keep current)'}</label>
                <input
                  type="password"
                  name="password"
                  value={formData.password}
                  onChange={handleChange}
                  className="input"
                  required={!editingUser}
                />
              </div>

              <div className="form-group">
                <label>Confirm Password {editingUser && '(match new password)'}</label>
                <input
                  type="password"
                  name="confirmPassword"
                  value={formData.confirmPassword}
                  onChange={handleChange}
                  className="input"
                  required={!editingUser}
                />
              </div>

              <div className="form-group">
                <label>Email</label>
                <input
                  type="email"
                  name="email"
                  value={formData.email}
                  onChange={handleChange}
                  className="input"
                />
              </div>

              <div className="form-group">
                <label>Comment</label>
                <input
                  type="text"
                  name="comment"
                  value={formData.comment}
                  onChange={handleChange}
                  className="input"
                />
              </div>

              <div className="form-group">
                <label>Proxy Type</label>
                <select
                  name="proxy_type"
                  value={formData.proxy_type}
                  onChange={handleChange}
                  className="input"
                  disabled={formData.is_admin}
                >
                  <option value="default">Default</option>
                  <option value="whitelist">White List</option>
                  <option value="blacklist">Black List</option>
                </select>
              </div>

              {!formData.is_admin && (
                <>
                  <div className="form-group">
                    <label>White List (one domain per line)</label>
                    <textarea
                      value={whitelistText}
                      onChange={(e) => setWhitelistText(e.target.value)}
                      className="input"
                      rows={4}
                      placeholder="example.com"
                    />
                  </div>
                  <div className="form-group">
                    <label>Black List (one domain per line)</label>
                    <textarea
                      value={blacklistText}
                      onChange={(e) => setBlacklistText(e.target.value)}
                      className="input"
                      rows={4}
                      placeholder="example.com"
                    />
                  </div>
                </>
              )}

              <div className="form-group">
                <div className="checkbox-group">
                  <input
                    type="checkbox"
                    name="is_admin"
                    checked={formData.is_admin}
                    onChange={handleChange}
                    id="is_admin"
                  />
                  <label htmlFor="is_admin">Admin</label>
                </div>
              </div>

              {editingUser && (
                <div className="form-group">
                  <div className="checkbox-group">
                    <input
                      type="checkbox"
                      name="is_active"
                      checked={formData.is_active}
                      onChange={handleChange}
                      id="is_active"
                    />
                    <label htmlFor="is_active">Active</label>
                  </div>
                </div>
              )}

              <div className="modal-actions">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="button button-secondary"
                >
                  Cancel
                </button>
                <button type="submit" className="button button-primary">
                  {editingUser ? 'Update' : 'Create'}
                </button>
              </div>
              </form>
            )}
          </div>
        </div>
      )}
    </Layout>
  );
}

export default Users;
