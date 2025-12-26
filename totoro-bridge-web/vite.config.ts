import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(() => {
  return {
    base: "/",
    server: {
      port: 3100,
      host: "0.0.0.0",
      // 开发环境通过同源代理转发到 bridge 后端，避免 CORS（X-Admin-Key 会触发预检）
      proxy: {
        "/api": {
          target: "http://localhost:18090",
          changeOrigin: true,
        },
      },
    },
    build: {
      // 输出到桥梁后端可 embed 的目录（最终由 totoro-bridge 单端口提供）
      outDir: path.resolve(__dirname, "../totoro-bridge/internal/bridgeui/dist"),
      // outDir 在项目根目录之外时，Vite 默认不会清空；这里显式允许清空
      emptyOutDir: true,
    },
    plugins: [react()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "."),
      },
    },
  };
});


