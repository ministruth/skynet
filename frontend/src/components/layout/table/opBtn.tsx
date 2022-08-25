import { checkPerm, UserPerm } from '@/utils';
import { BetaSchemaForm, ProFormInstance } from '@ant-design/pro-form';
import { FormSchema } from '@ant-design/pro-form/lib/components/SchemaForm';
import { ActionType } from '@ant-design/pro-table';
import { ReactElement, useEffect, useRef } from 'react';
import { useAccess, useModel } from 'umi';

export type TableOpProps = {
  forceRollback?: boolean;
  perm?: UserPerm;
  permName?: string;
  rollback?: ReactElement<any, any>;
  finish: (values: Record<string, any>) => Promise<boolean>;
  tableRef?: React.MutableRefObject<ActionType | undefined>;
};

const onFinish = async (
  func: (values: Record<string, any>) => Promise<boolean>,
  tableRef?: React.MutableRefObject<ActionType | undefined>,
  formRef?: React.MutableRefObject<ProFormInstance<any> | undefined>,
  data?: Record<string, any>,
) => {
  if (await func(data ? data : {})) {
    tableRef?.current?.reloadAndRest?.();
    return true;
  }
  return false;
};

const TableOp: React.FC<TableOpProps & FormSchema> = (props) => {
  const { initialState } = useModel('@@initialState');
  const access = useAccess();
  const { forceRollback, perm, permName, tableRef, rollback, finish, ...rest } =
    props;
  const formRef = useRef<ProFormInstance>();
  useEffect(() => {
    formRef.current?.setFieldsValue(props.initialValues);
  });
  if (
    forceRollback ||
    (perm &&
      permName &&
      !checkPerm(initialState?.signin, access, permName, perm))
  )
    return rollback ? rollback : <></>;
  else
    return (
      <BetaSchemaForm
        formRef={formRef}
        layoutType="ModalForm"
        layout="horizontal"
        labelCol={{ span: 6 }}
        autoFocusFirstInput
        preserve={false}
        modalProps={{
          forceRender: true,
          onCancel: (e) => {
            formRef.current?.resetFields();
          },
          destroyOnClose: true,
        }}
        onFinish={(data) => onFinish(finish, tableRef, formRef, data)}
        {...rest}
      />
    );
};
export default TableOp;
