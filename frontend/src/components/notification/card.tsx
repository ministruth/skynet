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
import { useAccess, useModel } from '@umijs/max';
import { Button, Tag } from 'antd';
import type { SortOrder } from 'antd/es/table/interface';
import Paragraph from 'antd/es/typography/Paragraph';
import { CustomTagProps } from 'rc-select/es/BaseSelect';
import { useRef } from 'react';
import { FormattedMessage } from 'react-intl';
import confirm from '../layout/modal';
import Table from '../layout/table';
import {
  CreatedAtColumn,
  IDColumn,
  SearchColumn,
} from '../layout/table/column';
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
    title: intl.get('pages.notification.op.deleteall.title'),
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
    SearchColumn(intl),
    IDColumn(intl),
    {
      title: intl.get('pages.notification.table.name'),
      dataIndex: 'name',
      align: 'center',
      hideInSearch: true,
      onCell: () => {
        return {
          style: {
            maxWidth: 150,
          },
        };
      },
    },
    {
      title: intl.get('pages.notification.table.level'),
      dataIndex: 'level',
      align: 'center',
      valueType: 'select',
      fieldProps: {
        mode: 'multiple',
        tagRender: (props: CustomTagProps) => {
          return (
            <Tag
              color={levelEnum[props.value].color}
              closable={props.closable}
              onClose={props.onClose}
              style={{ marginRight: 4 }}
            >
              {props.label}
            </Tag>
          );
        },
      },
      valueEnum: Object.entries(levelEnum).reduce(
        (p, c) => ({ ...p, [c[0]]: { text: c[1].label } }),
        {},
      ),
      render: (_, row) => (
        <Tag style={{ marginRight: 0 }} color={levelEnum[row.level].color}>
          {levelEnum[row.level].label}
        </Tag>
      ),
    },
    {
      title: intl.get('pages.notification.table.message'),
      dataIndex: 'message',
      align: 'center',
      hideInSearch: true,
      onCell: () => {
        return {
          style: {
            maxWidth: 350,
          },
        };
      },
    },
    ...CreatedAtColumn(intl),
  ];

  return (
    <ProCard bordered>
      <Table
        actionRef={ref}
        poll
        rowKey="id"
        request={request}
        columns={columns}
        action={[
          <Button
            key="delete"
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
            <FormattedMessage id="app.op.deleteall" />
          </Button>,
        ]}
        expandable={{
          expandRowByClick: true,
          expandedRowRender: (record: any) => {
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
