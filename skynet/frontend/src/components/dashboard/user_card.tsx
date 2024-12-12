import { getAPI, getIntl, postAPI, StringIntl } from '@/utils';
import {
  ProCard,
  ProDescriptions,
  ProDescriptionsItemProps,
} from '@ant-design/pro-components';
import { FormattedMessage } from '@umijs/max';
import { Avatar, Button, Col, Flex, Row, Space, Typography } from 'antd';
import { useEffect, useState } from 'react';
import GeoIP from '../geoip';
import confirm from '../layout/modal';
import HistoryLink from './historyLink';
import UpdateBtn from './updateBtn';
const { Text } = Typography;

const handleKick = (intl: StringIntl) => {
  confirm({
    title: intl.get('pages.dashboard.logoutall.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI(`/users/self/kick`, {}).then((rsp) => {
          if (rsp && rsp.code === 0) {
            resolve(rsp);
            window.location.reload();
          } else {
            reject(rsp);
          }
        });
      });
    },
    intl: intl,
  });
};

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
        return (
          <Space>
            <GeoIP value={row.last_ip} />
            <HistoryLink />
          </Space>
        );
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
        <Col>
          <Space align="center" size="middle">
            <Avatar size="large" src={<img src={data.avatar} alt="avatar" />} />
            <Text>{data.username}</Text>
          </Space>
        </Col>
        <Col flex="auto">
          <Flex justify="flex-end">
            <Space align="center" size="small">
              <UpdateBtn initialValues={data} reload={fetch} />
              <Button size="small" danger onClick={() => handleKick(intl)}>
                <FormattedMessage id="pages.dashboard.logoutall" />
              </Button>
            </Space>
          </Flex>
        </Col>
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
