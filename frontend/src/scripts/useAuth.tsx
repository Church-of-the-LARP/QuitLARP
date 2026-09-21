import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import client, { getErrorText } from './api.ts';
import type { ActionResult, AuthStatus, User } from '../types/types.ts';

interface AuthContextValue {
  status: AuthStatus;
  user: User | null;
  loginUser: (email: string, password: string) => Promise<ActionResult>;
  registerUser: (
    username: string,
    email: string,
    password: string,
  ) => Promise<ActionResult>;
  logoutUser: () => Promise<void>;
  resendVerification: () => Promise<ActionResult>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;

    const loadSession = async () => {
      try {
        const res = await client.GET('/api/v1/auth/me');
        if (!cancelled) setUser(res.data?.user ?? null);
      } catch (err) {
        console.error('Failed to load the session', err);
        if (!cancelled) setUser(null);
      } finally {
        if (!cancelled) setReady(true);
      }
    };

    loadSession();

    return () => {
      cancelled = true;
    };
  }, []);

  const refreshUser = useCallback(async () => {
    const res = await client.GET('/api/v1/auth/me');
    setUser(res.data?.user ?? null);
  }, []);

  const loginUser = useCallback(async (email: string, password: string) => {
    const res = await client.POST('/api/v1/auth/login', {
      body: { email, password },
    });
    if (res.error) return { error: getErrorText(res.error, 'Sign in failed') };

    setUser(res.data?.user ?? null);
    return {};
  }, []);

  const registerUser = useCallback(
    async (username: string, email: string, password: string) => {
      const res = await client.POST('/api/v1/auth/register', {
        body: { username, email, password },
      });
      if (res.error) {
        return { error: getErrorText(res.error, 'Registration failed') };
      }

      setUser(res.data?.user ?? null);
      return {
        notice: 'Account created — check your inbox for the verification email.',
      };
    },
    [],
  );

  const logoutUser = useCallback(async () => {
    try {
      await client.POST('/api/v1/auth/logout');
    } catch (err) {
      // the cookie gets dropped below either way
      console.error('Failed to log out cleanly', err);
    }
    setUser(null);
  }, []);

  const resendVerification = useCallback(async () => {
    const res = await client.POST('/api/v1/auth/resend-verification');
    if (res.error) {
      return { error: getErrorText(res.error, 'Could not send the email') };
    }

    return { notice: res.data?.message ?? 'Verification email sent.' };
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      status: !ready ? 'loading' : user ? 'authenticated' : 'anonymous',
      user,
      loginUser,
      registerUser,
      logoutUser,
      resendVerification,
      refreshUser,
    }),
    [
      ready,
      user,
      loginUser,
      registerUser,
      logoutUser,
      resendVerification,
      refreshUser,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error('useAuth must be used inside <AuthProvider>');

  return value;
}

export default useAuth;
