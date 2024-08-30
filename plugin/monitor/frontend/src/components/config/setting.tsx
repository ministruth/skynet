import ExSchema, { ExSchemaHandle } from '@/common_components/layout/exschema';
import confirm from '@/common_components/layout/modal';
import { API_PREFIX, BASE_URL } from '@/config';
import {
  StringIntl,
  UserPerm,
  checkAPI,
  checkPerm,
  getAPI,
  getIntl,
  postAPI,
  putAPI,
} from '@/utils';
import { CaretRightOutlined, PauseOutlined } from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import {
  ParamsType,
  ProFormColumnsType,
  ProFormInstance,
} from '@ant-design/pro-components';
import { FormattedMessage, useModel } from '@umijs/max';
import { Button, Space, Typography } from 'antd';
import _ from 'lodash';
import { useRef } from 'react';
import styles from './style.less';
import TagList from './taglist';

const { Text } = Typography;

const request = async (_: ParamsType) => {
  return (await getAPI(`${API_PREFIX}/settings`)).data;
};

const handleSubmit = (
  params: Record<string, any>,
  initial: Record<string, any>,
) => {
  _.forEach(params, (v, k) => {
    if (_.isEqual(initial[k], v)) delete params[k];
  });
  return checkAPI(putAPI(`${API_PREFIX}/settings`, params));
};

const handleRegenerateCert = (intl: StringIntl) => {
  confirm({
    title: intl.get('pages.config.setting.cert.regenerate.title'),
    content: intl.get('pages.config.setting.cert.regenerate.content'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI(`${API_PREFIX}/settings/certificate`, {}).then((rsp) => {
          if (rsp && rsp.code === 0) resolve(rsp);
          else reject(rsp);
        });
      });
    },
    intl: intl,
  });
};

const handleStart = (
  ref: React.MutableRefObject<ExSchemaHandle | undefined>,
) => {
  return checkAPI(
    postAPI(`${API_PREFIX}/settings/server`, {
      start: true,
    }),
  ).then(() =>
    setTimeout(() => {
      ref.current?.refresh();
    }, 1000),
  );
};

const handleStop = (
  intl: StringIntl,
  ref: React.MutableRefObject<ExSchemaHandle | undefined>,
) => {
  confirm({
    title: intl.get('pages.config.setting.status.stopped.title'),
    content: intl.get('pages.config.setting.status.stopped.content'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI(`${API_PREFIX}/settings/server`, { start: false }).then(
          (rsp) => {
            if (rsp && rsp.code === 0) {
              setTimeout(() => {
                ref.current?.refresh();
              }, 1000);
              resolve(rsp);
            } else {
              reject(rsp);
            }
          },
        );
      });
    },
    intl: intl,
  });
};

const SettingCard = () => {
  const intl = getIntl();
  const formRef = useRef<ProFormInstance>();
  const ref = useRef<ExSchemaHandle>();
  const { access } = useModel('@@qiankunStateFromMaster');
  const perm_disable = !checkPerm(access, 'manage.plugin', UserPerm.PermWrite);

  const columns: ProFormColumnsType[] = [
    {
      title: intl.get('pages.config.setting.status.text'),
      dataIndex: 'running',
      readonly: true,
      render: (e) => {
        let status = e ? (
          <Text type="success" strong>
            <FormattedMessage id="pages.config.setting.status.running" />
          </Text>
        ) : (
          <Text type="danger" strong>
            <FormattedMessage id="pages.config.setting.status.stopped" />
          </Text>
        );
        return (
          <>
            {status}
            <Space style={{ marginLeft: '50px' }}>
              <Button
                icon={<CaretRightOutlined />}
                disabled={e === true}
                onClick={() => handleStart(ref)}
              />
              <Button
                icon={<PauseOutlined />}
                danger
                disabled={e === false}
                onClick={() => handleStop(intl, ref)}
              />
            </Space>
          </>
        );
      },
    },
    {
      title: intl.get('pages.config.setting.address.text'),
      dataIndex: 'address',
      fieldProps: {
        className: styles.addr,
      },
      formItemProps: {
        rules: [{ required: true }],
      },
    },
    {
      title: intl.get('pages.config.setting.shell.text'),
      dataIndex: 'shell',
      renderFormItem: () => <TagList disabled={perm_disable} />,
    },
    {
      title: intl.get('pages.config.setting.cert.text'),
      renderFormItem: () => {
        return (
          <Space>
            <Button href={`${BASE_URL}${API_PREFIX}/settings/certificate`}>
              <FormattedMessage id="pages.config.setting.cert.get" />
            </Button>
            <Button onClick={() => handleRegenerateCert(intl)} danger>
              <FormattedMessage id="pages.config.setting.cert.regenerate" />
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <ProCard title={intl.get('pages.config.setting.title')} bordered>
      <ExSchema
        perm_disabled={perm_disable}
        layoutType="Form"
        layout="horizontal"
        labelAlign="left"
        request={request}
        labelCol={{ span: 3 }}
        ref={ref}
        formRef={formRef}
        columns={columns}
        onSubmit={handleSubmit}
      />
    </ProCard>
  );
};

export default SettingCard;
