import { BetaSchemaForm, ProFormInstance } from '@ant-design/pro-form';
import { FormSchema } from '@ant-design/pro-form/es/components/SchemaForm';
import { Modal } from 'antd';
import { isEqual } from 'lodash';
import React, { useRef, useState } from 'react';

export interface ModalSchemaProps {
  trigger?: JSX.Element;
  title?: React.ReactNode;
  width?: string | number;
  schemaProps: FormSchema;
  changedSubmit?: boolean;
}

// bug in BetaSchemaForm ModalForm, split manually.
const ModalSchema: React.FC<ModalSchemaProps> = (props) => {
  const formRef = useRef<ProFormInstance>(null);
  const [open, setOpen] = useState(false);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [changed, setChanged] = useState(false);
  const trigger = props.trigger ? (
    React.cloneElement(props.trigger, {
      onClick: () => {
        setOpen(true);
        setChanged(false); // reopen modal will not refresh 'changed'
      },
    })
  ) : (
    <></>
  );
  const ok = () => {
    setConfirmLoading(true);
    formRef.current
      ?.validateFields()
      .then(async () => {
        if (
          await props.schemaProps?.onFinish?.(formRef.current?.getFieldsValue())
        )
          setOpen(false);
      })
      .catch(() => {})
      .finally(() => setConfirmLoading(false));
  };

  return (
    <>
      {trigger}
      <Modal
        title={props.title}
        open={open}
        onCancel={() => setOpen(false)}
        destroyOnClose={true}
        width={props.width}
        confirmLoading={confirmLoading}
        onOk={ok}
        okButtonProps={{ disabled: props.changedSubmit ? !changed : false }}
        maskClosable={false}
      >
        <BetaSchemaForm
          formRef={formRef}
          submitter={false}
          preserve={false}
          onValuesChange={(_: any, all: Record<string, any>) => {
            if (props.schemaProps.initialValues)
              for (let k in all) {
                // possible object
                if (!isEqual(props.schemaProps.initialValues[k], all[k])) {
                  setChanged(true);
                  return;
                }
              }
            setChanged(false);
          }}
          {...props.schemaProps}
        />
      </Modal>
    </>
  );
};
export default ModalSchema;
