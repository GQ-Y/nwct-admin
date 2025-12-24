import path from 'path';
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
    const env = loadEnv(mode, '.', '');
    return {
      base: '/',
      server: {
        port: 3000,
        host: '0.0.0.0',
        // 允许通过 FRP 分配的域名访问 Vite dev server（避免 "Blocked request. This host is not allowed."）
        // 这里用后缀匹配，支持任意前缀：*.frpc.zyckj.club
        allowedHosts: ['.frpc.zyckj.club'],
      },
      build: {
        // 输出到后端可 embed 的目录（最终由 client-nps 单端口提供）
        outDir: path.resolve(__dirname, '../client-nps/internal/webui/dist'),
        // outDir 在项目根目录之外时，Vite 默认不会清空；这里显式允许清空
        emptyOutDir: true,
      },
      plugins: [react()],
      define: {
        'process.env.API_KEY': JSON.stringify(env.GEMINI_API_KEY),
        'process.env.GEMINI_API_KEY': JSON.stringify(env.GEMINI_API_KEY)
      },
      resolve: {
        alias: {
          '@': path.resolve(__dirname, '.'),
        }
      }
    };
});
