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
import { Alert, Button, Tag } from 'antd';
import { Store } from 'antd/es/form/interface';
import type { SortOrder } from 'antd/es/table/interface';
import { CustomTagProps } from 'rc-select/es/BaseSelect';
import { Key, useRef, useState } from 'react';
import { FormattedMessage } from 'react-intl';
import GeoIP from '../geoip';
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
import Permission from '../permission/permBtn';
import AvatarUpload from './avatar';
import UserClone from './cloneBtn';
import HistoryBtn from './historyBtn';
import UserUpdate from './updateBtn';

export interface UserBtnProps {
  tableRef: React.MutableRefObject<ActionType | undefined>;
  initialValues?: Store;
  disabled?: boolean;
}

const handleDeleteSelected = async (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
  keys: Key[],
) => {
  confirm({
    title: intl.get('pages.user.delete.selected.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        deleleAPI('/users', { id: keys }).then((rsp) => {
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
    title: intl.get('pages.user.kick.title', {
      username: name,
    }),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI(`/users/${id}/kick`, {}).then((rsp) => {
          if (rsp && rsp.code === 0) {
            resolve(rsp);
            if (refresh) window.location.reload();
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
  const msg = await getAPI('/users', {
    login_sort: paramSort(sort?.last_login),
    text: params?.text,
    login_start: paramTime(params?.lastLoginStart),
    login_end: paramTime(params?.lastLoginEnd, true),
    created_sort: paramSort(sort?.created_at) || 'desc',
    created_start: paramTime(params?.createdStart),
    created_end: paramTime(params?.createdEnd, true),
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
    title: intl.get('tables.username'),
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
    title: intl.get('tables.password'),
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
      labelInValue: true,
      tagRender: (props: CustomTagProps) => {
        // BUG: rc-select undefined value
        if (props.value)
          return (
            <Tag
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
        '/groups',
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
    title: intl.get('tables.avatar'),
    dataIndex: 'avatar',
    renderFormItem: () => {
      return <AvatarUpload />;
    },
  };
};

export const AddColumns: Columns = (intl) => [
  {
    renderFormItem: () => (
      <Alert
        message={intl.get('pages.user.add.content')}
        type="info"
        showIcon
      />
    ),
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
      title: intl.get('tables.avatar'),
      dataIndex: 'avatar',
      valueType: 'avatar',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('tables.username'),
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
      title: intl.get('tables.lastip'),
      dataIndex: 'last_ip',
      align: 'center',
      hideInSearch: true,
      render: (_, row) => <GeoIP value={row.last_ip} />,
    },
    {
      title: intl.get('tables.lastlogin'),
      dataIndex: 'last_login',
      align: 'center',
      valueType: 'dateTime',
      sorter: true,
      hideInSearch: true,
    },
    {
      title: intl.get('tables.lastlogin'),
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
      width: 150,
      className: styles.operation,
      render: (_, row) => {
        let root = row.id === '00000000-0000-0000-0000-000000000000';
        let self_root =
          initialState?.id === '00000000-0000-0000-0000-000000000000';
        return [
          <UserClone
            key="clone"
            tableRef={ref}
            initialValues={{
              base: row.id,
              baseName: row.username,
            }}
            disabled={root}
          />,
          <UserUpdate
            key="update"
            tableRef={ref}
            initialValues={{
              ...row,
            }}
            disabled={root && !self_root}
          />,
          <Permission
            key="perm"
            ugid={row.id}
            origin={true}
            refresh={row.id == initialState?.id}
            disabled={root}
          />,
          <HistoryBtn key="history" uid={row.id} />,
          <TableBtn
            key="kick"
            icon={LogoutOutlined}
            tip={intl.get('pages.user.kick.tip')}
            color="#faad14"
            perm={UserPerm.PermWrite}
            permName="manage.user"
            onClick={() =>
              handleKick(
                intl,
                row.id,
                row.username,
                row.id === initialState?.id,
              )
            }
            disabled={root && !self_root}
          />,
          <TableDelete
            key="delete"
            permName="manage.user"
            perm={UserPerm.PermWrite}
            tableRef={ref}
            url={`/users/${row.id}`}
            confirmTitle={intl.get('pages.user.delete.title', {
              username: row.username,
            })}
            disabled={root && !self_root}
            refresh={row.id === initialState?.id}
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
    getCheckboxProps: (rec: { id: string }) => {
      if (rec.id === '00000000-0000-0000-0000-000000000000')
        return { disabled: true };
      return {};
    },
  };
  const handleAdd = async (params: ParamsType) => {
    params['group'] = params['group']?.map((x: { value: string }) => x.value);
    if (await checkAPI(postAPI('/users', params))) {
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
            title={intl.get('pages.user.add.title')}
            schemaProps={{
              onFinish: handleAdd,
              columns: AddColumns(intl),
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

export default UserCard;
