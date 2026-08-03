import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { viteSingleFile } from 'vite-plugin-singlefile';

export default defineConfig(({ command }) => {
  // 单文件构建模式：npm run build:single
  const isSingleFile = process.env.SINGLEFILE === '1';

  return {
    plugins: [
      react(),
      // 单文件模式：把所有 JS/CSS 内联到 index.html
      ...(isSingleFile ? [viteSingleFile()] : []),
    ],
    server: {
      port: 5173,
      proxy: {
        '/api': 'http://localhost:8080',
      },
    },
    build: {
      // 单文件模式需要关闭资源内联上限（否则字体等走独立文件）
      assetsInlineLimit: isSingleFile ? 100000000 : 4096,
    },
    base: isSingleFile ? './' : '/',
  };
});
