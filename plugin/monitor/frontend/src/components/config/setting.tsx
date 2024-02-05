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
  BetaSchemaForm,
  ParamsType,
  ProFormColumnsType,
  ProFormInstance,
  ProFormText,
} from '@ant-design/pro-components';
import { FormattedMessage, useModel } from '@umijs/max';
import { Button, Space, Tooltip, message } from 'antd';
import copy from 'copy-to-clipboard';
import { isEqual } from 'lodash';
import randomstring from 'randomstring';
import { useRef, useState } from 'react';
import styles from './style.less';

const SettingCard = () => {
  const intl = getIntl();
  const formRef = useRef<ProFormInstance>();
  const [seq, setSeq] = useState(0);
  const { access } = useModel('@@qiankunStateFromMaster');
  const perm_disable = !checkPerm(access, 'manage.plugin', UserPerm.PermWrite);
  const [values, setValues] = useState();

  const handleSubmit = (form: Record<string, any>) => {
    return checkAPI(putAPI(`${API_PREFIX}/settings`, form)).then((rsp) => {
      if (rsp) setSeq(seq + 1);
    });
  };
  const request = async (_: ParamsType) => {
    let data = (await getAPI(`${API_PREFIX}/settings`)).data;
    setValues(data);
    return data;
  };
  const [changed, setChanged] = useState(false);

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
              formItemProps={{ className: styles.token }}
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
                setChanged(true);
              }}
              disabled={perm_disable}
            >
              <FormattedMessage id="pages.config.setting.token.regenerate" />
            </Button>
          </Space.Compact>
        );
      },
    },
  ];

  return (
    <ProCard title={intl.get('pages.config.setting.title')} bordered>
      <BetaSchemaForm
        layoutType="Form"
        layout="horizontal"
        labelAlign="left"
        params={{ seq: seq }}
        request={request}
        labelCol={{ span: 3 }}
        formRef={formRef}
        columns={columns}
        onFinish={handleSubmit}
        submitter={{
          onReset: () => setChanged(false),
          resetButtonProps: { disabled: perm_disable },
          submitButtonProps: { disabled: perm_disable || !changed },
          render: (_, dom) => [...dom.reverse()],
        }}
        initialValues={values}
        onValuesChange={(_: any, all: Record<string, any>) => {
          if (values)
            for (let k in all) {
              // possible object
              if (!isEqual(values[k], all[k])) {
                setChanged(true);
                return;
              }
            }
          setChanged(false);
        }}
      />
    </ProCard>
  );
};

export default SettingCard;
