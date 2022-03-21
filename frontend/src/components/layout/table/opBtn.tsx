import { checkPerm, UserPerm } from '@/utils';
import { BetaSchemaForm } from '@ant-design/pro-form';
import { FormSchema } from '@ant-design/pro-form/lib/components/SchemaForm';
import { ActionType } from '@ant-design/pro-table';
import { ReactElement } from 'react';
import { useAccess } from 'umi';

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
  data?: Record<string, any>,
) => {
  if (await func(data ? data : {})) {
    tableRef?.current?.reloadAndRest?.();
    return true;
  }
  return false;
};

const TableOp: React.FC<TableOpProps & FormSchema> = (props) => {
  const access = useAccess();
  const { forceRollback, perm, permName, tableRef, rollback, finish, ...rest } =
    props;
  if (forceRollback || (perm && permName && !checkPerm(access, permName, perm)))
    return rollback ? rollback : <></>;
  else
    return (
      <BetaSchemaForm
        layoutType="ModalForm"
        layout="horizontal"
        labelCol={{ span: 6 }}
        autoFocusFirstInput
        modalProps={{
          destroyOnClose: true,
        }}
        onFinish={(data) => onFinish(finish, tableRef, data)}
        {...rest}
      />
    );
};
export default TableOp;
