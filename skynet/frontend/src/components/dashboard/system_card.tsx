import { getAPI, getIntl } from '@/utils';
import { QuestionCircleOutlined } from '@ant-design/icons';
import {
  ProCard,
  ProDescriptions,
  ProDescriptionsItemProps,
} from '@ant-design/pro-components';
import { FormattedMessage, useAccess } from '@umijs/max';
import { Space, Tooltip, Typography } from 'antd';
import bytes from 'bytes';
import { flatMap } from 'lodash';
import { useEffect, useState } from 'react';

const { Text } = Typography;

const SystemCard = () => {
  const intl = getIntl();
  const access = useAccess();
  const [data, setData] = useState<{ [x: string]: any }>({});

  useEffect(() => {
    const fetch = async () => {
      let msg = await getAPI('/dashboard/system_info');
      let health = await getAPI('/health');
      if (msg.data.warning.length > 0) health.code = 2;
      msg.data.status = health.code;
      setData(msg.data);
    };
    fetch();
  }, []);

  const columns: ProDescriptionsItemProps[] = [
    {
      title: intl.get('tables.status'),
      dataIndex: 'status',
      render: (_, row) => {
        return row.status === 0 ? (
          <Text type="success" strong>
            <FormattedMessage id="pages.dashboard.status.healthy" />
          </Text>
        ) : row.status === 1 ? (
          <Text type="danger" strong>
            <FormattedMessage id="pages.dashboard.status.notready" />
          </Text>
        ) : (
          <Space>
            <Text type="warning" strong>
              <FormattedMessage id="pages.dashboard.status.pending" />
            </Text>
            <Tooltip
              title={
                <>
                  {flatMap(row.warning, (item: any) => [
                    <p style={{ marginBottom: 0 }}>{item}</p>,
                  ])}
                </>
              }
            >
              <QuestionCircleOutlined style={{ cursor: 'help' }} />
            </Tooltip>
          </Space>
        );
      },
    },
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
