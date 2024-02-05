import {
  UserPerm,
  checkAPI,
  getAPI,
  getIntl,
  paramSort,
  postAPI,
} from '@/utils';
import { UploadOutlined } from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { FormattedMessage } from '@umijs/max';
import { Alert, Button, Tag } from 'antd';
import { SortOrder } from 'antd/es/table/interface';
import { CustomTagProps } from 'rc-select/es/BaseSelect';
import { useRef } from 'react';
import Table from '../layout/table';
import { Columns, IDColumn, SearchColumn } from '../layout/table/column';
import TableDelete from '../layout/table/deleteBtn';
import TableNew from '../layout/table/newBtn';
import styles from '../layout/table/style.less';
import PluginAble from './ableBtn';
import PluginUpload from './upload';
var CRC32 = require('crc-32');

const request = async (
  params?: ParamsType,
  sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI('/plugins', {
    priority_sort: paramSort(sort?.priority),
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
      label: 'Pending Disable',
      color: 'warning',
    },
    2: {
      label: 'Pending Enable',
      color: 'orange',
    },
    3: {
      label: 'Enable',
      color: 'success',
    },
  };
  const handleUpload = async (params: ParamsType) => {
    if (await checkAPI(postAPI('/plugins', params))) {
      ref.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };
  const UploadColumns: Columns = (intl) => [
    {
      renderFormItem: () => (
        <Alert
          message={intl.get('pages.plugin.op.upload.warning')}
          description={intl.get('pages.plugin.op.upload.content')}
          type="warning"
          showIcon
        />
      ),
    },
    {
      title: intl.get('pages.plugin.table.path'),
      dataIndex: 'path',
      tooltip: intl.get('pages.plugin.form.path.tip'),
      fieldProps: {
        maxLength: 32,
      },
      formItemProps: {
        getValueFromEvent: (event) => {
          return event.target.value.replace(/[^a-zA-Z0-9\-_]+/g, '');
        },
        rules: [{ required: true }],
      },
    },
    {
      title: intl.get('pages.plugin.form.file'),
      dataIndex: 'file',
      renderFormItem: (_schema, _config, form) => {
        return (
          <PluginUpload
            changeHook={(f: string) => {
              form.setFieldValue(
                'crc32',
                f != ''
                  ? CRC32.buf(Buffer.from(f.split(',')[1], 'base64')) >>> 0
                  : f,
              );
              return f;
            }}
          />
        );
      },
      formItemProps: {
        rules: [{ required: true }],
      },
    },
    {
      title: intl.get('pages.plugin.form.crc32'),
      dataIndex: 'crc32',
      readonly: true,
    },
  ];
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
      title: intl.get('pages.plugin.table.priority'),
      dataIndex: 'priority',
      align: 'center',
      sorter: true,
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
            enable={row.status === 2 || row.status === 3}
            tableRef={ref}
            pid={row.id}
            pname={row.name}
          />,
          <TableDelete
            key="delete"
            tableRef={ref}
            disabled={row.status !== 0}
            permName="manage.plugin"
            perm={UserPerm.PermWrite}
            url={`/plugins/${row.id}`}
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
        action={[
          <TableNew
            permName="manage.plugin"
            perm={UserPerm.PermWrite}
            key="upload"
            width={500}
            title={intl.get('pages.plugin.op.upload.title')}
            schemaProps={{
              onFinish: handleUpload,
              columns: UploadColumns(intl),
            }}
            trigger={
              <Button
                key="upload"
                type="primary"
                icon={<UploadOutlined style={{ marginRight: '8px' }} />}
              >
                <FormattedMessage id="app.op.upload" />
              </Button>
            }
            rollback={
              <Button
                key="upload"
                type="primary"
                icon={<UploadOutlined style={{ marginRight: '8px' }} />}
                disabled
              >
                <FormattedMessage id="aoo.op.upload" />
              </Button>
            }
          />,
        ]}
      />
    </ProCard>
  );
};

export default PluginCard;
