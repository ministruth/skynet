import { getIntl, StringIntl } from '@/utils';
import {
  AndroidOutlined,
  AppleOutlined,
  LinuxOutlined,
  MobileOutlined,
  QuestionCircleOutlined,
  WindowsOutlined,
} from '@ant-design/icons';
import { Space, Tooltip } from 'antd';

export interface UserAgentProps {
  value?: string;
}

const getDevice = (intl: StringIntl, ua?: string) => {
  if (ua) {
    ua = ua.toLowerCase();
    if (ua.indexOf('windows') != -1) {
      return ['Windows', <WindowsOutlined />];
    }
    if (ua.indexOf('android') != -1) {
      return ['Android', <AndroidOutlined />];
    }
    if (ua.indexOf('iphone') != -1) {
      return ['iPhone', <AppleOutlined />];
    }
    if (ua.indexOf('macintosh') != -1) {
      return ['Mac', <AppleOutlined />];
    }
    if (ua.indexOf('mobile') != -1) {
      return [intl.get('pages.history.ua.mobile'), <MobileOutlined />];
    }
    if (ua.indexOf('linux') != -1) {
      return ['Linux', <LinuxOutlined />];
    }
  }
  return [intl.get('pages.history.ua.unknown'), <QuestionCircleOutlined />];
};

const UserAgent: React.FC<UserAgentProps> = (props) => {
  const intl = getIntl();
  let [deviceName, deviceIcon] = getDevice(intl, props.value);
  return (
    <Tooltip title={props.value ?? '-'}>
      <Space>
        {deviceIcon}
        {deviceName}
      </Space>
    </Tooltip>
  );
};

export default UserAgent;
