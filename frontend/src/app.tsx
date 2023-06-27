import { getAPI, UserPerm } from '@/utils';
import { RequestConfig } from '@umijs/max';
import { message } from 'antd';
import 'antd/dist/reset.css';
import { stringify } from 'qs';

export interface GlobalState {
  signin: boolean;
  id: string | undefined;
  permission: { [Key: string]: UserPerm };
}

export const request: RequestConfig = {
  baseURL: '/api',
  timeout: 10000,
  paramsSerializer: function (params) {
    return stringify(params, { arrayFormat: 'brackets' });
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

export async function getInitialState(): Promise<GlobalState> {
  var access = (await getAPI('/access')).data;
  return {
    signin: access.signin,
    id: access.id,
    permission: access.permission,
  };
}

export const qiankun = fetch('/api/plugin/entry')
  .then((rsp) => (rsp.status === 200 ? rsp.json() : { code: -1 }))
  .then((rsp) => {
    let data = [];
    if (rsp.code === 0)
      data = rsp.data.map((v: string) => {
        return {
          name: v,
          entry:
            process.env.NODE_ENV === 'production'
              ? `/_plugin/${v}`
              : `//localhost:8001`,
        };
      });
    return {
      apps: data,
      prefetch: false,
    };
  });
