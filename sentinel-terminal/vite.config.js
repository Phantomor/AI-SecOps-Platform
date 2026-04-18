import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src')
    }
  },
  server: {
    port: 5173, // Vite 推荐默认端口
    cors: true,
    proxy: {
      // 拦截所有以 /api 开头的请求，转发到 Go 后端
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        // 这里不需要 rewrite，因为我们 Go 里的路由组正好也是 /api
      }
    }
  }
})