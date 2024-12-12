import { getAPI, getIntl } from '@/utils';
import { Tooltip } from 'antd';
import { useEffect, useState } from 'react';

export interface GeoIPProps {
  value?: string;
  renderer?: (value?: string) => React.ReactNode;
}

const GeoIP: React.FC<GeoIPProps> = (props) => {
  const intl = getIntl();
  const [tip, setTip] = useState('');
  let value: React.ReactNode = props.value;
  if (props.renderer) value = props.renderer(props.value);

  useEffect(() => {
    const fetch = async () => {
      if (props.value && props.value.length != 0) {
        const msg = await getAPI('/geoip', {
          ip: props.value,
        });
        setTip(msg.data);
      } else {
        setTip(intl.get('app.loading'));
      }
    };
    fetch();
  }, [props.value]);
  return <Tooltip title={tip}>{value}</Tooltip>;
};

export default GeoIP;
