import { Outlet } from "react-router-dom";
import LandingNavbar from "./LandingNavbar";

export default function LandingLayout() {

  return (
    <>
      <div className="min-h-screen bg-glow flex flex-col">
        <LandingNavbar />
        <Outlet />
      </div>
    </>
  );
}
