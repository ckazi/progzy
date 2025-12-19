import { Navigate } from 'react-router-dom';
import { isAuthenticated, isAdmin, clearAuth } from '../utils/auth';

function ProtectedRoute({ children, adminOnly = false }) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />;
  }

  if (adminOnly && !isAdmin()) {
    clearAuth();
    return <Navigate to="/login" replace state={{ error: 'Access restricted to administrators.' }} />;
  }

  return children;
}

export default ProtectedRoute;
