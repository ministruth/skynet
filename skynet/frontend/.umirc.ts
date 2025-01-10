import { defineConfig } from 'umi';
import WebpackPwaManifest from 'webpack-pwa-manifest';
import pkg from './package.json';

const pwaInfo = {
  name: 'Skynet',
  shortName: 'Skynet',
  description:
    'Service integration and management system, optimized for home-lab use.',
  backgroundColor: '#ffffff',
  themeColor: '#ffffff',
  filename: `manifest.${pkg.version}.json`,
};

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
  links: [
    {
      rel: 'manifest',
      href: `/${pwaInfo.filename}`,
    },
  ],
  metas: [
    {
      name: 'apple-mobile-web-app-title',
      content: pwaInfo.name,
    },
    {
      name: 'apple-mobile-web-app-capable',
      content: 'yes',
    },
    {
      name: 'apple-mobile-web-app-status-bar-style',
      content: 'default',
    },
    {
      name: 'theme-color',
      content: pwaInfo.themeColor,
    },
  ],
  chainWebpack(config) {
    config.plugin('webpack-pwa-manifest').use(WebpackPwaManifest, [
      {
        filename: pwaInfo.filename,
        name: pwaInfo.name,
        short_name: pwaInfo.shortName,
        description: pwaInfo.description,
        background_color: pwaInfo.backgroundColor,
        theme_color: pwaInfo.themeColor,
        start_url: '/',
        ios: true,
        icons: [
          {
            src: 'src/assets/android-chrome-192x192.png',
            size: '192x192',
            type: 'image/png',
          },
          {
            src: 'src/assets/android-chrome-512x512.png',
            size: '512x512',
            type: 'image/png',
          },
          {
            src: 'src/assets/apple-touch-icon.png',
            size: '180x180',
            type: 'image/png',
            ios: true,
          },
        ],
      },
    ]);
  },
});
