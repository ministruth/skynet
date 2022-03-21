import { defineConfig } from 'umi';
import { PLUGIN_ID } from './src/config';

export default defineConfig({
  base: `/plugin/${PLUGIN_ID}/`,
  publicPath: `/_plugin/${PLUGIN_ID}/`,
  qiankun: {
    slave: {},
  },
  chainWebpack(memo, { env, webpack, createCSSRule }) {
    memo.resolve.symlinks(false);
  },
  hash: true,
  antd: {},
  exportStatic: {},
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
