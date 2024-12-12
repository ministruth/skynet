import { checkPerm, UserPerm } from '@/utils';
import { useAccess } from '@umijs/max';
import { Col, Row, Typography } from 'antd';
import RuntimeCard from './runtime_card';
import SystemCard from './system_card';
import UserCard from './user_card';
const { Text } = Typography;

export interface CardProps {
  style?: React.CSSProperties;
}

const DashboardCard = () => {
  const access = useAccess();
  return (
    <>
      <Row
        style={{ marginTop: '30px', marginLeft: '30px', marginRight: '30px' }}
        gutter={[16, 16]}
      >
        <Col xs={24} md={12}>
          <UserCard />
        </Col>
        {checkPerm(access, 'manage.system', UserPerm.PermRead) && (
          <Col xs={24} md={12}>
            <SystemCard />
          </Col>
        )}
      </Row>
      {checkPerm(access, 'manage.system', UserPerm.PermRead) && (
        <Row
          style={{
            marginTop: '30px',
            marginLeft: '30px',
            marginRight: '30px',
          }}
          gutter={[16, 16]}
        >
          <RuntimeCard />
        </Row>
      )}
    </>
  );
};

export default DashboardCard;
