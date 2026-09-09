import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { components } from "../api/schema";
import { API_BASE_URL, client, errorMessage } from "../api/client";

export type User = components["schemas"]["User"];
export type Role = User["role"];

export type AuthStatus = "loading" | "anonymous" | "authenticated";

export type ActionResult = { error?: string; notice?: string };

interface AuthContextValue {
  status: AuthStatus;
  user: User | null;
  googleHref: string;
  signIn: (email: string, password: string) => Promise<ActionResult>;
  signUp: (
    username: string,
    email: string,
    password: string,
  ) => Promise<ActionResult>;
  signOut: () => Promise<void>;
  refresh: () => Promise<void>;
  resendVerification: () => Promise<ActionResult>;
  setUser: (user: User | null) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    let cancelled = false;
    client
      .GET("/api/v1/auth/me")
      .then((res) => {
        if (!cancelled) {
          setUser(res.data?.user ?? null);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setUser(null);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoaded(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const refresh = useCallback(async () => {
    const res = await client.GET("/api/v1/auth/me");
    setUser(res.data?.user ?? null);
  }, []);

  const signIn = useCallback(async (email: string, password: string) => {
    const res = await client.POST("/api/v1/auth/login", {
      body: { email, password },
    });
    if (res.error) {
      return { error: errorMessage(res.error, "Sign in failed") };
    }
    setUser(res.data?.user ?? null);
    return {};
  }, []);

  const signUp = useCallback(
    async (username: string, email: string, password: string) => {
      const res = await client.POST("/api/v1/auth/register", {
        body: { username, email, password },
      });
      if (res.error) {
        return { error: errorMessage(res.error, "Registration failed") };
      }
      setUser(res.data?.user ?? null);
      return {
        notice:
          "Account created — check your inbox for the verification email.",
      };
    },
    [],
  );

  const signOut = useCallback(async () => {
    try {
      await client.POST("/api/v1/auth/logout");
    } catch {
      // The cookie is cleared below regardless; nothing else to do.
    }
    setUser(null);
  }, []);

  const resendVerification = useCallback(async () => {
    const res = await client.POST("/api/v1/auth/resend-verification");
    if (res.error) {
      return { error: errorMessage(res.error, "Could not send the email") };
    }
    return { notice: res.data?.message ?? "Verification email sent." };
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      status: !loaded ? "loading" : user ? "authenticated" : "anonymous",
      user,
      googleHref: `${API_BASE_URL}/api/v1/auth/google`,
      signIn,
      signUp,
      signOut,
      refresh,
      resendVerification,
      setUser,
    }),
    [loaded, user, signIn, signUp, signOut, refresh, resendVerification],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used inside <AuthProvider>");
  }
  return value;
}
