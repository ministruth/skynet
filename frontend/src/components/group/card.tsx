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
import { useAccess, useModel } from '@umijs/max';
import { Button } from 'antd';
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
import GroupClone from './cloneBtn';
import GroupPerm from './permBtn';
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
        deleleAPI('/group', { id: keys }).then((rsp) => {
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
  const msg = await getAPI('/group', {
    createdSort: paramSort(sort?.created_at),
    updatedSort: paramSort(sort?.updated_at),
    text: params?.text,
    createdStart: paramTime(params?.createdStart),
    createdEnd: paramTime(params?.createdEnd),
    updatedStart: paramTime(params?.updatedStart),
    updatedEnd: paramTime(params?.updatedEnd),
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
    fieldProps: {
      showCount: true,
      maxLength: 256,
    },
  },
];

const addColumns: Columns = (intl) => [
  {
    renderFormItem: () => <FormattedMessage id="pages.group.op.add.content" />,
  },
  ...GroupColumns(intl),
];

const GroupCard = () => {
  const { initialState } = useModel('@@initialState');
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
        const root = row.name === 'root';
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
          <GroupPerm key="perm" disableModify={root} gid={row.id} />,
          <GroupUser key="user" gid={row.id} />,
          <TableDelete
            key="delete"
            disabled={root}
            permName="manage.user"
            perm={UserPerm.PermWriteExecute}
            tableRef={ref}
            url={`/group/${row.id}`}
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
    getCheckboxProps: (rec: { name: string }) => {
      if (rec.name === 'root') return { disabled: true };
      return {};
    },
  };

  const handleAdd = async (params: ParamsType) => {
    if (await checkAPI(postAPI('/group', params))) {
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
            perm={UserPerm.PermWriteExecute}
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
              !checkPerm(
                initialState?.signin,
                access,
                'manage.user',
                UserPerm.PermWriteExecute,
              ) || selectedRowKeys.length === 0
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
