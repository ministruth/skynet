import SettingForm, {
  SettingFormRef,
} from '@/common_components/layout/form/setting';
import confirm from '@/common_components/layout/modal';
import { checkAPI, getAPI, getIntl, putAPI } from '@/utils';
import { CopyOutlined } from '@ant-design/icons';
import ProCard from '@ant-design/pro-card';
import {
  ProFormColumnsType,
  ProFormInstance,
  ProFormText,
} from '@ant-design/pro-form';
import { Button, Input, message, Tooltip } from 'antd';
import randomstring from 'randomstring';
import React, { RefObject, useRef } from 'react';
import { CopyToClipboard } from 'react-copy-to-clipboard';
import { FormattedMessage } from 'umi';

const SettingCard = () => {
  const intl = getIntl();
  const formRef = useRef<ProFormInstance>();

  const handleSubmit = (form: Record<string, any>) => {
    return checkAPI(putAPI('/setting', form)).then((rsp) => {
      if (rsp) {
        // order matters to update state
        ref.current?.update(form);
        ref.current?.enable(false);
      }
    });
  };
  const generate = (show: boolean) => {
    if (!show) {
      formRef.current?.setFieldsValue({
        token: randomstring.generate(32),
      });
      ref.current?.enable(true);
    } else {
      confirm({
        title: 'pages.config.setting.token.regenerate.title',
        content: 'pages.config.setting.token.regenerate.content',
        onOk() {
          return new Promise((resolve, reject) => {
            formRef.current?.setFieldsValue({
              token: randomstring.generate(32),
            });
            ref.current?.enable(true);
            resolve(true);
          });
        },
        intl: intl,
      });
    }
  };

  const columns: ProFormColumnsType[] = [
    {
      title: intl.get('pages.config.setting.token.text'),
      dataIndex: 'token',
      formItemProps: {
        style: { marginBottom: 0 }, // delete additional space
      },
      renderFormItem: () => {
        return (
          <Input.Group compact>
            <ProFormText
              name="token"
              placeholder={intl.get('pages.config.setting.token.placeholder')}
              fieldProps={{
                maxLength: 32,
              }}
              width="lg"
            />
            <Tooltip title={intl.get('pages.config.setting.token.tooltip')}>
              <CopyToClipboard
                text={formRef.current?.getFieldValue('token')}
                onCopy={() => {
                  message.success(
                    intl.get('pages.config.setting.token.copied'),
                  );
                }}
              >
                <Button icon={<CopyOutlined />} />
              </CopyToClipboard>
            </Tooltip>
            {ref.current?.default?.token === '' ? (
              <Button type="primary" onClick={() => generate(false)}>
                <FormattedMessage id="pages.config.setting.token.generate" />
              </Button>
            ) : (
              <Button danger onClick={() => generate(true)}>
                <FormattedMessage id="pages.config.setting.token.regenerate" />
              </Button>
            )}
          </Input.Group>
        );
      },
    },
  ];

  const ref = useRef<SettingFormRef>() as RefObject<SettingFormRef>;
  return (
    <ProCard title={intl.get('pages.config.setting.title')} bordered>
      <SettingForm
        req={() => getAPI('/setting')}
        formRef={formRef}
        columns={columns}
        ref={ref}
        onFinish={handleSubmit}
      />
    </ProCard>
  );
};

export default SettingCard;
