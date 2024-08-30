import { getLocale, request, useIntl } from "@umijs/max";
import { message } from "antd";
import { SortOrder } from "antd/es/table/interface";
import { PrimitiveType } from "intl-messageformat";
import Cookies from "js-cookie";
import moment from "moment";
import { IntlShape } from "react-intl";

export enum UserPerm {
  PermBan = -1,
  PermInherit = -1,
  PermNone = 0,
  PermWrite = 1,
  PermRead = 1 << 1,
  PermAll = (1 << 2) - 1,
}

export function checkPerm(
  access: { [Key: string]: UserPerm },
  name: string,
  perm: UserPerm
) {
  if (access["root"] !== undefined) return true;
  if (access[name] !== undefined) return (access[name] & perm) === perm;
  return false;
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
      values
    );
  }
}

export function getIntl() {
  return new StringIntl(useIntl());
}

export function api(
  method: string,
  url: string,
  params?: object,
  data?: any,
  headers?: any,
  showmsg: boolean = false
) {
  let obj = {
    lang: getLocale(),
  };
  if (params === undefined) params = obj;
  else Object.assign(params, obj);
  return request(url, {
    method: method,
    params: params,
    data: data,
    headers: headers,
  }).then((rsp) => {
    if (rsp) {
      if (showmsg) {
        if (rsp.code == 0) message.success(rsp.message);
        else message.error(rsp.message);
      }
      return rsp;
    }
  });
}

export function withToken(
  method: string,
  url: string,
  params?: object,
  data?: any,
  showmsg: boolean = false
) {
  return request("/token", {
    method: "get",
  }).then((rsp) => {
    if (rsp)
      return api(
        method,
        url,
        params,
        data,
        {
          "X-CSRF-Token": Cookies.get("CSRF_TOKEN"),
        },
        showmsg
      );
  });
}

export function getAPI(url: string, params?: object, showmsg: boolean = false) {
  return api("get", url, params, undefined, undefined, showmsg);
}

export function postAPI(url: string, data: any, showmsg: boolean = true) {
  return withToken("post", url, undefined, data, showmsg);
}

export function putAPI(url: string, data: any, showmsg: boolean = true) {
  return withToken("put", url, undefined, data, showmsg);
}

export function deleleAPI(url: string, data: any, showmsg: boolean = true) {
  return withToken("delete", url, undefined, data, showmsg);
}

export async function checkAPI(ret: Promise<any>) {
  return await ret.then((rsp) => {
    return rsp && rsp.code === 0;
  });
}

export function paramSort(v?: SortOrder) {
  if (v === "ascend") return "asc";
  else if (v === "descend") return "desc";
  return undefined;
}

export function paramTime(v?: string, end?: boolean) {
  return moment(v || 0).valueOf() + (v && end ? 86400000 : 0) || undefined;
}

export const fileToBase64 = (file: File | undefined) =>
  new Promise((resolve: (value: string) => void, reject) => {
    if (file === undefined) {
      resolve("");
    } else {
      const reader = new FileReader();
      reader.readAsDataURL(file);
      reader.onload = () => {
        resolve(reader.result as string);
      };
      reader.onerror = (error) => reject(error);
    }
  });
