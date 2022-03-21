import Table from '@/common_components/layout/table';
import { getAPI, getIntl } from '@/utils';
import ProCard from '@ant-design/pro-card';
import ProDescriptions from '@ant-design/pro-descriptions';
import { ParamsType } from '@ant-design/pro-provider';
import { ProColumns } from '@ant-design/pro-table';
import { Tag } from 'antd';
import type { SortOrder } from 'antd/lib/table/interface';
import bytes from 'bytes';
import moment from 'moment';

const request = async (
  params?: ParamsType,
  sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI('/agent', {
    status: params?.status,
    text: params?.text,
  });
  return {
    data: msg.data,
    success: true,
    total: msg.total,
  };
};

const MonitorCard = () => {
  const intl = getIntl();
  const statusEnum: { [Key: number]: { label: string; color: string } } = {
    0: {
      label: intl.get('pages.service.table.offline'),
      color: 'default',
    },
    1: {
      label: intl.get('pages.service.table.online'),
      color: 'success',
    },
    2: {
      label: intl.get('pages.service.table.updating'),
      color: 'orange',
    },
  };
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
      title: intl.get('pages.service.table.name'),
      dataIndex: 'name',
      align: 'center',
      hideInSearch: true,
    },
    {
      title: intl.get('pages.service.table.status'),
      dataIndex: 'status',
      align: 'center',
      valueType: 'checkbox',
      valueEnum: Object.entries(statusEnum).reduce(
        (p, c) => ({ ...p, [c[0]]: { text: c[1].label } }),
        {},
      ),
      width: 100,
      render: (_, row) => (
        <Tag style={{ marginRight: 0 }} color={statusEnum[row.status].color}>
          {statusEnum[row.status].label}
        </Tag>
      ),
    },
    {
      title: intl.get('app.table.searchtext'),
      key: 'text',
      hideInTable: true,
    },
    {
      title: intl.get('pages.service.table.ip'),
      dataIndex: 'ip',
      align: 'center',
      width: 150,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.service.table.lastrsp'),
      dataIndex: 'last_rsp',
      align: 'center',
      width: 150,
      hideInSearch: true,
      renderText: (text) => {
        let now = moment(text);
        return now.isSame('0001-01-01T00:00:00Z') ? 'N/A' : now.toNow();
      },
    },
    {
      title: intl.get('pages.service.table.cpu'),
      dataIndex: 'cpu',
      valueType: 'percent',
      align: 'center',
      width: 80,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.service.table.mem'),
      dataIndex: 'percent_mem',
      valueType: 'percent',
      align: 'center',
      width: 80,
      hideInSearch: true,
    },
    {
      title: intl.get('pages.service.table.disk'),
      valueType: 'percent',
      align: 'center',
      width: 80,
      hideInSearch: true,
      renderText: (_, row) => {
        return row.total_disk ? (row.disk * 100) / row.total_disk : 0;
      },
    },
    {
      title: intl.get('pages.service.table.latency'),
      dataIndex: 'latency',
      align: 'center',
      width: 80,
      hideInSearch: true,
      renderText: (text) => (text > 9999 ? '9999 ms' : text + ' ms'),
    },
  ];

  return (
    <ProCard>
      <Table
        rowKey="id"
        poll
        columns={columns}
        request={(params, sort) => request(params, sort)}
        postData={(data: any[]) => {
          data.forEach((e: any) => {
            e.percent_mem = e.total_mem == 0 ? 0 : (e.mem * 100) / e.total_mem;
          });
          return data;
        }}
        expandable={{
          expandRowByClick: true,
          expandedRowRender: (record) => {
            return (
              <ProDescriptions
                dataSource={record}
                columns={[
                  {
                    title: intl.get('pages.service.table.os'),
                    dataIndex: 'os',
                  },
                  {
                    title: intl.get('pages.service.table.hostname'),
                    dataIndex: 'hostname',
                  },
                  {
                    title: intl.get('pages.service.table.system'),
                    dataIndex: 'system',
                  },
                  {
                    title: intl.get('pages.service.table.machine'),
                    dataIndex: 'machine',
                  },
                  {
                    title: intl.get('pages.service.table.lastlogin'),
                    dataIndex: 'last_login',
                    valueType: 'dateTime',
                  },
                  {
                    title: intl.get('pages.service.table.memory'),
                    renderText: (_, row) =>
                      bytes.format(row.mem, { unitSeparator: ' ' }) +
                      ' / ' +
                      bytes.format(row.total_mem, { unitSeparator: ' ' }),
                  },
                  {
                    title: intl.get('pages.service.table.disk'),
                    renderText: (_, row) =>
                      bytes.format(row.disk, { unitSeparator: ' ' }) +
                      ' / ' +
                      bytes.format(row.total_disk, { unitSeparator: ' ' }),
                  },
                  {
                    title: intl.get('pages.service.table.network'),
                    renderText: (_, row) =>
                      bytes.format(row.net_up, { unitSeparator: ' ' }) +
                      ' ↑ | ' +
                      bytes.format(row.net_down, { unitSeparator: ' ' }) +
                      ' ↓',
                  },
                  {
                    title: intl.get('pages.service.table.bandwidth'),
                    renderText: (_, row) =>
                      bytes.format(row.band_up, { unitSeparator: ' ' }) +
                      ' ↑ | ' +
                      bytes.format(row.band_down, { unitSeparator: ' ' }) +
                      ' ↓',
                  },
                  {
                    title: intl.get('pages.service.table.load1'),
                    dataIndex: 'load1',
                    renderText: (text) => text.toPrecision(3),
                  },
                ]}
              />
            );
          },
        }}
      />
    </ProCard>
  );
};

export default MonitorCard;
