import { BetaSchemaForm } from '@ant-design/pro-form';
import { FormSchema } from '@ant-design/pro-form/lib/components/SchemaForm';
import React, { forwardRef, useImperativeHandle, useState } from 'react';
import { useRequest } from 'umi';

interface SettingFormProps {
  req: () => Promise<any>;
}

export interface SettingFormRef {
  default: Record<string, any> | undefined;
  enable: (e: boolean) => void;
  status: boolean;
  update: (data: Record<string, any>) => void;
}

const SettingForm = forwardRef<SettingFormRef, SettingFormProps & FormSchema>(
  (props, ref) => {
    const [setting, setSetting] = useState<{ [Key: string]: any }>();
    const { req, ...rest } = props;
    const [change, setChange] = useState(false);

    useImperativeHandle(ref, () => ({
      enable: (e) => {
        setChange(e);
      },
      default: setting,
      status: change,
      update: (data) => {
        setSetting(data);
      },
    }));

    useRequest(req, {
      onSuccess: (rsp) => setSetting(rsp),
    });

    if (setting)
      return (
        <BetaSchemaForm
          submitter={{
            render: (_, doms) => {
              return doms.reverse(); // fix button reverse bug
            },
            submitButtonProps: {
              disabled: !change,
            },
            onReset: () => {
              props.form?.resetFields();
              setChange(false);
            },
          }}
          // onchange bug wont trigger when click clear.
          onFieldsChange={() => {
            setChange(true);
          }}
          layoutType="Form"
          layout="horizontal"
          labelAlign="left"
          labelCol={{ span: 3 }}
          initialValues={setting}
          {...rest}
        />
      );
    else return null;
  },
);

export default SettingForm;
