import { getAPI, getIntl } from '@/utils';
import {
  ProCard,
  ProDescriptions,
  ProDescriptionsItemProps,
} from '@ant-design/pro-components';
import bytes from 'bytes';
import { useEffect, useState } from 'react';

const SystemCard = () => {
  const intl = getIntl();
  const [data, setData] = useState<{ [x: string]: any }>({});

  useEffect(() => {
    const fetch = async () => {
      const msg = await getAPI('/dashboard/system_info');
      setData(msg.data);
    };
    fetch();
  }, []);

  const columns: ProDescriptionsItemProps[] = [
    {
      title: intl.get('tables.skynet_version'),
      dataIndex: 'version',
    },
    {
      title: intl.get('tables.cpu'),
      dataIndex: 'cpu',
    },
    {
      title: intl.get('tables.memory'),
      dataIndex: 'memory',
      renderText: (_, row) =>
        bytes.format(row.memory, { unitSeparator: ' ' }) ?? '-',
    },
    {
      title: intl.get('tables.start_time'),
      dataIndex: 'start_time',
      valueType: 'dateTime',
    },
  ];

  return (
    <ProCard bordered title={intl.get('pages.dashboard.system.title')}>
      <ProDescriptions
        columns={columns}
        emptyText={'-'}
        dataSource={data}
        column={1}
      />
    </ProCard>
  );
};

export default SystemCard;
