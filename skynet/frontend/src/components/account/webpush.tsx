import {
  checkAPI,
  deleleAPI,
  getAPI,
  getIntl,
  postAPI,
  putAPI,
  StringIntl,
} from '@/utils';
import { ParamsType } from '@ant-design/pro-components';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { FormattedMessage, history, useModel } from '@umijs/max';
import { Button, message } from 'antd';
import { SortOrder } from 'antd/es/table/interface';
import { useEffect, useRef, useState } from 'react';
import Table from '../layout/table';
import { IDColumn } from '../layout/table/column';
import styles from './style.less';

const handleSubscribe = async (
  intl: StringIntl,
  key: string,
  enable: boolean,
) => {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    message.error(intl.get('pages.account.webpush.unsupport'));
    return;
  }
  if (enable) {
    return navigator.serviceWorker
      .register('/sw.js')
      .then(() =>
        navigator.serviceWorker.ready
          .then((registration) =>
            registration.pushManager
              .getSubscription()
              .then((s) => (s ? s.unsubscribe() : s))
              .then(() =>
                registration.pushManager.subscribe({
                  userVisibleOnly: true,
                  applicationServerKey: key,
                }),
              ),
          )
          .then((s) => postAPI('/users/self/webpush', s))
          .then(() => {
            navigator.serviceWorker.addEventListener('message', (event) => {
              if (!event.data.action) {
                return;
              }
              switch (event.data.action) {
                case 'skynet-click':
                  history.push(event.data.url);
                  break;
              }
            });
          }),
      )
      .catch(() => {
        message.error(intl.get('pages.account.webpush.failed'));
      });
  } else {
    return navigator.serviceWorker.getRegistrations().then((registrations) => {
      for (const registration of registrations) {
        registration.pushManager.getSubscription().then(async (s) => {
          if (s) {
            await deleleAPI('/users/self/webpush', {
              endpoint: s.endpoint,
            });
            s.unsubscribe();
          }
        });
        registration.unregister();
      }
    });
  }
};

const request = async (
  params?: ParamsType,
  _sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI('/users/self/webpush', {
    page: params?.current,
    size: params?.pageSize,
  });
  return {
    data: msg.data.data,
    success: true,
    total: msg.data.total,
  };
};

const handleStatus = async (
  id: string,
  enable: boolean,
  ref: React.MutableRefObject<ActionType | undefined>,
) => {
  if (
    await checkAPI(
      putAPI('/users/self/webpush', {
        id: id,
        enable: enable,
      }),
    )
  )
    ref.current?.reloadAndRest?.();
};

const WebpushList = () => {
  const intl = getIntl();
  const { setting, getSetting } = useModel('setting');
  const [subscribe, setSubscribe] = useState(false);
  useEffect(() => {
    getSetting();
    check();
  }, []);
  const ref = useRef<ActionType>();

  const check = () => {
    if ('serviceWorker' in navigator && 'PushManager' in window) {
      navigator.serviceWorker.ready.then((registration) =>
        registration.pushManager.getSubscription().then(async (s) => {
          if (s) {
            const data = await postAPI(
              '/users/self/webpush/check',
              {
                endpoint: s.endpoint,
              },
              false,
            );
            setSubscribe(data.data);
          } else {
            setSubscribe(false);
          }
        }),
      );
    }
  };

  const columns: ProColumns[] = [
    IDColumn(intl),
    {
      title: intl.get('tables.name'),
      dataIndex: 'name',
      align: 'center',
      ellipsis: true,
      copyable: true,
      onCell: () => {
        return {
          style: {
            minWidth: 150,
            maxWidth: 150,
          },
        };
      },
    },
    {
      title: intl.get('app.op'),
      dataIndex: 'enable',
      align: 'center',
      render: (_, row) => {
        if (row.enable)
          return (
            <Button onClick={() => handleStatus(row.id, false, ref)}>
              <FormattedMessage id="pages.account.webpush.disable" />
            </Button>
          );
        else
          return (
            <Button
              type="primary"
              onClick={() => handleStatus(row.id, true, ref)}
              ghost
            >
              <FormattedMessage id="pages.account.webpush.enable" />
            </Button>
          );
      },
    },
  ];
  let action = [];
  if (!subscribe)
    action.push(
      <Button
        key="subscribe"
        type="primary"
        onClick={() => {
          handleSubscribe(intl, setting['webpush.key'], true).then(() =>
            setTimeout(() => {
              check();
            }, 200),
          );
        }}
      >
        <FormattedMessage id="pages.account.subscribe" />
      </Button>,
    );
  else
    action.push(
      <Button
        key="unsubscribe"
        onClick={() => {
          handleSubscribe(intl, setting['webpush.key'], false).then(() =>
            setTimeout(() => {
              check();
            }, 200),
          );
        }}
        danger
      >
        <FormattedMessage id="pages.account.unsubscribe" />
      </Button>,
    );

  return (
    <Table
      rowKey="name"
      actionRef={ref}
      className={styles['column-padding-start']}
      headerTitle={intl.get('pages.account.webpush.title')}
      options={false}
      search={false}
      request={request}
      columns={columns}
      cardBordered={false}
      action={action}
      pagination={{
        defaultPageSize: 5,
      }}
    />
  );
};

export default WebpushList;
