import { useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { authAPI } from '../services/api';

function InitGate() {
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    let active = true;

    const checkInitialization = async () => {
      try {
        const response = await authAPI.checkInit();
        if (!active) {
          return;
        }
        if (!response.data.initialized) {
          if (location.pathname !== '/init-setup') {
            navigate('/init-setup', { replace: true });
          }
          return;
        }

        if (location.pathname === '/init-setup') {
          navigate('/login', { replace: true });
        }
      } catch (err) {
        console.error('Failed to check initialization:', err);
      }
    };

    checkInitialization();

    return () => {
      active = false;
    };
  }, [location.pathname, navigate]);

  return null;
}

export default InitGate;
