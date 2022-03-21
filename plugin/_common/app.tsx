import { PageLoading } from '@ant-design/pro-layout';
import { message } from 'antd';
import 'antd/dist/antd.css';
import _ from 'lodash';
import { stringify } from 'qs';
import { ErrorShowType, getLocale, RequestConfig } from 'umi';
import { ResponseError } from 'umi-request';
import { PLUGIN_ID } from './config';

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
  prefix: `/api/plugin/${PLUGIN_ID}`,
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

export const initialStateConfig = {
  loading: <PageLoading />,
};
