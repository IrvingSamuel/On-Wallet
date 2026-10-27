import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../backend/web/dist",
    emptyOutDir: false,
  },
  server: {
    proxy: { "/api": "http://127.0.0.1:7546" },
  },
});
