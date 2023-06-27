import { defineConfig } from 'umi';

export default defineConfig({
  // mfsu: false,
  access: {
    strictMode: true,
  },
  initialState: {
    loading: '@/components/PageLoading',
  },
  model: {},
  base: '/',
  publicPath: '/',
  qiankun: {
    master: {},
  },
  hash: true,
  antd: {
    import: false,
  },
  locale: {
    default: 'en-US',
    antd: true,
    title: true,
    baseNavigator: true,
  },
  request: {
    dataField: 'data',
  },
  fastRefresh: true,
  proxy: {
    '/api/': {
      target: 'http://localhost:8080/',
      changeOrigin: true,
    },
  },
});
