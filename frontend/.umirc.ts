import { defineConfig } from 'umi';

export default defineConfig({
  base: '/',
  publicPath: '/',
  qiankun: {
    master: {},
  },
  hash: true,
  antd: {},
  exportStatic: {},
  dynamicImport: {
    loading: '@ant-design/pro-layout/es/PageLoading',
  },
  nodeModulesTransform: {
    type: 'none',
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
