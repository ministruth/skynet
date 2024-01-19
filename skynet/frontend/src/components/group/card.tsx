import Table from '@/components/layout/table';
import {
  checkAPI,
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
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { useAccess } from '@umijs/max';
import { Alert, Button } from 'antd';
import { Store } from 'antd/es/form/interface';
import type { SortOrder } from 'antd/es/table/interface';
import { Key, useRef, useState } from 'react';
import { FormattedMessage } from 'react-intl';
import confirm from '../layout/modal';
import {
  Columns,
  CreatedAtColumn,
  IDColumn,
  SearchColumn,
  UpdatedAtColumn,
} from '../layout/table/column';
import TableDelete from '../layout/table/deleteBtn';
import TableNew from '../layout/table/newBtn';
import styles from '../layout/table/style.less';
import Permission from '../permission/permBtn';
import GroupClone from './cloneBtn';
import GroupUpdate from './updateBtn';
import GroupUser from './userBtn';

export interface GroupBtnProps {
  tableRef: React.MutableRefObject<ActionType | undefined>;
  initialValues?: Store;
}

const handleDeleteSelected = async (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
  keys: Key[],
) => {
  confirm({
    title: intl.get('pages.group.op.delete.selected.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        deleleAPI('/groups', { id: keys }).then((rsp) => {
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

const request = async (
  params?: ParamsType,
  sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI('/groups', {
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

export const GroupColumns: Columns = (intl) => [
  {
    title: intl.get('pages.group.table.name'),
    dataIndex: 'name',
    tooltip: intl.get('pages.group.form.name.tip'),
    fieldProps: {
      maxLength: 32,
    },
    formItemProps: {
      rules: [{ required: true }],
    },
  },
  {
    title: intl.get('pages.group.table.note'),
    dataIndex: 'note',
    valueType: 'textarea',
    initialValue: '',
    fieldProps: {
      showCount: true,
      maxLength: 256,
    },
  },
];

const addColumns: Columns = (intl) => [
  {
    renderFormItem: () => (
      <Alert
        message={intl.get('pages.group.op.add.content')}
        type="info"
        showIcon
      />
    ),
  },
  ...GroupColumns(intl),
];

const GroupCard = () => {
  const intl = getIntl();
  const ref = useRef<ActionType>();
  const access = useAccess();
  const columns: ProColumns[] = [
    SearchColumn(intl),
    IDColumn(intl),
    {
      title: intl.get('pages.group.table.name'),
      dataIndex: 'name',
      align: 'center',
      hideInSearch: true,
      ellipsis: true,
      onCell: () => {
        return {
          style: {
            maxWidth: 200,
          },
        };
      },
    },
    {
      title: intl.get('pages.group.table.note'),
      dataIndex: 'note',
      ellipsis: true,
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
          <GroupClone
            key="clone"
            tableRef={ref}
            initialValues={{
              base: row.id,
              baseName: row.name,
            }}
          />,
          <GroupUpdate
            key="update"
            tableRef={ref}
            initialValues={{
              ...row,
            }}
          />,
          <Permission key="perm" ugid={row.id} origin={false} />,
          <GroupUser key="user" gid={row.id} />,
          <TableDelete
            key="delete"
            permName="manage.user"
            perm={UserPerm.PermWrite}
            tableRef={ref}
            url={`/groups/${row.id}`}
            confirmTitle={intl.get('pages.group.op.delete.title', {
              name: row.name,
            })}
          />,
        ];
      },
    },
  ];

  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);
  const onSelectChange = (keys: Key[]) => {
    setSelectedRowKeys(keys);
  };
  const rowSelection = {
    selectedRowKeys,
    onChange: onSelectChange,
  };

  const handleAdd = async (params: ParamsType) => {
    if (await checkAPI(postAPI('/groups', params))) {
      ref.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };

  return (
    <ProCard bordered>
      <Table
        actionRef={ref}
        rowKey="id"
        rowSelection={rowSelection}
        tableAlertRender={false}
        request={request}
        columns={columns}
        action={[
          <TableNew
            permName="manage.user"
            perm={UserPerm.PermWrite}
            key="add"
            width={500}
            title={intl.get('pages.group.op.add.title')}
            schemaProps={{
              onFinish: handleAdd,
              columns: addColumns(intl),
            }}
          />,
          <Button
            key="delete"
            danger
            disabled={
              !checkPerm(access, 'manage.user', UserPerm.PermWrite) ||
              selectedRowKeys.length === 0
            }
            onClick={() => handleDeleteSelected(intl, ref, selectedRowKeys)}
          >
            <FormattedMessage id="app.op.delete" />
          </Button>,
        ]}
      />
    </ProCard>
  );
};

export default GroupCard;
