import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import fs from 'fs';
import path from 'path';
import { parse } from 'yaml';
import { fileURLToPath } from 'url';

// 读取并解析全局 config.yaml 文件
function getGlobalConfig(): any {
  try {
    const __dirname = path.dirname(fileURLToPath(import.meta.url));
    const configPath = path.resolve(__dirname, '../config.yaml');
    if (fs.existsSync(configPath)) {
      const fileContent = fs.readFileSync(configPath, 'utf-8');
      return parse(fileContent) || {};
    }
  } catch (err) {
    console.warn('⚠️ 读取或解析 config.yaml 失败，将采用默认配置', err);
  }
  return {};
}

const config = getGlobalConfig();
const backendPort = config.server?.port || ':8080';
const frontendPort = config.server?.frontend_port || 5173;

// 确保后端端口以 : 开头
const formattedBackendPort = String(backendPort).startsWith(':') ? backendPort : `:${backendPort}`;

export default defineConfig({
  plugins: [react()],
  server: {
    port: Number(frontendPort),
    proxy: {
      '/api': {
        target: `http://localhost${formattedBackendPort}`,
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
