import { defineConfig } from 'umi';

export default defineConfig({
  access: {
    strictMode: true,
  },
  base: '/',
  publicPath: '/',
  qiankun: {
    master: {},
  },
  hash: true,
  antd: {},
  dynamicImport: {
    loading: '@ant-design/pro-layout/es/PageLoading',
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
  fastRefresh: {},
  // mfsu: {},
  webpack5: {},
  proxy: {
    '/api/': {
      target: 'http://localhost:8080/',
      changeOrigin: true,
    },
  },
});
