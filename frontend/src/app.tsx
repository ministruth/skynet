import { getAPI } from '@/utils';
import { PageLoading } from '@ant-design/pro-layout';
import { message } from 'antd';
import 'antd/dist/antd.css';
import _ from 'lodash';
import { stringify } from 'qs';
import { ErrorShowType, getLocale, RequestConfig } from 'umi';
import { ResponseError } from 'umi-request';

const errorHandler = (error: ResponseError) => {
  if (error.name == 'BizError') {
    message.error(error.data.msg);
  } else if (error.response.status != 200) {
    if (_.isEmpty(error.data)) message.error(error.response.statusText);
    else message.error(error.data);
  }
};

export const request: RequestConfig = {
  errorHandler,
  prefix: '/api',
  timeout: 3000,
  params: {
    lang: getLocale(),
  },
  paramsSerializer: function (params) {
    return stringify(params, { arrayFormat: 'brackets' });
  },
  timeoutMessage: 'Request timeout',
  errorConfig: {
    adaptor: (data) => {
      return {
        ...data,
        success: data.code === 0,
        errorMessage: data.msg,
      };
    },
  },
  showType: ErrorShowType.ERROR_MESSAGE,
};

export async function getInitialState(): Promise<{
  signin: boolean;
  permission: { [Key: string]: any };
  menu: [{ [Key: string]: any }];
}> {
  var access = (await getAPI('/access')).data;
  var menu = access.signin ? (await getAPI('/menu')).data : [];
  return {
    signin: access.signin,
    permission: access.permission,
    menu: menu,
  };
}

export const initialStateConfig = {
  loading: <PageLoading />,
};

export const qiankun = fetch('/api/plugin/entry')
  .then((rsp) => (rsp.status == 200 ? rsp.json() : { code: -1 }))
  .then((rsp) => {
    let data = [];
    if (rsp.code == 0)
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
