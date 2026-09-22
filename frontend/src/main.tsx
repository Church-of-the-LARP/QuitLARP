import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
} from "react-router-dom";
import GuestRoute from "./components/GuestRoute.tsx";
import SecuredRoute from "./components/SecuredRoute.tsx";
import Login from "./pages/Login.tsx";
import Register from "./pages/Register.tsx";
import ForgotPassword from "./pages/ForgotPassword.tsx";
import ResetPassword from "./pages/ResetPassword.tsx";
import Dashboard from "./pages/Dashboard.tsx";
import Users from "./pages/Users.tsx";
import Create from "./pages/Create.tsx";
import AssessmentGit from "./pages/AssessmentGit.tsx";
import Layout from "./components/Layout.tsx";
import LandingLayout from "./components/LandingLayout.tsx";
import { AuthProvider } from "./scripts/useAuth.tsx";
import "./index.css";
import Pricing from "./pages/Pricing.tsx";
import Landing from "./pages/Landing.tsx";
import AssessmentPreview from "./pages/AssessmentPreview.tsx";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Router>
      <AuthProvider>
        <Routes>
          <Route element={<GuestRoute />}>
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/forgot-password" element={<ForgotPassword />} />
            <Route element={<LandingLayout />}>
              <Route path="/pricing" element={<Pricing />} />
              <Route path="/" element={<Landing />} />
            </Route>
          </Route>
          <Route path="/reset-password" element={<ResetPassword />} />

          <Route element={<SecuredRoute />}>
            <Route element={<Layout />}>
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/create" element={<Create />} />
              <Route path="/assessments/:id/git" element={<AssessmentGit />} />
              <Route path="/assessments/preview/" element={<AssessmentPreview />} />
              <Route element={<SecuredRoute roles={["admin", "superadmin"]} />}>
                <Route path="/users" element={<Users />} />
              </Route>
            </Route>
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </Router>
  </StrictMode>,
);
