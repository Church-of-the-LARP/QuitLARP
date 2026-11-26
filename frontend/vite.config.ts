import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  // The container runs the Vite dev server behind Traefik/Cloudflare, where the
  // Host header is the public hostname rather than localhost.
  server: {
    allowedHosts: true,
    // Single-hostname deployments: the SPA calls the backend on the same origin
    // and these prefixes are proxied to it. With a separate API hostname,
    // VITE_API_URL points there directly and these proxies stay unused.
    proxy: {
      "/api": "http://backend:8888",
      "/git": "http://backend:8888",
      "/deploy": "http://backend:8888",
      "/openapi.json": "http://backend:8888",
    },
  },
});
