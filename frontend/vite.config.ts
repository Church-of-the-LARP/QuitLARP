import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  // The container runs the Vite dev server behind Traefik/Cloudflare, where the
  // Host header is the public hostname rather than localhost.
  server: { allowedHosts: true },
});
