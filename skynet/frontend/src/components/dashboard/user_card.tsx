import { getAPI, getIntl } from '@/utils';
import {
  ProCard,
  ProDescriptions,
  ProDescriptionsItemProps,
} from '@ant-design/pro-components';
import { Avatar, Row, Space, Typography } from 'antd';
import { useEffect, useState } from 'react';
import GeoIP from '../geoip';
const { Text } = Typography;

const UserCard = () => {
  const intl = getIntl();
  const [data, setData] = useState<{ [x: string]: any }>({});
  const fetch = async () => {
    const msg = await getAPI(`/users/self`);
    setData(msg.data);
  };

  useEffect(() => {
    fetch();
  }, []);

  const columns: ProDescriptionsItemProps[] = [
    {
      title: intl.get('app.table.id'),
      dataIndex: 'id',
      copyable: true,
    },
    {
      title: intl.get('tables.lastip'),
      dataIndex: 'last_ip',
      render: (_, row) => {
        return <GeoIP value={row.last_ip} />;
      },
    },
    {
      title: intl.get('tables.lastlogin'),
      dataIndex: 'last_login',
      valueType: 'dateTime',
    },
    {
      title: intl.get('app.table.createdat'),
      dataIndex: 'created_at',
      valueType: 'dateTime',
    },
  ];

  return (
    <ProCard bordered>
      <Row align="middle">
        <Space align="center" size="middle">
          <Avatar size="large" src={<img src={data.avatar} alt="avatar" />} />
          <Text>{data.username}</Text>
        </Space>
      </Row>
      <ProDescriptions
        style={{ marginTop: '8px' }}
        columns={columns}
        emptyText={'-'}
        dataSource={data}
        column={1}
      />
    </ProCard>
  );
};

export default UserCard;
