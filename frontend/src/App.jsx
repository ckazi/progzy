import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import InitGate from './components/InitGate';
import ProtectedRoute from './components/ProtectedRoute';
import Login from './pages/Login';
import InitSetup from './pages/InitSetup';
import Dashboard from './pages/Dashboard';
import Users from './pages/Users';
import Logs from './pages/Logs';
import Settings from './pages/Settings';
import Audit from './pages/Audit';
import Profile from './pages/Profile';

function App() {
  return (
    <Router>
      <InitGate />
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/init-setup" element={<InitSetup />} />

        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />

        <Route
          path="/users"
          element={
            <ProtectedRoute adminOnly>
              <Users />
            </ProtectedRoute>
          }
        />
        <Route
          path="/audit"
          element={
            <ProtectedRoute adminOnly>
              <Audit />
            </ProtectedRoute>
          }
        />

        <Route
          path="/logs"
          element={
            <ProtectedRoute>
              <Logs />
            </ProtectedRoute>
          }
        />

        <Route
          path="/settings"
          element={
            <ProtectedRoute adminOnly>
              <Settings />
            </ProtectedRoute>
          }
        />

        <Route
          path="/profile"
          element={
            <ProtectedRoute>
              <Profile />
            </ProtectedRoute>
          }
        />

        <Route path="/" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </Router>
  );
}

export default App;
