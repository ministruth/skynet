import Table from '@/components/layout/table';
import {
  checkPerm,
  deleleAPI,
  getAPI,
  getIntl,
  paramSort,
  paramTime,
  StringIntl,
  UserPerm,
} from '@/utils';
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { Button, Tag } from 'antd';
import type { SortOrder } from 'antd/lib/table/interface';
import Paragraph from 'antd/lib/typography/Paragraph';
import { useRef } from 'react';
import { FormattedMessage } from 'react-intl';
import { useAccess, useModel } from 'umi';
import confirm from '../layout/modal';
import styles from './style.less';

const request = async (
  params?: ParamsType,
  sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI('/notification', {
    createdSort: paramSort(sort?.created_at) || 'desc',
    level: params?.level,
    text: params?.text,
    createdStart: paramTime(params?.createdStart),
    createdEnd: paramTime(params?.createdEnd),
    page: params?.current,
    size: params?.pageSize,
  });
  return {
    data: msg.data.data,
    success: true,
    total: msg.data.total,
  };
};

const handleDeleteAll = async (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
) => {
  confirm({
    title: intl.get('pages.notification.table.deleteall.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        deleleAPI('/notification', {}).then((rsp) => {
          if (rsp && rsp.code === 0) {
            ref.current?.reloadAndRest?.();
            resolve(rsp);
          } else {
            reject(rsp);
          }
        });
      });
    },
    intl: intl,
  });
};

const NotificationCard = () => {
  const intl = getIntl();
  const { initialState } = useModel('@@initialState');
  const ref = useRef<ActionType>();
  const access = useAccess();
  const levelEnum: { [Key: number]: { label: string; color: string } } = {
    0: {
      label: 'Info',
      color: 'processing',
    },
    1: {
      label: 'Success',
      color: 'success',
    },
    2: {
      label: 'Warning',
      color: 'warning',
    },
    3: {
      label: 'Error',
      color: 'error',
    },
    4: {
      label: 'Fatal',
      color: 'volcano',
    },
  };

  const columns: ProColumns[] = [
    {
      title: intl.get('app.table.id'),
      ellipsis: true,
      dataIndex: 'id',
      align: 'center',
      copyable: true,
      hideInSearch: true,
      width: 150,
    },
    {
      title: intl.get('pages.notification.table.name'),
      dataIndex: 'name',
      align: 'center',
      width: 150,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.notification.table.level'),
      dataIndex: 'level',
      align: 'center',
      valueType: 'checkbox',
      colSize: 3,
      valueEnum: Object.entries(levelEnum).reduce(
        (p, c) => ({ ...p, [c[0]]: { text: c[1].label } }),
        {},
      ),
      width: 100,
      render: (_, row) => (
        <Tag style={{ marginRight: 0 }} color={levelEnum[row.level].color}>
          {levelEnum[row.level].label}
        </Tag>
      ),
    },
    {
      title: intl.get('app.table.searchtext'),
      key: 'text',
      hideInTable: true,
    },
    {
      title: intl.get('pages.notification.table.message'),
      dataIndex: 'message',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('app.table.createdat'),
      dataIndex: 'created_at',
      align: 'center',
      width: 180,
      valueType: 'dateTime',
      sorter: true,
      hideInSearch: true,
    },
    {
      title: intl.get('app.table.createdat'),
      dataIndex: 'created_at',
      valueType: 'dateRange',
      hideInTable: true,
      search: {
        transform: (value) => {
          return {
            createdStart: value[0],
            createdEnd: value[1],
          };
        },
      },
    },
  ];

  return (
    <ProCard>
      <Table
        actionRef={ref}
        poll
        rowKey="id"
        request={(params, sort) => request(params, sort)}
        columns={columns}
        action={[
          <Button
            key="delete"
            type="ghost"
            danger
            onClick={() => handleDeleteAll(intl, ref)}
            disabled={
              !checkPerm(
                initialState?.signin,
                access,
                'manage.notification',
                UserPerm.PermWriteExecute,
              )
            }
          >
            <FormattedMessage id="app.table.deleteall" />
          </Button>,
        ]}
        expandable={{
          expandRowByClick: true,
          expandedRowRender: (record) => {
            return (
              <Paragraph>
                <pre className={styles.detail}>
                  {JSON.stringify(JSON.parse(record.detail), null, 2)}
                </pre>
              </Paragraph>
            );
          },
        }}
      />
    </ProCard>
  );
};

export default NotificationCard;
