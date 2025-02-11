import {
  checkAPI,
  checkPerm,
  getAPI,
  getIntl,
  putAPI,
  UserPerm,
} from '@/utils';
import ProCard from '@ant-design/pro-card';
import { FormattedMessage, useAccess } from '@umijs/max';
import { Button, Input, InputNumber, Space } from 'antd';
import _ from 'lodash';
import { HTMLAttributes, useEffect, useState } from 'react';
import RowItem from '../layout/rowItem';

const { TextArea } = Input;

const SettingCard: React.FC<HTMLAttributes<HTMLDivElement>> = (props) => {
  const intl = getIntl();
  const access = useAccess();
  const disable = !checkPerm(access, 'manage.system', UserPerm.PermWrite);
  const [initial, setInitial] = useState<{ [x: string]: any }>({});
  const [data, setData] = useState<{ [x: string]: any }>({});
  const fetch = async () => {
    const msg = await getAPI(`/settings/system`);
    msg.data['webpush.endpoint'] = msg.data['webpush.endpoint'].join('\n');
    setData(msg.data);
    setInitial(msg.data);
  };
  useEffect(() => {
    fetch();
  }, []);
  const handlePut = async (param: { [x: string]: any }) => {
    let data: { [x: string]: any } = {};
    _.forEach(param, (v, k) => {
      if (!_.isEqual(initial[k], v)) data[k] = param[k];
    });
    data['webpush.endpoint'] = data['webpush.endpoint'].split('\n');
    if (await checkAPI(putAPI('/settings/system', data))) {
      fetch();
    }
  };

  return (
    <ProCard title={intl.get('pages.system.setting.title')} bordered {...props}>
      <RowItem
        span={{ xs: 14, md: 6 }}
        text={<FormattedMessage id="pages.system.setting.session.expire" />}
        item={
          <InputNumber
            min={0}
            disabled={disable}
            value={data['session.expire']}
            onChange={(value) => {
              setData({ ...data, 'session.expire': value });
            }}
            addonAfter={intl.get('pages.system.setting.session.second')}
          />
        }
      />
      <RowItem
        span={{ xs: 14, md: 6 }}
        text={<FormattedMessage id="pages.system.setting.session.remember" />}
        item={
          <InputNumber
            min={0}
            disabled={disable}
            value={data['session.remember']}
            onChange={(value) => {
              setData({ ...data, 'session.remember': value });
            }}
            addonAfter={intl.get('pages.system.setting.session.second')}
          />
        }
      />
      <RowItem
        span={{ xs: 14, md: 6 }}
        nextSpan={{ xs: 24, md: 10 }}
        text={<FormattedMessage id="pages.system.setting.webpush.endpoint" />}
        item={
          <TextArea
            disabled={disable}
            style={{ width: '100%' }}
            value={data['webpush.endpoint']}
            autoSize
            onChange={(e) => {
              setData({ ...data, 'webpush.endpoint': e.target.value });
            }}
          />
        }
      />
      <Space size="middle">
        <Button
          disabled={disable || _.isEqual(data, initial)}
          type="primary"
          onClick={() => handlePut(data)}
        >
          <FormattedMessage id="app.ok" />
        </Button>
        <Button disabled={disable} onClick={() => setData(initial)}>
          <FormattedMessage id="app.reset" />
        </Button>
      </Space>
    </ProCard>
  );
};

export default SettingCard;
