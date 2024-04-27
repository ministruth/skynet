import { defineConfig } from 'umi';

export default defineConfig({
  //   mfsu: false,
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
  exportStatic: {},
  hash: true,
  antd: {},
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
