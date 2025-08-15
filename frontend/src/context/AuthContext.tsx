import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { login as apiLogin, register as apiRegister } from '../services/api';
import { AuthState, User } from '../types/index.ts';

interface AuthContextType extends AuthState {
  login: (handle: string, password: string) => Promise<boolean>;
  register: (handle: string, email: string, password: string) => Promise<boolean>;
  logout: () => void;
}

const initialState: AuthState = {
  user: null,
  token: null,
  isAuthenticated: false,
  loading: true,
  error: null,
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [state, setState] = useState<AuthState>(initialState);

  // Check for existing token on mount
  useEffect(() => {
    const initAuth = async () => {
      const token = localStorage.getItem('token');
      const handleStr = localStorage.getItem('userHandle');
      
      if (token && handleStr) {
        setState({
          user: { handle: handleStr },
          token,
          isAuthenticated: true,
          loading: false,
          error: null,
        });
      } else {
        setState({ ...initialState, loading: false });
      }
    };

    initAuth();
  }, []);

  const login = async (handle: string, password: string): Promise<boolean> => {
    setState({ ...state, loading: true, error: null });
    
    const result = await apiLogin({ handle, password });
    
    if (result.error) {
      setState({
        ...state,
        loading: false,
        error: result.error,
      });
      return false;
    }
    
    if (result.data?.token) {
      // Save token to localStorage
      localStorage.setItem('token', result.data.token);
      localStorage.setItem('userHandle', handle);
      
      setState({
        user: { handle },
        token: result.data.token,
        isAuthenticated: true,
        loading: false,
        error: null,
      });
      
      return true;
    }
    
    setState({
      ...state,
      loading: false,
      error: 'Login failed: Invalid response from server',
    });
    
    return false;
  };

  const register = async (
    handle: string,
    email: string,
    password: string
  ): Promise<boolean> => {
    setState({ ...state, loading: true, error: null });
    
    const result = await apiRegister({ handle, email, password });
    
    if (result.error) {
      setState({
        ...state,
        loading: false,
        error: result.error,
      });
      return false;
    }
    
    // Registration successful - now login
    return login(handle, password);
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('userHandle');
    
    setState({
      user: null,
      token: null,
      isAuthenticated: false,
      loading: false,
      error: null,
    });
  };

  return (
    <AuthContext.Provider
      value={{
        ...state,
        login,
        register,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export default AuthContext;
