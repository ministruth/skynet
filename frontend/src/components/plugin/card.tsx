import { UserPerm, getAPI, getIntl } from '@/utils';
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { Tag } from 'antd';
import { SortOrder } from 'antd/es/table/interface';
import { CustomTagProps } from 'rc-select/es/BaseSelect';
import { useRef } from 'react';
import Table from '../layout/table';
import { IDColumn, SearchColumn } from '../layout/table/column';
import TableDelete from '../layout/table/deleteBtn';
import styles from '../layout/table/style.less';
import PluginAble from './ableBtn';

const request = async (params?: ParamsType, _?: Record<string, SortOrder>) => {
  const msg = await getAPI('/plugin', {
    status: params?.status,
    text: params?.text,
    page: params?.current,
    size: params?.pageSize,
  });
  return {
    data: msg.data.data,
    success: true,
    total: msg.data.total,
  };
};

const PluginCard = () => {
  const intl = getIntl();
  const ref = useRef<ActionType>();
  const statusEnum: { [Key: number]: { label: string; color: string } } = {
    0: {
      label: 'Unload',
      color: 'default',
    },
    1: {
      label: 'Disable',
      color: 'warning',
    },
    2: {
      label: 'Enable',
      color: 'success',
    },
  };
  const columns: ProColumns[] = [
    SearchColumn(intl),
    IDColumn(intl),
    {
      title: intl.get('pages.plugin.table.name'),
      dataIndex: 'name',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('pages.plugin.table.status'),
      dataIndex: 'status',
      align: 'center',
      valueType: 'select',
      fieldProps: {
        mode: 'multiple',
        tagRender: (props: CustomTagProps) => {
          return (
            <Tag
              color={statusEnum[props.value].color}
              closable={props.closable}
              onClose={props.onClose}
              style={{ marginRight: 4 }}
            >
              {props.label}
            </Tag>
          );
        },
      },
      valueEnum: Object.entries(statusEnum).reduce(
        (p, c) => ({ ...p, [c[0]]: { text: c[1].label } }),
        {},
      ),
      render: (_, row) => (
        <Tag style={{ marginRight: 0 }} color={statusEnum[row.status].color}>
          {statusEnum[row.status].label}
        </Tag>
      ),
    },
    {
      title: intl.get('pages.plugin.table.version'),
      dataIndex: 'version',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('pages.plugin.table.path'),
      dataIndex: 'path',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('app.op'),
      valueType: 'option',
      align: 'center',
      width: 100,
      className: styles.operation,
      render: (_, row) => {
        return [
          <PluginAble
            key="able"
            enable={row.status === 2}
            tableRef={ref}
            pid={row.id}
            pname={row.name}
          />,
          <TableDelete
            key="delete"
            tableRef={ref}
            disabled={row.status !== 0}
            permName="manage.plugin"
            perm={UserPerm.PermWriteExecute}
            url={`/plugin/${row.id}`}
            confirmTitle={intl.get('pages.plugin.table.delete.title', {
              name: row.name,
            })}
          />,
        ];
      },
    },
  ];
  return (
    <ProCard bordered>
      <Table
        actionRef={ref}
        rowKey="id"
        request={request}
        columns={columns}
        // action={[
        //   <Upload
        //     key="upload"
        //     name="file"
        //     accept=".sp"
        //     maxCount={1}
        //     showUploadList={false}
        //     action={(file: RcFile) => handleUpload(ref, file)}
        //   >
        //     <Button type="primary" icon={<UploadOutlined />}>
        //       {intl.get('pages.plugin.table.upload')}
        //     </Button>
        //   </Upload>,
        // ]}
      />
    </ProCard>
  );
};

export default PluginCard;
