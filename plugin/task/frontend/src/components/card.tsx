import confirm from '@/common_components/layout/modal';
import Table from '@/common_components/layout/table';
import {
  CreatedAtColumn,
  IDColumn,
  SearchColumn,
  UpdatedAtColumn,
} from '@/common_components/layout/table/column';
import styles from '@/common_components/layout/table/style.less';
import TableBtn from '@/common_components/layout/table/tableBtn';
import { API_PREFIX } from '@/config';
import {
  checkPerm,
  deleleAPI,
  getAPI,
  getIntl,
  paramSort,
  paramTime,
  postAPI,
  StringIntl,
  UserPerm,
} from '@/utils';
import { StopOutlined } from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-components';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { useModel } from '@umijs/max';
import { Button } from 'antd';
import type { SortOrder } from 'antd/es/table/interface';
import Paragraph from 'antd/es/typography/Paragraph';
import { useRef } from 'react';
import { FormattedMessage } from 'react-intl';
import TaskOutput from './output';
import custom_styles from './style.less';

const request = async (
  params?: ParamsType,
  sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI(`${API_PREFIX}/tasks`, {
    created_sort: paramSort(sort?.created_at) || 'desc',
    updated_sort: paramSort(sort?.updated_at),
    text: params?.text,
    created_start: paramTime(params?.createdStart),
    created_end: paramTime(params?.createdEnd, true),
    updated_start: paramTime(params?.updatedStart),
    updated_end: paramTime(params?.updatedEnd, true),
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
    title: intl.get('pages.task.op.deleteall.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        deleleAPI(`${API_PREFIX}/tasks`, {}).then((rsp) => {
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

const handleStop = (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
  id: string,
  name: string,
) => {
  confirm({
    title: intl.get('pages.task.op.stop.title', {
      name: name,
    }),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI(`${API_PREFIX}/tasks/${id}/stop`, {}).then((rsp) => {
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

const TaskCard = () => {
  const intl = getIntl();
  const ref = useRef<ActionType>();
  const { access } = useModel('@@qiankunStateFromMaster');

  const getStatus = (item: any) => {
    if (item.result !== undefined) {
      if (item.result != 0) return 'exception';
      if (item.percent != 100) return 'normal';
    }
    if (item.percent == 100) return 'success';
    return 'active';
  };

  const columns: ProColumns[] = [
    SearchColumn(intl),
    IDColumn(intl),
    {
      title: intl.get('pages.task.table.name'),
      dataIndex: 'name',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('pages.task.table.result'),
      dataIndex: 'result',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('pages.task.table.percent'),
      valueType: (item) => ({
        type: 'progress',
        status: getStatus(item),
      }),
      dataIndex: 'percent',
      align: 'center',
      hideInSearch: true,
    },
    ...CreatedAtColumn(intl),
    ...UpdatedAtColumn(intl),
    {
      title: intl.get('app.op'),
      valueType: 'option',
      align: 'center',
      className: styles.operation,
      width: 100,
      render: (_, row) => {
        return [
          <TaskOutput id={row.id} key="output" />,
          <TableBtn
            key="stop"
            icon={StopOutlined}
            tip={intl.get('pages.task.op.stop.tip')}
            color="#ff4d4f"
            perm={UserPerm.PermWrite}
            permName="manage.plugin"
            onClick={() => handleStop(intl, ref, row.id, row.name)}
            disabled={row.result != undefined}
          />,
        ];
      },
    },
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
              !checkPerm(access, 'manage.notification', UserPerm.PermWrite)
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
                <pre className={custom_styles.detail}>{record.detail}</pre>
              </Paragraph>
            );
          },
        }}
      />
    </ProCard>
  );
};

export default TaskCard;
