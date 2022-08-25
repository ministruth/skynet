import Table from '@/components/layout/table';
import { checkAPI, fileToBase64, getAPI, getIntl, postAPI } from '@/utils';
import { UploadOutlined } from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { Button, message, Tag, Upload } from 'antd';
import type { SortOrder } from 'antd/lib/table/interface';
import { RcFile } from 'antd/lib/upload';
import { useRef } from 'react';
import { useModel } from 'umi';
import TableDelete from '../layout/table/deleteBtn';
import PluginAble from './ableBtn';

const handleUpload = async (
  ref: React.MutableRefObject<ActionType | undefined>,
  refresh: () => Promise<void>,
  file: RcFile,
) => {
  let f = await fileToBase64(file).catch((e) => Error(e));
  if (f instanceof Error) {
    message.error(`Error: ${f.message}`);
    return false;
  }
  return checkAPI(
    postAPI('/plugin', {
      file: f,
    }),
  ).then((rsp) => {
    if (rsp) ref.current?.reloadAndRest?.();
    return rsp;
  });
};

const request = async (
  params?: ParamsType,
  sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI('/plugin', {
    enable: params?.enable,
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

// const pluginColumns: Columns = (intl) => [
//   {
//     title: intl.get('pages.group.table.name'),
//     dataIndex: 'name',
//     tooltip: intl.get('pages.group.table.add.nametip'),
//     formItemProps: {
//       rules: [
//         {
//           required: true,
//           message: intl.get('app.table.required'),
//         },
//       ],
//     },
//   },
//   {
//     title: intl.get('pages.group.table.note'),
//     dataIndex: 'note',
//   },
// ];

const PluginCard = () => {
  const intl = getIntl();
  const ref = useRef<ActionType>();
  const { refresh } = useModel('@@initialState');
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
      title: intl.get('pages.plugin.table.name'),
      dataIndex: 'name',
      align: 'center',
      width: 150,
      hideInSearch: true,
    },
    {
      title: intl.get('app.table.searchtext'),
      key: 'text',
      hideInTable: true,
    },
    {
      title: intl.get('pages.plugin.table.status'),
      dataIndex: 'enable',
      align: 'center',
      valueType: 'checkbox',
      valueEnum: {
        false: intl.get('pages.plugin.table.disabled'),
        true: intl.get('pages.plugin.table.enabled'),
      },
      width: 100,
      render: (_, row) => (
        <Tag
          style={{ marginRight: 0 }}
          color={row.enable ? 'success' : undefined}
        >
          {row.enable
            ? intl.get('pages.plugin.table.enabled')
            : intl.get('pages.plugin.table.disabled')}
        </Tag>
      ),
    },
    {
      title: intl.get('pages.plugin.table.version'),
      dataIndex: 'version',
      align: 'center',
      width: 100,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.plugin.table.skynetversion'),
      dataIndex: 'skynet_version',
      align: 'center',
      width: 150,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.plugin.table.message'),
      dataIndex: 'message',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('app.table.operation'),
      valueType: 'option',
      align: 'center',
      width: 100,
      render: (_, row) => {
        return [
          <PluginAble
            key="able"
            enable={row.enable}
            tableRef={ref}
            pluginID={row.id}
            pluginName={row.name}
          />,
          <TableDelete
            key="delete"
            tableRef={ref}
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
    <ProCard>
      <Table
        actionRef={ref}
        rowKey="id"
        request={request}
        columns={columns}
        action={[
          <Upload
            key="upload"
            name="file"
            accept=".sp"
            maxCount={1}
            showUploadList={false}
            action={(file: RcFile) => handleUpload(ref, refresh, file)}
          >
            <Button type="primary" icon={<UploadOutlined />}>
              {intl.get('pages.plugin.table.upload')}
            </Button>
          </Upload>,
        ]}
      />
    </ProCard>
  );
};

export default PluginCard;
