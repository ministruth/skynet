import { RequestConfig } from '@umijs/max';
import { message } from 'antd';
import 'antd/dist/reset.css';
import { stringify } from 'qs';

export const request: RequestConfig = {
  baseURL: '/api',
  timeout: 10000,
  paramsSerializer: function (params) {
    return stringify(params, { encodeValuesOnly: true });
  },
  errorConfig: {
    errorHandler: (error: any, opts: any) => {
      if (opts?.skipErrorHandler) throw error;
      if (error.response) {
        message.error(`${error.response.status}: ${error.response.statusText}`);
        if (error.response.status === 403)
          setTimeout(() => {
            window.location.href = '/';
          }, 1000);
      } else {
        message.error('Unknown error');
      }
    },
  },
};
