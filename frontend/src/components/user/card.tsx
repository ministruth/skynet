import Table from '@/components/layout/table';
import {
  checkAPI,
  fileToBase64,
  getAPI,
  getIntl,
  paramSort,
  paramTime,
  postAPI,
  UserPerm,
} from '@/utils';
import {
  EditOutlined,
  LogoutOutlined,
  PlusOutlined,
  ProfileOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { Button, Form, message, Upload } from 'antd';
import type { SortOrder } from 'antd/lib/table/interface';
import { useRef } from 'react';
import { FormattedMessage } from 'react-intl';
import { Columns, CreatedAtColumn } from '../layout/table/column';
import TableDelete from '../layout/table/deleteBtn';
import TableNew from '../layout/table/newBtn';

const handleAdd = async (params: ParamsType) => {
  var file;
  const { avatar, ...rest } = params;
  if (avatar && avatar.length != 0) {
    file = await fileToBase64(avatar[0].originFileObj).catch((e) => Error(e));
    if (file instanceof Error) {
      message.error(`Error: ${file.message}`);
      return false;
    }
  }
  return checkAPI(
    postAPI('/user', {
      avatar: file,
      ...rest,
    }),
  );
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

const userColumns: Columns = (intl) => [
  {
    title: intl.get('pages.user.table.username'),
    dataIndex: 'username',
    tooltip: intl.get('pages.user.table.add.usernametip'),
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
    title: intl.get('pages.user.table.password'),
    dataIndex: 'password',
    formItemProps: {
      rules: [
        {
          required: true,
          message: intl.get('app.table.required'),
        },
      ],
    },
  },
];

const addColumns: Columns = (intl) => [
  {
    renderFormItem: () => (
      <FormattedMessage id="pages.user.table.add.content" />
    ),
  },
  ...userColumns(intl),
  {
    title: intl.get('pages.user.table.avatar'),
    dataIndex: 'avatar',
    formItemProps: {
      valuePropName: 'fileList',
    },
    renderFormItem: () => {
      return (
        <Form.Item
          name="avatar"
          valuePropName="fileList"
          getValueFromEvent={(e: any) => {
            if (Array.isArray(e)) {
              return e;
            }
            return e && e.fileList;
          }}
          noStyle
        >
          <Upload
            maxCount={1}
            listType="picture"
            accept=".png,.jpg,.jpeg,.webp"
            beforeUpload={(file) => {
              if (
                !['image/png', 'image/jpeg', 'image/webp'].includes(file.type)
              ) {
                message.error(`${file.name} is not allowed`);
                return Upload.LIST_IGNORE;
              }
              return false;
            }}
          >
            <Button icon={<UploadOutlined />}>
              {intl.get('pages.user.table.add.upload')}
            </Button>
          </Upload>
        </Form.Item>
      );
    },
  },
  {
    title: intl.get('pages.user.table.group'),
    tooltip: intl.get('pages.user.table.add.grouptip'),
    dataIndex: 'group',
    debounceTime: 3000,
    fieldProps: {
      mode: 'multiple',
      showSearch: true,
    },
    placeholder: intl.get('pages.user.table.newline'),
    request: async ({ keyWords }: any) => {
      const msg = await getAPI(
        '/group',
        {
          text: keyWords,
          page: 1,
          size: 5,
        },
        false,
      );
      return msg.data.data.map((e: any) => ({ value: e.id, label: e.name }));
    },
  },
];

const UserCard = () => {
  const intl = getIntl();
  const ref = useRef<ActionType>();
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
      title: intl.get('pages.user.table.avatar'),
      dataIndex: 'avatar',
      valueType: 'avatar',
      align: 'center',
      width: 80,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.user.table.username'),
      dataIndex: 'username',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('app.table.searchtext'),
      key: 'text',
      hideInTable: true,
    },
    {
      title: intl.get('pages.user.table.lastip'),
      dataIndex: 'last_ip',
      align: 'center',
      width: 180,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.user.table.lastlogin'),
      dataIndex: 'last_login',
      align: 'center',
      width: 180,
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
      title: intl.get('app.table.operation'),
      valueType: 'option',
      align: 'center',
      width: 170,
      render: (_, row) => {
        return [
          <PlusOutlined key="clone" />,
          <EditOutlined key="update" />,
          <ProfileOutlined key="view" />,
          <LogoutOutlined key="kick" />,
          <TableDelete
            key="delete"
            permName="manage.user"
            perm={UserPerm.PermWriteExecute}
            tableRef={ref}
            url={`/user/${row.id}`}
            confirmTitle={intl.get('pages.user.table.delete.title', {
              username: row.username,
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
        postData={(data: any[]) => {
          data.map((e) => {
            e.avatar = 'data:image/webp;base64,' + e.avatar;
          });
          return data;
        }}
        request={request}
        columns={columns}
        action={[
          <TableNew
            tableRef={ref}
            permName="manage.user"
            perm={UserPerm.PermWriteExecute}
            key="add"
            width={500}
            title={intl.get('pages.user.table.add')}
            finish={handleAdd}
            columns={addColumns(intl)}
          />,
        ]}
      />
    </ProCard>
  );
};

export default UserCard;
