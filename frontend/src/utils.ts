import { message } from 'antd';
import { SortOrder } from 'antd/lib/table/interface';
import { PrimitiveType } from 'intl-messageformat';
import moment from 'moment';
import { IntlShape, useIntl } from 'react-intl';
import { request } from 'umi';

export enum UserPerm {
  PermNone = 0,
  PermExecute = 1,
  PermWrite = 1 << 1,
  PermRead = 1 << 2,
  PermAll = 1 << (3 - 1),
  PermWriteExecute = PermWrite | PermExecute,
}

export function checkPerm(
  access: { [Key: string]: UserPerm },
  name: string,
  perm: UserPerm,
) {
  if (access[name] !== undefined) {
    return (access[name] & perm) === perm;
  }
  if ((access['all'] & perm) === perm) return true;
  return name === 'user' || name === 'guest';
}

export class StringIntl {
  intl: IntlShape;
  constructor(intl: IntlShape) {
    this.intl = intl;
  }
  get(id: string, values?: Record<string, PrimitiveType>) {
    return this.intl.formatMessage(
      {
        id: id,
      },
      values,
    );
  }
}

export function getIntl() {
  return new StringIntl(useIntl());
}

export function ping() {
  return request('/ping', {
    method: 'get',
    skipErrorHandler: true,
    errorHandler: (e) => {},
  })
    .then((rsp) => {
      if (rsp) return rsp.code === 0;
      return false;
    })
    .catch((rsp) => {
      return false;
    });
}

export function getAPI(url: string, params?: object, showmsg: boolean = false) {
  return request(url, {
    method: 'get',
    params: params,
  }).then((rsp) => {
    if (rsp) {
      if (showmsg) message.success(rsp.msg);
      return rsp;
    }
  });
}

export function withToken(
  method: 'post' | 'delete' | 'put',
  url: string,
  data: any,
  showmsg: boolean = true,
) {
  return request('/token', {
    method: 'get',
  }).then((rsp) => {
    if (rsp)
      return request(url, {
        method: method,
        data: data,
        headers: {
          'X-CSRF-Token': rsp.data,
        },
      }).then((rsp) => {
        if (rsp) {
          if (showmsg) message.success(rsp.msg);
          return rsp;
        }
      });
  });
}

export function postAPI(url: string, data: any, showmsg: boolean = true) {
  return withToken('post', url, data, showmsg);
}

export function putAPI(url: string, data: any, showmsg: boolean = true) {
  return withToken('put', url, data, showmsg);
}

export function deleleAPI(url: string, data: any, showmsg: boolean = true) {
  return withToken('delete', url, data, showmsg);
}

export async function checkAPI(ret: Promise<any>) {
  return await ret.then((rsp) => {
    return rsp && rsp.code === 0;
  });
}

export function paramSort(v?: SortOrder) {
  if (v === 'ascend') return 'asc';
  else if (v === 'descend') return 'desc';
  return undefined;
}

export function paramTime(v?: string) {
  return moment(v || 0).valueOf() || undefined;
}

export const fileToBase64 = (file: File) =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => {
      let res = reader.result as string;
      resolve(res.split(',')[1]);
    };
    reader.onerror = (error) => reject(error);
  });
