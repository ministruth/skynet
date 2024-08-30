import confirm from '@/common_components/layout/modal';
import Table from '@/common_components/layout/table';
import {
  CreatedAtColumn,
  IDColumn,
} from '@/common_components/layout/table/column';
import TableDelete from '@/common_components/layout/table/deleteBtn';
import TableNew from '@/common_components/layout/table/newBtn';
import styles from '@/common_components/layout/table/style.less';
import { API_PREFIX } from '@/config';
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
import { ParamsType } from '@ant-design/pro-components';
import { ProFormColumnsType } from '@ant-design/pro-form';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { FormattedMessage, useModel } from '@umijs/max';
import { Button, Modal } from 'antd';
import { SortOrder } from 'antd/es/table/interface';
import { Key, useRef, useState } from 'react';
import styles_custom from './style.less';

const handleDeleteSelected = async (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
  keys: Key[],
) => {
  confirm({
    title: intl.get('pages.agent.op.delete.passive.selected.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        deleleAPI(`${API_PREFIX}/passive_agents`, { id: keys }).then((rsp) => {
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
  const msg = await getAPI(`${API_PREFIX}/passive_agents`, {
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

const PassiveAgent = () => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const intl = getIntl();
  const { access } = useModel('@@qiankunStateFromMaster');
  const ref = useRef<ActionType>();
  const addColumns: ProFormColumnsType[] = [
    {
      title: intl.get('pages.agent.op.passive.table.name'),
      dataIndex: 'name',
      fieldProps: {
        maxLength: 32,
      },
      formItemProps: {
        rules: [{ required: true }],
      },
    },
    {
      title: intl.get('pages.agent.op.passive.table.address'),
      dataIndex: 'address',
      fieldProps: {
        maxLength: 64,
      },
      formItemProps: {
        rules: [{ required: true }],
      },
    },
    {
      title: intl.get('pages.agent.op.passive.table.retrytime'),
      dataIndex: 'retry_time',
      valueType: 'digit',
      tooltip: intl.get('pages.agent.op.passive.table.retrytime.tip'),
      initialValue: 0,
      formItemProps: {
        rules: [{ required: true }],
      },
    },
  ];
  const columns: ProColumns[] = [
    IDColumn(intl),
    {
      title: intl.get('pages.agent.op.passive.table.name'),
      dataIndex: 'name',
      align: 'center',
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
      title: intl.get('pages.agent.op.passive.table.address'),
      dataIndex: 'address',
      align: 'center',
    },
    {
      title: intl.get('pages.agent.op.passive.table.retrytime'),
      dataIndex: 'retry_time',
      align: 'center',
      tooltip: intl.get('pages.agent.op.passive.table.retrytime.tip'),
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
            permName="manage.plugin"
            perm={UserPerm.PermWrite}
            url={`${API_PREFIX}/passive_agents/${row.id}`}
            confirmTitle={intl.get('pages.agent.op.delete.passive.title', {
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
  const [searchText, setSearchText] = useState('');
  const handleAdd = async (params: ParamsType) => {
    if (await checkAPI(postAPI(`${API_PREFIX}/passive_agents`, params))) {
      ref.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };
  return (
    <>
      <Button
        key="passive"
        type="primary"
        disabled={!checkPerm(access, 'manage.plugin', UserPerm.PermRead)}
        onClick={() => setIsModalOpen(true)}
      >
        <FormattedMessage id="pages.agent.op.passive" />
      </Button>
      <Modal
        title={intl.get('pages.agent.op.passive.title')}
        open={isModalOpen}
        footer={null}
        onCancel={() => {
          setIsModalOpen(false);
          setSearchText('');
        }}
        width={1000}
        destroyOnClose={true}
        maskClosable={false}
      >
        <Table
          rowKey="id"
          rowSelection={rowSelection}
          request={request}
          params={{ text: searchText }}
          tableAlertRender={false}
          columns={columns}
          search={false}
          actionRef={ref}
          headerTitle={undefined}
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
                permName="manage.plugin"
                perm={UserPerm.PermWrite}
                key="add"
                width={500}
                title={intl.get('pages.agent.op.add.passive.title')}
                schemaProps={{
                  onFinish: handleAdd,
                  columns: addColumns,
                }}
              />,
              <Button
                key="delete"
                danger
                disabled={
                  !checkPerm(access, 'manage.plugin', UserPerm.PermWrite) ||
                  selectedRowKeys.length === 0
                }
                onClick={() => handleDeleteSelected(intl, ref, selectedRowKeys)}
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

export default PassiveAgent;