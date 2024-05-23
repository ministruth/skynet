import ExSchema, { ExSchemaHandle } from '@/common_components/layout/exschema';
import { API_PREFIX } from '@/config';
import {
  UserPerm,
  checkAPI,
  checkPerm,
  getAPI,
  getIntl,
  putAPI,
} from '@/utils';
import { CopyOutlined } from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import {
  ParamsType,
  ProFormColumnsType,
  ProFormInstance,
  ProFormText,
} from '@ant-design/pro-components';
import { FormattedMessage, useModel } from '@umijs/max';
import { Button, Space, Tooltip, message } from 'antd';
import copy from 'copy-to-clipboard';
import _ from 'lodash';
import randomstring from 'randomstring';
import { useRef } from 'react';
import styles from './style.less';
import TagList from './taglist';

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

const SettingCard = () => {
  const intl = getIntl();
  const formRef = useRef<ProFormInstance>();
  const ref = useRef<ExSchemaHandle>();
  const { access } = useModel('@@qiankunStateFromMaster');
  const perm_disable = !checkPerm(access, 'manage.plugin', UserPerm.PermWrite);

  const columns: ProFormColumnsType[] = [
    {
      title: intl.get('pages.config.setting.token.text'),
      dataIndex: 'token',
      renderFormItem: () => {
        return (
          <Space.Compact block>
            <ProFormText
              name="token"
              placeholder={intl.get('pages.config.setting.token.placeholder')}
              fieldProps={{
                maxLength: 32,
              }}
              disabled={perm_disable}
              formItemProps={{
                className: styles.token,
                style: { marginBottom: 0 },
              }}
            />
            <Tooltip title={intl.get('pages.config.setting.token.tooltip')}>
              <Button
                icon={<CopyOutlined />}
                onClick={() => {
                  copy(formRef.current?.getFieldValue('token'), {
                    format: 'text/plain',
                  });
                  message.success(
                    intl.get('pages.config.setting.token.copied'),
                  );
                }}
              />
            </Tooltip>
            <Button
              danger
              onClick={() => {
                formRef.current?.setFieldsValue({
                  token: randomstring.generate(32),
                });
                ref.current?.enableSubmit(true);
              }}
              disabled={perm_disable}
            >
              <FormattedMessage id="pages.config.setting.token.regenerate" />
            </Button>
          </Space.Compact>
        );
      },
    },
    {
      title: intl.get('pages.config.setting.shell.text'),
      dataIndex: 'shell',
      renderFormItem: () => <TagList disabled={perm_disable} />,
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
