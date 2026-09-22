import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      // Caddy, not a service: /api is nine upstreams now and only the gateway
      // knows which prefix goes where. Port follows HTTP_PORT from the
      // infrastructure .env — 80 by default, 8080 on a rootless Docker host.
      "/api": {
        target: process.env.DEV_API_TARGET ?? "http://localhost:80",
        changeOrigin: true,
      },
    },
  },
})
