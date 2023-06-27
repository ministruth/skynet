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
import { LogoutOutlined } from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { useAccess, useModel } from '@umijs/max';
import { Button, Tag } from 'antd';
import { Store } from 'antd/es/form/interface';
import type { SortOrder } from 'antd/es/table/interface';
import { CustomTagProps } from 'rc-select/es/BaseSelect';
import { Key, useRef, useState } from 'react';
import { FormattedMessage } from 'react-intl';
import confirm from '../layout/modal';
import {
  Column,
  Columns,
  CreatedAtColumn,
  IDColumn,
  SearchColumn,
} from '../layout/table/column';
import TableDelete from '../layout/table/deleteBtn';
import TableNew from '../layout/table/newBtn';
import styles from '../layout/table/style.less';
import TableBtn from '../layout/table/tableBtn';
import AvatarUpload from './avatar';
import UserClone from './cloneBtn';
import UserPermBtn from './permBtn';
import UserUpdate from './updateBtn';

export interface UserBtnProps {
  tableRef: React.MutableRefObject<ActionType | undefined>;
  initialValues?: Store;
}

const handleDeleteSelected = async (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
  keys: Key[],
) => {
  confirm({
    title: intl.get('pages.user.op.delete.selected.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        deleleAPI('/user', { id: keys }).then((rsp) => {
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

const handleKick = (
  intl: StringIntl,
  id: string,
  name: string,
  refresh: boolean,
) => {
  confirm({
    title: intl.get('pages.user.op.kick.title', {
      username: name,
    }),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI(`/user/${id}/kick`, {}).then((rsp) => {
          if (rsp && rsp.code === 0) {
            resolve(rsp);
            if (refresh) window.location.href = '/user';
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
  const msg = await getAPI('/user', {
    lastLoginSort: paramSort(sort?.last_login),
    text: params?.text,
    lastLoginStart: paramTime(params?.lastLoginStart),
    lastLoginEnd: paramTime(params?.lastLoginEnd),
    createdSort: paramSort(sort?.created_at),
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

export const UserColumns: Columns = (intl) => [
  {
    title: intl.get('pages.user.table.username'),
    dataIndex: 'username',
    tooltip: intl.get('pages.user.form.username.tip'),
    fieldProps: {
      maxLength: 32,
    },
    formItemProps: {
      rules: [{ required: true }],
    },
  },
  {
    title: intl.get('pages.user.table.password'),
    dataIndex: 'password',
    valueType: 'password',
    formItemProps: {
      rules: [{ required: true }],
    },
  },
];

export const GroupColumn: Column = (intl) => {
  return {
    title: intl.get('pages.user.form.group'),
    tooltip: intl.get('pages.user.form.group.tip'),
    dataIndex: 'group',
    fieldProps: {
      mode: 'multiple',
      showSearch: true,
      tagRender: (props: CustomTagProps) => {
        return (
          <Tag
            color={props.label === 'root' ? 'red' : undefined}
            closable={props.closable}
            onClose={props.onClose}
            style={{ marginRight: 4 }}
          >
            {props.label}
          </Tag>
        );
      },
    },
    request: async ({ keyWords }: any) => {
      const msg = await getAPI(
        '/group',
        {
          name: keyWords,
          page: 1,
          size: 5,
        },
        false,
      );
      return msg.data.data.map((e: any) => ({ value: e.id, label: e.name }));
    },
  };
};

export const AvatarColumn: Column = (intl) => {
  return {
    title: intl.get('pages.user.table.avatar'),
    dataIndex: 'avatar',
    renderFormItem: () => {
      return <AvatarUpload />;
    },
  };
};

export const AddColumns: Columns = (intl) => [
  {
    renderFormItem: () => <FormattedMessage id="pages.user.op.add.content" />,
  },
  ...UserColumns(intl),
  AvatarColumn(intl),
  GroupColumn(intl),
];

const UserCard = () => {
  const intl = getIntl();
  const ref = useRef<ActionType>();
  const { initialState } = useModel('@@initialState');
  const access = useAccess();
  const columns: ProColumns[] = [
    SearchColumn(intl),
    IDColumn(intl),
    {
      title: intl.get('pages.user.table.avatar'),
      dataIndex: 'avatar',
      valueType: 'avatar',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('pages.user.table.username'),
      dataIndex: 'username',
      align: 'center',
      hideInSearch: true,
      ellipsis: true,
      onCell: () => {
        return {
          style: {
            maxWidth: 150,
          },
        };
      },
    },
    {
      title: intl.get('pages.user.table.lastip'),
      dataIndex: 'last_ip',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('pages.user.table.lastlogin'),
      dataIndex: 'last_login',
      align: 'center',
      valueType: 'dateTime',
      sorter: true,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.user.table.lastlogin'),
      dataIndex: 'last_login',
      valueType: 'dateRange',
      hideInTable: true,
      search: {
        transform: (value) => {
          return {
            lastLoginStart: value[0],
            lastLoginEnd: value[1],
          };
        },
      },
    },
    ...CreatedAtColumn(intl),
    {
      title: intl.get('app.op'),
      valueType: 'option',
      align: 'center',
      width: 100,
      className: styles.operation,
      render: (_, row) => {
        return [
          <UserClone
            key="clone"
            tableRef={ref}
            initialValues={{
              base: row.id,
              baseName: row.username,
            }}
          />,
          <UserUpdate
            key="update"
            tableRef={ref}
            initialValues={{
              ...row,
            }}
          />,
          <UserPermBtn key="perm" uid={row.id} />,
          <TableBtn
            key="kick"
            icon={LogoutOutlined}
            tip={intl.get('pages.user.op.kick.tip')}
            color="#faad14"
            perm={UserPerm.PermExecute}
            permName="manage.user"
            onClick={() =>
              handleKick(
                intl,
                row.id,
                row.username,
                row.id === initialState?.id,
              )
            }
          />,
          <TableDelete
            key="delete"
            permName="manage.user"
            perm={UserPerm.PermWriteExecute}
            tableRef={ref}
            url={`/user/${row.id}`}
            confirmTitle={intl.get('pages.user.op.delete.title', {
              username: row.username,
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
    if (await checkAPI(postAPI('/user', params))) {
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
            title={intl.get('pages.user.op.add.title')}
            schemaProps={{
              onFinish: handleAdd,
              columns: AddColumns(intl),
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

export default UserCard;
