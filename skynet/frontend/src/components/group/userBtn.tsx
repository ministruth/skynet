import {
  checkAPI,
  checkPerm,
  deleleAPI,
  getAPI,
  getIntl,
  paramSort,
  postAPI,
  StringIntl,
  UserPerm,
} from '@/utils';
import { UserOutlined } from '@ant-design/icons';
import { ProFormColumnsType } from '@ant-design/pro-form';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { FormattedMessage, useAccess } from '@umijs/max';
import { Button, Modal } from 'antd';
import { SortOrder } from 'antd/es/table/interface';
import { Key, useRef, useState } from 'react';
import confirm from '../layout/modal';
import Table from '../layout/table';
import { CreatedAtColumn, IDColumn } from '../layout/table/column';
import TableDelete from '../layout/table/deleteBtn';
import TableNew from '../layout/table/newBtn';
import styles from '../layout/table/style.less';
import TableBtn from '../layout/table/tableBtn';
import styles_custom from './style.less';

export interface UserBtnProps {
  gid: string;
}

const handleDeleteSelected = async (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
  keys: Key[],
  gid: string,
) => {
  confirm({
    title: intl.get('pages.group.op.delete.selected.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        deleleAPI(`/groups/${gid}/users`, { id: keys }).then((rsp) => {
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
  id: string,
  params?: ParamsType,
  sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI(`/groups/${id}/users`, {
    text: params?.text === '' ? undefined : params?.text,
    created_sort: paramSort(sort?.created_at) || 'desc',
    page: params?.current,
    size: params?.pageSize,
  });
  return {
    data: msg.data.data,
    success: true,
    total: msg.data.total,
  };
};

const GroupUser: React.FC<UserBtnProps> = (props) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const intl = getIntl();
  const access = useAccess();
  const ref = useRef<ActionType>();
  const addColumns: ProFormColumnsType[] = [
    {
      title: intl.get('pages.user.table.username'),
      dataIndex: 'id',
      fieldProps: {
        mode: 'multiple',
        showSearch: true,
      },
      request: async ({ keyWords }: any) => {
        const msg = await getAPI(
          '/users',
          {
            username: keyWords,
            page: 1,
            size: 5,
          },
          false,
        );
        return msg.data.data.map((e: any) => ({
          value: e.id,
          label: e.username,
        }));
      },
      formItemProps: {
        rules: [{ required: true }],
      },
    },
  ];
  const columns: ProColumns[] = [
    IDColumn(intl),
    {
      title: intl.get('pages.user.table.username'),
      dataIndex: 'username',
      align: 'center',
      ellipsis: true,
      onCell: () => {
        return {
          style: {
            maxWidth: 150,
          },
        };
      },
    },
    ...CreatedAtColumn(intl),
    {
      title: intl.get('app.op'),
      valueType: 'option',
      align: 'center',
      width: 80,
      className: styles.operation,
      render: (_, row) => {
        return [
          <TableDelete
            key="delete"
            tableRef={ref}
            permName="manage.user"
            perm={UserPerm.PermWrite}
            url={`/groups/${props.gid}/users/${row.id}`}
            confirmTitle={intl.get('pages.group.op.delete.user.title', {
              name: row.username,
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
  const [searchText, setSearchText] = useState('');
  const handleAdd = async (params: ParamsType) => {
    if (await checkAPI(postAPI(`/groups/${props.gid}/users`, params))) {
      ref.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };

  if (!checkPerm(access, 'manage.user', UserPerm.PermRead))
    return <UserOutlined key="user" />;
  else
    return (
      <>
        <TableBtn
          icon={UserOutlined}
          tip={intl.get('pages.group.op.user.tip')}
          onClick={() => setIsModalOpen(true)}
        />
        <Modal
          title={intl.get('pages.group.op.user.title')}
          open={isModalOpen}
          footer={null}
          onCancel={() => {
            setIsModalOpen(false);
            setSearchText('');
          }}
          width={700}
          destroyOnClose={true}
          maskClosable={false}
        >
          <Table
            params={{ text: searchText }}
            search={false}
            headerTitle={undefined}
            actionRef={ref}
            rowKey="id"
            rowSelection={rowSelection}
            tableAlertRender={false}
            request={(params, sort) => request(props.gid, params, sort)}
            columns={columns}
            cardBordered={false}
            cardProps={{ bodyStyle: { paddingInline: 0, paddingBlock: 0 } }}
            toolbar={{
              className: styles_custom['toolbar-row'],
              search: {
                onSearch: (value: string) => {
                  setSearchText(value);
                },
              },
              actions: [
                <TableNew
                  permName="manage.user"
                  perm={UserPerm.PermWrite}
                  key="add"
                  width={500}
                  title={intl.get('pages.group.op.add.user.title')}
                  schemaProps={{
                    onFinish: handleAdd,
                    columns: addColumns,
                  }}
                />,
                <Button
                  key="delete"
                  danger
                  disabled={
                    !checkPerm(access, 'manage.user', UserPerm.PermWrite) ||
                    selectedRowKeys.length === 0
                  }
                  onClick={() =>
                    handleDeleteSelected(intl, ref, selectedRowKeys, props.gid)
                  }
                >
                  <FormattedMessage id="app.op.delete" />
                </Button>,
              ],
            }}
          />
        </Modal>
      </>
    );
};

export default GroupUser;
