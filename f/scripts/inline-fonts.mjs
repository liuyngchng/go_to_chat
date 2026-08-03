// ============================================================
// 构建后脚本：把 webfonts 字体 base64 内联进 dist/index.html
// 使单文件 HTML 在 file:// 下也能正常显示图标
// ============================================================

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const distDir = path.resolve(__dirname, '../dist');
const htmlPath = path.join(distDir, 'index.html');
const webfontDir = path.resolve(__dirname, '../public/webfonts');

if (!fs.existsSync(htmlPath)) {
  console.error('❌ dist/index.html 不存在，请先构建');
  process.exit(1);
}

let html = fs.readFileSync(htmlPath, 'utf8');

// 替换所有 ../webfonts/xxx.woff2 和 ./webfonts/xxx.woff2 为 base64
let replaced = 0;
html = html.replace(/url\(([./]*)(webfonts\/([a-z0-9-]+\.woff2))\)/g, (match, _prefix, _webfont, fontName) => {
  const fontPath = path.join(webfontDir, fontName);
  if (fs.existsSync(fontPath)) {
    const b64 = fs.readFileSync(fontPath).toString('base64');
    replaced++;
    return `url(data:font/woff2;base64,${b64})`;
  }
  console.warn(`⚠️  字体不存在: ${fontPath}`);
  return match;
});

// 兜底：public/webfonts 拷到 dist 的引用也内联
if (replaced === 0) {
  const distWebfonts = path.join(distDir, 'webfonts');
  if (fs.existsSync(distWebfonts)) {
    html = html.replace(/url\((?:\.\.\/)?webfonts\/([a-z0-9-]+\.woff2)\)/g, (match, fontName) => {
      const fontPath = path.join(distWebfonts, fontName);
      if (fs.existsSync(fontPath)) {
        const b64 = fs.readFileSync(fontPath).toString('base64');
        replaced++;
        return `url(data:font/woff2;base64,${b64})`;
      }
      return match;
    });
  }
}

fs.writeFileSync(htmlPath, html);
console.log(`✅ 内联 ${replaced} 个字体文件到 index.html`);
console.log(`📦 最终大小: ${(fs.statSync(htmlPath).size / 1024).toFixed(1)} KB`);
