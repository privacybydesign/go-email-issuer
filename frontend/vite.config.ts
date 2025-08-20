import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  return {
    plugins: [react()],
    server:
      mode === "development"
        ? {
            port: 3000,
            host: "localhost",
            proxy: {
              "/api/send": {
                target: "http://localhost:8080/",
                changeOrigin: true,
              },
              "/api/verify": {
                target: "http://localhost:8080/",
                changeOrigin: true,
              },
            },
          }
        : undefined,
    build: {
      outDir: "build",
    },
  };
});
