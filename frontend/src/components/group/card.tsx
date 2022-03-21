import Table from '@/components/layout/table';
import {
  checkAPI,
  checkPerm,
  getAPI,
  getIntl,
  paramSort,
  paramTime,
  postAPI,
  UserPerm,
} from '@/utils';
import { ProfileOutlined } from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import type { SortOrder } from 'antd/lib/table/interface';
import { Store } from 'rc-field-form/lib/interface';
import { useRef } from 'react';
import { FormattedMessage } from 'react-intl';
import { useAccess } from 'umi';
import {
  Columns,
  CreatedAtColumn,
  UpdatedAtColumn,
} from '../layout/table/column';
import TableDelete from '../layout/table/deleteBtn';
import TableNew from '../layout/table/newBtn';
import GroupClone from './cloneBtn';
import GroupUpdate from './updateBtn';

export interface GroupBtnProps {
  tableRef: React.MutableRefObject<ActionType | undefined>;
  initialValues?: Store;
}

const handleAdd = (params: ParamsType) => checkAPI(postAPI('/group', params));

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
    data: msg.data,
    success: true,
    total: msg.total,
  };
};

export const groupColumns: Columns = (intl) => [
  {
    title: intl.get('pages.group.table.name'),
    dataIndex: 'name',
    tooltip: intl.get('pages.group.table.add.nametip'),
    formItemProps: {
      rules: [
        {
          required: true,
          message: intl.get('app.table.required'),
        },
      ],
    },
  },
  {
    title: intl.get('pages.group.table.note'),
    dataIndex: 'note',
  },
];

const addColumns: Columns = (intl) => [
  {
    renderFormItem: () => (
      <FormattedMessage id="pages.group.table.add.content" />
    ),
  },
  ...groupColumns(intl),
];

const GroupCard = () => {
  const intl = getIntl();
  const ref = useRef<ActionType>();
  const access = useAccess();
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
      title: intl.get('pages.group.table.name'),
      dataIndex: 'name',
      align: 'center',
      width: 200,
      hideInSearch: true,
    },
    {
      title: intl.get('app.table.searchtext'),
      key: 'text',
      hideInTable: true,
    },
    {
      title: intl.get('pages.group.table.note'),
      dataIndex: 'note',
      align: 'center',
      hideInSearch: true,
    },
    ...CreatedAtColumn(intl),
    ...UpdatedAtColumn(intl),
    {
      title: intl.get('app.table.operation'),
      valueType: 'option',
      align: 'center',
      width: 150,
      render: (_, row) => {
        const root = row.name === 'root';
        const right = checkPerm(
          access,
          'manage.group',
          UserPerm.PermWriteExecute,
        );
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
          <ProfileOutlined key="view" />,
          <TableDelete
            key="delete"
            disabled={!right || root}
            tableRef={ref}
            url={`/group/${row.id}`}
            confirmTitle={intl.get('pages.group.table.delete.title', {
              name: row.name,
            })}
          />,
        ];
      },
    },
  ];

  return (
    <ProCard>
      <Table
        actionRef={ref}
        rowKey="id"
        request={request}
        columns={columns}
        action={[
          <TableNew
            tableRef={ref}
            permName="manage.group"
            perm={UserPerm.PermWriteExecute}
            key="add"
            width={500}
            title={intl.get('pages.group.table.add')}
            finish={handleAdd}
            columns={addColumns(intl)}
          />,
        ]}
      />
    </ProCard>
  );
};

export default GroupCard;
